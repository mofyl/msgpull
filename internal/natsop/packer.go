package natsop

import (
	"errors"
	"sync"

	jsoniter "github.com/json-iterator/go"
)

type MsgInfo struct {
	MsgId      int64
	Topic      string
	Queue      string
	ReplyTopic string
	ReqMsgId   int64
	Data       []byte
	Timeout    int64
	NotifyChan chan *MsgInfo `json:"-"`
}

type Cancel func()

type Packer struct {
	json_iterator jsoniter.API
	msgInfoPool   sync.Pool
}

func NewPacker() *Packer {
	return &Packer{
		json_iterator: jsoniter.ConfigCompatibleWithStandardLibrary,
		msgInfoPool:   sync.Pool{New: func() interface{} { return &MsgInfo{} }},
	}
}
func (p *Packer) Encode(m *MsgInfo) ([]byte, Cancel, error) {
	stream := p.json_iterator.BorrowStream(nil)
	stream.WriteVal(m)
	if stream.Error != nil {
		return nil, nil, stream.Error
	}
	result := stream.Buffer()

	f := func() {

		p.json_iterator.ReturnStream(stream)
	}
	return result, f, nil

}

func (p *Packer) Decode(data []byte) (*MsgInfo, Cancel, error) {
	// m := &MsgInfo{}
	m, ok := p.msgInfoPool.Get().(*MsgInfo)

	if !ok {
		return nil, nil, errors.New("Get Pool Fail")
	}
	err := p.json_iterator.Unmarshal(data, m)

	if err != nil {
		return nil, nil, err
	}

	f := func() {
		if m != nil {
			p.msgInfoPool.Put(m)
		}
	}

	return m, f, nil
}
