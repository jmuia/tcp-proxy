package loadbalancer

import (
	"math/rand"
	"net"

	"github.com/jmuia/tcp-proxy/backend"
)

type P2C struct {
	random *Random
}

func NewP2C() *P2C {
	return &P2C{NewRandom()}
}

func (lb *P2C) UpdateBackend(s *backend.Backend) {
	lb.random.UpdateBackend(s)
}

func (lb *P2C) NextBackend(c net.Conn) *backend.Backend {
	lb.random.lock.RLock()
	defer lb.random.lock.RUnlock()

	if len(lb.random.backendList) <= 1 {
		return lb.random.backendList[0]
	}

	for {
		choice1 := rand.Intn(len(lb.random.backendList))
		choice2 := rand.Intn(len(lb.random.backendList))

		if choice1 == choice2 {
			continue
		}

		srv1 := lb.random.backendList[choice1]
		srv2 := lb.random.backendList[choice2]

		if srv1.ActiveConns() > srv2.ActiveConns() {
			return srv2
		}
		return srv1
	}
}
