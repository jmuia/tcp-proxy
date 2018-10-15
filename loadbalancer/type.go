package loadbalancer

import (
	"fmt"

	"github.com/pkg/errors"
)

type Type uint32

const (
	RANDOM_TYPE Type = 1
	P2C_TYPE    Type = 2
)

func (t Type) String() string {
	strings := [...]string{"RANDOM", "P2C"}
	switch t {
	case RANDOM_TYPE, P2C_TYPE:
		return strings[t-1]
	default:
		return "UNKNOWN"
	}
}

func ParseType(s string) (Type, error) {
	switch s {
	case "RANDOM":
		return RANDOM_TYPE, nil
	case "P2C":
		return P2C_TYPE, nil
	default:
		return 0, errors.New(fmt.Sprintf("invalid load balancer type %s", s))
	}
}
