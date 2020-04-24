package natsop

import (
	"sync"
)

type LockMap struct {
	m *sync.Map
}

func NewLockMap() *LockMap {
	return &LockMap{
		m: &sync.Map{},
	}

}

func (l *LockMap) GetValue(key interface{}) interface{} {
	if kValue, ok := l.m.Load(key); ok {
		return kValue
	}
	return nil
}

func (l *LockMap) SetValue(key, v interface{}) {
	l.m.Store(key, v)
}

func (l *LockMap) DeleteValue(key interface{}) {
	l.m.Delete(key)
}

func (l *LockMap) Range(f func(key, value interface{}) bool) {
	l.m.Range(f)
}
