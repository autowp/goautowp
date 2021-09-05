package auth

import (
	"sync"
	"time"
)

// State State
type State struct {
	UserID      int64
	Language    string
	Service     ExternalService
	RedirectURI string
}

// StateMapItem StateMapItem
type StateMapItem struct {
	State      State
	lastAccess time.Time
}

// StateMap StateMap
type StateMap struct {
	m map[string]*StateMapItem
	l sync.Mutex
}

// NewStateMap NewStateMap
func NewStateMap(maxTTL time.Duration) (m *StateMap) {
	m = &StateMap{m: make(map[string]*StateMapItem)}
	go func() {
		for now := range time.Tick(time.Minute) {
			m.l.Lock()
			for k, v := range m.m {
				expiresAt := v.lastAccess.Add(maxTTL)
				if now.After(expiresAt) {
					delete(m.m, k)
				}
			}
			m.l.Unlock()
		}
	}()
	return
}

// Len Len
func (m *StateMap) Len() int {
	return len(m.m)
}

// Put Put
func (m *StateMap) Put(k string, v State) {
	m.l.Lock()
	it, ok := m.m[k]
	if !ok {
		it = &StateMapItem{State: v}
		m.m[k] = it
	}
	it.lastAccess = time.Now()
	m.l.Unlock()
}

// Get Get
func (m *StateMap) Get(k string) (v *State) {
	m.l.Lock()
	if it, ok := m.m[k]; ok {
		v = &it.State
		it.lastAccess = time.Now()
	}
	m.l.Unlock()
	return
}

// Delete Delete
func (m *StateMap) Delete(k string) {
	m.l.Lock()
	delete(m.m, k)
	m.l.Unlock()
}
