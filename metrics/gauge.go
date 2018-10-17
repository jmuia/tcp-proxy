package metrics

type Gauge interface {
	Value() interface{}
}

type Uint64Gauge struct {
	measure func() uint64
}

func NewUint64Gauge(measure func() uint64) *Uint64Gauge {
	return &Uint64Gauge{measure}
}

func (g *Uint64Gauge) Value() interface{} {
	return g.measure()
}
