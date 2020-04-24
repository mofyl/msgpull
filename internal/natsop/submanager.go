package natsop

import (
	"github.com/nats-io/nats.go"
)

type SubInfo struct {
	Suber *nats.Subscription
	C     chan *MsgInfo
}

type SubManager struct {
	l *LockMap
}

func NewSubManager() *SubManager {
	return &SubManager{
		l: NewLockMap(),
	}
}

func (m *SubManager) Load(topic string) *SubInfo {
	if v := m.l.GetValue(topic); v != nil {
		if sub, ok := v.(*SubInfo); ok {
			return sub
		}

	}
	return nil
}

func (m *SubManager) Store(topic string, suber *SubInfo) {
	v := m.Load(topic)

	if v == nil {
		m.l.SetValue(topic, suber)
	}
}

func (m *SubManager) Delete(topic string) {
	m.l.DeleteValue(topic)
}

func (m *SubManager) Close() {
	m.l.Range(func(key, value interface{}) bool {
		if v, ok := value.(*nats.Subscription); ok {
			v.Unsubscribe()
			m.l.DeleteValue(key)
		}
		return true
	})
}
