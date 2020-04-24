package natsop

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
	uuid "github.com/satori/go.uuid"
)

var (
	reqTopicLen = 20
)

const (
	REQTOPIC = "REQTOPIC"
)

type NatsOp struct {
	sub    chan *nats.Msg
	pub    chan *MsgInfo
	packer *Packer

	NatConfig  *Config
	Conn       *nats.Conn
	SubManager *SubManager

	tokenMap map[int64]chan *MsgInfo

	timerPool *sync.Pool

	replyTopic string
	isSubReply uint32 // 1表示未sub 2表示已经sub
	startMsgID int64

	ctx    context.Context
	cancel context.CancelFunc
}

func newNatsOp() *NatsOp {

	reqTopic, _ := getReqTopic()
	ctx, cancel := context.WithCancel(context.Background())
	fmt.Println(reqTopic)
	return &NatsOp{
		ctx:        ctx,
		cancel:     cancel,
		sub:        make(chan *nats.Msg, 1024),
		pub:        make(chan *MsgInfo, 1024),
		packer:     NewPacker(),
		SubManager: NewSubManager(),
		tokenMap:   make(map[int64]chan *MsgInfo),
		timerPool:  &sync.Pool{},
		replyTopic: reqTopic,
		isSubReply: 1,
	}
}

func (n *NatsOp) errorHandler(nc *nats.Conn, sub *nats.Subscription, err error) {
	cliId, err := nc.GetClientID()
	if err != nil {
		fmt.Println("errorHandler cliID is 0 err is  ", cliId, err)
	} else {

		fmt.Println("errorHandler come ")
	}

}

func (n *NatsOp) closedHandler(nc *nats.Conn) {

	cliId, err := nc.GetClientID()
	if err != nil {
		fmt.Println("closedHandler cliID is 0 err is  ", cliId, err)
	} else {
		fmt.Println("closedHandler come ")
	}
	n.cancel()
}

func (n *NatsOp) disconnectErrHandler(nc *nats.Conn, err error) {
	cliId, _ := nc.GetClientID()
	fmt.Println("disconnectErrHandler ", cliId, err)
	n.cancel()
}

func (n *NatsOp) reConnectedHandler(nc *nats.Conn) {
	cliId, _ := nc.GetClientID()
	fmt.Println("disconnectErrHandler ", cliId)
}

func (n *NatsOp) Connect(conf *Config) error {

	conn, err := nats.Connect(
		conf.Url,
		// nats.MaxReconnects(cfg.MaxReConnect),
		nats.MaxReconnects(0),
		nats.ReconnectWait(conf.ReConnectWait),
		nats.PingInterval(conf.PingInerval),
		nats.MaxPingsOutstanding(conf.MaxPingOut),
		nats.ReconnectHandler(n.reConnectedHandler),
		nats.ClosedHandler(n.closedHandler),
	)

	if err != nil {
		return err
	}

	n.Conn = conn
	n.NatConfig = conf
	// c.wg.Add(1)
	go n.loop()
	// go c.handler()

	return nil
}

func (n *NatsOp) loop() {
	for {
		select {
		case <-n.ctx.Done():
			return
		case m := <-n.pub:
			n.startMsgID++
			m.MsgId = n.startMsgID
			if m.NotifyChan != nil {
				if m.ReqMsgId == 0 {
					m.ReqMsgId = m.MsgId
				}
				n.tokenMap[m.MsgId] = m.NotifyChan
			}

			// fmt.Println("send  ", m)
			sendData, cancel, err := n.packer.Encode(m)
			if err != nil {
				fmt.Println("nats loop err encode ", err)
				continue
			}
			err = n.Conn.Publish(m.Topic, sendData)
			cancel()
			if err != nil {
				fmt.Println("nats loop publish err ", err, m.Topic)
				continue
			}

		case m, ok := <-n.sub:
			if !ok || m == nil {
				continue
			}
			mInfo, cancel, err := n.packer.Decode(m.Data)
			if err != nil {
				fmt.Println("sub Decode ", err)
				continue
			}

			fmt.Println("sub ", mInfo.Topic, string(mInfo.Data), mInfo.ReqMsgId)
			if n.replyTopic == mInfo.Topic {
				now := time.Now().Unix()
				if now > mInfo.Timeout {
					continue
				}
				c, ok := n.tokenMap[mInfo.ReqMsgId]

				if ok {

					if IsChanClosed(c) {
						continue
					}

					select {
					case c <- mInfo:
					default:
					}
				}
				delete(n.tokenMap, mInfo.ReqMsgId)
				cancel()
			} else {

				subInfo := n.SubManager.Load(mInfo.Topic)

				if subInfo != nil {
					if subInfo.C == nil {
						continue
					}
					if IsChanClosed(subInfo.C) {
						continue
					}
				}

				select {
				case subInfo.C <- mInfo:
				default:
				}
				cancel()

			}
		}
	}
}

func (n *NatsOp) Send(topic string, reqMsgID int64, timeout int64, data []byte) error {

	msgSend := &MsgInfo{
		Topic: topic,
		Data:  data,
	}

	if reqMsgID != 0 {
		msgSend.ReqMsgId = reqMsgID
		msgSend.Timeout = timeout
	} else {
		msgSend.Timeout = time.Now().Add(time.Duration(n.NatConfig.MsgTimeout) * time.Second).Unix()
	}
	return n.publicMsg(msgSend)

}

func (n *NatsOp) publicMsg(msgSend *MsgInfo) error {
	select {
	case n.pub <- msgSend:
	default:
	}
	return nil
}
func (n *NatsOp) Call(topic string, reqMsgID int64, data []byte, timeout time.Duration) (*MsgInfo, error) {
	// create new topic
	c := make(chan *MsgInfo)
	if atomic.LoadUint32(&n.isSubReply) == 1 {

		subInfo, err := n.getSetTopic(n.replyTopic, nil)
		if err != nil {
			return nil, err
		}

		n.SubManager.Store(topic, subInfo)
		atomic.StoreUint32(&n.isSubReply, 2)
	}
	msgSend := &MsgInfo{
		Topic:      topic,
		Queue:      topic,
		ReplyTopic: n.replyTopic,
		Data:       data,
		Timeout:    time.Now().Add(timeout).Unix(),
		ReqMsgId:   reqMsgID,
		NotifyChan: c,
	}

	err := n.publicMsg(msgSend)

	if err != nil {
		return nil, err
	}
	timer := n.GetTimer(timeout)
	defer n.timerPool.Put(timer)

	var reqMsg *MsgInfo
	select {
	case <-timer.C:
		err = errors.New(TIME_OUT)
	case reqMsg = <-c:
	}

	close(c)
	return reqMsg, err
}
func (n *NatsOp) Close() error {

	n.cancel()

	if n.Conn != nil {
		n.Conn.Close()
	}
	n.SubManager.Close()
	for k, v := range n.tokenMap {
		delete(n.tokenMap, k)
		close(v)
	}
	close(n.sub)
	close(n.pub)
	return nil
}

func (n *NatsOp) SubMsg(topic string, c chan *MsgInfo) (*nats.Subscription, error) {
	subInfo, err := n.getSetTopic(topic, c)
	if err != nil {
		return nil, err
	}
	suber := subInfo.Suber
	subInfo.Suber = nil
	n.SubManager.Store(topic, subInfo)
	return suber, nil
}

func (n *NatsOp) getSetTopic(topic string, c chan *MsgInfo) (*SubInfo, error) {

	subInfo := n.SubManager.Load(topic)
	if subInfo == nil {

		suber, err := n.Conn.QueueSubscribeSyncWithChan(topic, topic, n.sub)
		if err != nil {
			return nil, err
		}
		subInfo = &SubInfo{
			Suber: suber,
		}

		if c != nil {
			subInfo.C = c
		}

		// n.SubManager.Store(topic, subInfo)

	}
	return subInfo, nil
}

func (n *NatsOp) GetTimer(timeout time.Duration) *time.Timer {
	timerInterface := n.timerPool.Get()

	timer, _ := timerInterface.(*time.Timer)

	if timer == nil {
		timer = time.NewTimer(timeout)
	} else {
		timer.Reset(timeout)
	}

	return timer
}

func getReqTopic() (string, error) {

	uuID, err := GetTopic("")

	if err == nil {
		b := strings.Builder{}
		b.WriteString(REQTOPIC)
		b.WriteString(".")
		b.WriteString(uuID[:reqTopicLen])
		return b.String(), nil
	}

	return "", err
}

func GetTopic(extendInfo string) (string, error) {
	uid := fmt.Sprintf("%d%s", time.Now().UnixNano(), extendInfo)

	h := md5.Sum([]byte(uid))
	hs := hex.EncodeToString(h[:])

	u, err := uuid.FromString(hs)

	if err != nil {
		return "", err
	}

	return u.String(), err

}
