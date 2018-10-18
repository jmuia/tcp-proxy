package loadbalancer

import (
	"testing"
)

func TestNoHealthyBackendsReturnsError(t *testing.T) {
	p2c := NewP2C()
	b, err := p2c.NextBackend(nil)
	if err == nil {
		t.Errorf("expected error when P2C loadbalancer has no healthy backends, got %v", b)
	}

	expected := "loadbalancer: no healthy backends available"
	if err.Error() != expected {
		t.Errorf("expected error '%s' when P2C loadbalancer has no healthy backends, got '%v'", expected, err.Error())
	}

	random := NewRandom()
	b, err = random.NextBackend(nil)
	if err == nil {
		t.Errorf("expected error when Random loadbalancer has no healthy backends, got %v", b)
	}

	expected = "loadbalancer: no healthy backends available"
	if err.Error() != expected {
		t.Errorf("expected error '%s' when Random loadbalancer has no healthy backends, got '%v'", expected, err.Error())
	}
}
