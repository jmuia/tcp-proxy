package loadbalancer

import (
	"math/rand"
	"net"

	"github.com/jmuia/tcp-proxy/service"
)

type P2C struct {
	random *Random
}

func NewP2C() *P2C {
	return &P2C{NewRandom()}
}

func (lb *P2C) UpdateService(s *service.Service) {
	lb.random.UpdateService(s)
}

func (lb *P2C) NextService(c net.Conn) *service.Service {
	lb.random.lock.RLock()
	defer lb.random.lock.RUnlock()

	if len(lb.random.srvlist) <= 1 {
		return lb.random.srvlist[0]
	}

	for {
		choice1 := rand.Intn(len(lb.random.srvlist))
		choice2 := rand.Intn(len(lb.random.srvlist))

		if choice1 == choice2 {
			continue
		}

		srv1 := lb.random.srvlist[choice1]
		srv2 := lb.random.srvlist[choice2]

		if srv1.ActiveConns() > srv2.ActiveConns() {
			return srv2
		}
		return srv1
	}
}
