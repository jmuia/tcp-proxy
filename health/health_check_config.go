package health

import "time"

type HealthCheckConfig struct {
	Timeout            time.Duration
	Interval           time.Duration
	UnhealthyThreshold int
	HealthyThreshold   int
}
