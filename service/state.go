package service

type State uint32

const (
	HEALTHY   State = 1
	UNHEALTHY State = 2
)

func (s State) String() string {
	strings := [...]string{"HEALTHY", "UNHEALTHY"}
	switch s {
	case HEALTHY, UNHEALTHY:
		return strings[s-1]
	default:
		return "UNKNOWN"
	}
}
