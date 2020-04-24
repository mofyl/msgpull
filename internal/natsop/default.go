package natsop

import (
	"time"

	"github.com/nats-io/nats.go"
)

var (
	// _clusterID = "test-cluster"
	// _clientID  = "test-client"
	// _defaultConn   *nats.Conn
	// _defaultSubConfig *SubConfig

	_defaultConfig *Config
	_defaultNatsOp *NatsOp
)

func init() {
	_defaultConfig = defaultConfig()
	_defaultNatsOp = newNatsOp()
}

func defaultConfig() *Config {
	return &Config{
		Url:           nats.DefaultURL,
		Timeout:       nats.DefaultTimeout,
		PingInerval:   nats.DefaultPingInterval,
		MaxPingOut:    nats.DefaultMaxPingOut,
		ReConnectWait: nats.DefaultReconnectWait,
		MaxReConnect:  nats.DefaultMaxReconnect,
		SubBufferSize: 8 * 1024 * 1024,
		MsgTimeout:    5,
	}
}

func Send(topic string, reqMsgID int64, timeout int64, data []byte) error {
	return _defaultNatsOp.Send(topic, reqMsgID, timeout, data)
}
func SubChan(topic string, c chan *MsgInfo) (*nats.Subscription, error) {
	return _defaultNatsOp.SubMsg(topic, c)
}

func Call(topic string, reqMsgID int64, data []byte, timeout time.Duration) (*MsgInfo, error) {
	return _defaultNatsOp.Call(topic, reqMsgID, data, timeout)
}

func Connect(cfg *Config) {
	if cfg == nil {
		cfg = _defaultConfig
	}
	_defaultNatsOp.Connect(cfg)
}
func Close() error {
	return _defaultNatsOp.Close()
}
