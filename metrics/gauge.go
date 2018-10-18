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

type StringGauge struct {
	measure func() string
}

func NewStringGauge(measure func() string) *StringGauge {
	return &StringGauge{measure}
}

func (g *StringGauge) Value() interface{} {
	return g.measure()
}
