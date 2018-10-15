package loadbalancer

import (
	"math/rand"
	"net"
	"sync"

	logger "github.com/jmuia/tcp-proxy/logging"
	"github.com/jmuia/tcp-proxy/service"
)

type Random struct {
	lock    sync.RWMutex
	srvlist []*service.Service
	srvmap  map[string]int
}

func NewRandom() *Random {
	return &Random{
		lock:    sync.RWMutex{},
		srvlist: make([]*service.Service, 0),
		srvmap:  make(map[string]int),
	}
}

func (lb *Random) UpdateService(s *service.Service) {
	lb.lock.Lock()
	defer lb.lock.Unlock()

	idx, exists := lb.srvmap[s.Addr()]

	switch s.State() {
	case service.UNHEALTHY:
		if exists {
			logger.Infof("loadbalancer: Removed %s as %s", s.Addr(), s.State().String())
			lb.remove(idx)
			delete(lb.srvmap, s.Addr())
		}
	case service.HEALTHY:
		if !exists {
			logger.Infof("loadbalancer: Added %s as %s", s.Addr(), s.State().String())
			lb.srvlist = append(lb.srvlist, s)
			lb.srvmap[s.Addr()] = len(lb.srvlist) - 1
		}
	}
}

func (lb *Random) NextService(c net.Conn) *service.Service {
	lb.lock.RLock()
	defer lb.lock.RUnlock()
	return lb.srvlist[rand.Intn(len(lb.srvlist))]
}

func (lb *Random) remove(index int) {
	lb.srvlist[index] = lb.srvlist[len(lb.srvlist)-1]
	lb.srvlist[len(lb.srvlist)-1] = nil
	lb.srvlist = lb.srvlist[:len(lb.srvlist)-1]
}
