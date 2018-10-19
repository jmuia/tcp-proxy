package loadbalancer

import (
	"math/rand"
	"net"

	"github.com/jmuia/tcp-proxy/backend"
	"github.com/pkg/errors"
)

/**
 * P2C (power of two choices) load balancing.
 * https://brooker.co.za/blog/2012/01/17/two-random.html
 *
 * Note: the benefits of this approach are not that useful
 * in this project because we're not operating on cached
 * connection count data. It still outperforms random,
 * however.
 */
type P2C struct {
	random *Random
}

func NewP2C() *P2C {
	return &P2C{NewRandom()}
}

func (lb *P2C) UpdateBackend(s *backend.Backend) {
	lb.random.UpdateBackend(s)
}

func (lb *P2C) NextBackend(c net.Conn) (*backend.Backend, error) {
	lb.random.lock.RLock()
	defer lb.random.lock.RUnlock()

	switch len(lb.random.backendList) {
	case 0:
		return nil, errors.New("loadbalancer: no healthy backends available")
	case 1:
		return lb.random.backendList[0], nil
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
			return srv2, nil
		}
		return srv1, nil
	}
}
