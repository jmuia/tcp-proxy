package health

type HealthCheck interface {
	Check() error
}
