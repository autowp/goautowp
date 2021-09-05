package store

import (
	"errors"
	"sync"

	"github.com/autowp/goautowp/auth/oauth2server"
)

// NewClientStore create client store
func NewClientStore() *ClientStore {
	return &ClientStore{
		data: make(map[string]oauth2server.ClientInfo),
	}
}

// ClientStore client information store
type ClientStore struct {
	sync.RWMutex
	data map[string]oauth2server.ClientInfo
}

// GetByID according to the ID for the client information
func (cs *ClientStore) GetByID(id string) (oauth2server.ClientInfo, error) {
	cs.RLock()
	defer cs.RUnlock()

	if c, ok := cs.data[id]; ok {
		return c, nil
	}
	return nil, errors.New("not found")
}

// Set set client information
func (cs *ClientStore) Set(id string, cli oauth2server.ClientInfo) (err error) {
	cs.Lock()
	defer cs.Unlock()

	cs.data[id] = cli
	return
}
