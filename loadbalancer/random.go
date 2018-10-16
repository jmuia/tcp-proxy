package loadbalancer

import (
	"math/rand"
	"net"
	"sync"

	"github.com/jmuia/tcp-proxy/backend"
	logger "github.com/jmuia/tcp-proxy/logging"
)

type Random struct {
	lock        sync.RWMutex
	backendList []*backend.Backend
	backendMap  map[string]int
}

func NewRandom() *Random {
	return &Random{
		lock:        sync.RWMutex{},
		backendList: make([]*backend.Backend, 0),
		backendMap:  make(map[string]int),
	}
}

func (lb *Random) UpdateBackend(s *backend.Backend) {
	lb.lock.Lock()
	defer lb.lock.Unlock()

	idx, exists := lb.backendMap[s.Addr()]

	switch s.State() {
	case backend.UNHEALTHY:
		if exists {
			logger.Infof("loadbalancer: Removed %s as %s", s.Addr(), s.State().String())
			lb.remove(idx)
			delete(lb.backendMap, s.Addr())
		}
	case backend.HEALTHY:
		if !exists {
			logger.Infof("loadbalancer: Added %s as %s", s.Addr(), s.State().String())
			lb.backendList = append(lb.backendList, s)
			lb.backendMap[s.Addr()] = len(lb.backendList) - 1
		}
	}
}

func (lb *Random) NextBackend(c net.Conn) *backend.Backend {
	lb.lock.RLock()
	defer lb.lock.RUnlock()
	// TODO: error if no healthy backends.
	return lb.backendList[rand.Intn(len(lb.backendList))]
}

func (lb *Random) remove(index int) {
	lb.backendList[index] = lb.backendList[len(lb.backendList)-1]
	lb.backendList[len(lb.backendList)-1] = nil
	lb.backendList = lb.backendList[:len(lb.backendList)-1]
}
