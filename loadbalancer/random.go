package loadbalancer

import (
	"math/rand"
	"net"
	"sync"

	"github.com/jmuia/tcp-proxy/backend"
	logger "github.com/jmuia/tcp-proxy/logging"
	"github.com/pkg/errors"
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

func (lb *Random) NextBackend(c net.Conn) (*backend.Backend, error) {
	lb.lock.RLock()
	defer lb.lock.RUnlock()
	switch len(lb.backendList) {
	case 0:
		return nil, errors.New("loadbalancer: no healthy backends available")
	case 1:
		return lb.backendList[0], nil
	default:
		return lb.backendList[rand.Intn(len(lb.backendList))], nil
	}
}

func (lb *Random) remove(index int) {
	lb.backendList[index] = lb.backendList[len(lb.backendList)-1]
	lb.backendList[len(lb.backendList)-1] = nil
	lb.backendList = lb.backendList[:len(lb.backendList)-1]
}
