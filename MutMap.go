package main

import "sync"

type MutMap struct {
	Map map[string]Channel
	Mut *sync.RWMutex
}

func (m *MutMap) Set(key string, value Channel) {
	m.Mut.Lock()
	m.Map[key] = value
	m.Mut.Unlock()
}

func (m *MutMap) Get(key string) Channel {
	m.Mut.RLock()
	defer m.Mut.RUnlock()
	return m.Map[key]
}
 
