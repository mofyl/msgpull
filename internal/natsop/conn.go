package natsop

import (
	"time"

	"github.com/nats-io/nats.go"
)

type Config struct {
	Url string
	// Dial 最大等待时间
	Timeout time.Duration
	// 发送ping消息的间隔
	PingInerval time.Duration
	// 发送ping消息后 最大允许 MaxPingOut个没有pong消息的回复
	MaxPingOut    int
	ReConnectWait time.Duration
	MaxReConnect  int
	UserName      string
	Pwd           string

	SubBufferSize int64
	MsgTimeout    int64
}

type MsgHandler func([]byte, error)

type Conn interface {
	Connect(*Config) error
	Send(topic string, reqMsgID int64, timeout int64, data []byte) error
	Call(topic string, reqMsgID int64, data []byte, timeout time.Duration) (*MsgInfo, error)
	SubMsg(topic string, m *MsgInfo) (*nats.Subscription, error)
	Close() error
}
