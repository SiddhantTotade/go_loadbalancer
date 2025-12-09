package circuitbreaker

import (
	"sync/atomic"
	"time"

	"../backend"
)

type State int32

const (
	Closed State = iota
	Open
	HalfOpen
)

type CircuitBreaker struct {
	FailureThreshold int32
	ResetTimeout     time.Duration
	state            atomic.Int32
	lastFailure      atomic.Int64
}

func NewCircuitBreaker(threshold int32, resetTimeout time.Duration) *CircuitBreaker {
	cb := &CircuitBreaker{
		FailureThreshold: threshold,
		ResetTimeout:     resetTimeout,
	}

	cb.state.Store(int32(Closed))

	return cb
}

func (cb *CircuitBreaker) setState(s State) {
	cb.state.Store(int32(s))
}

func (cb *CircuitBreaker) BeforeRequest(b *backend.Backend) bool {
	switch cb.State() {
	case Open:
		if time.Now().UnixNano()-cb.lastFailure.Load() > cb.ResetTimeout.Nanoseconds() {
			cb.setState(HalfOpen)
			return true
		}
		return false

	case HalfOpen:
		return false
	case Closed:
		return true
	}

	return true
}

func (cb *CircuitBreaker) AfterRequestSuccess(b *backend.Backend) {
	if cb.State() == HalfOpen {
		cb.setState(Closed)
		b.ResetFailCount()
	}
	b.IncrementSuccessCount()
}

func (cb *CircuitBreaker) AfterRequestFailure(b *backend.Backend) {
	b.IncrementFailCount()
	cb.lastFailure.Store(time.Now().UnixNano())

	if b.FailCount() >= cb.FailureThreshold {
		cb.setState(Open)
		b.MarkDead()
	}
}

func (cb *CircuitBreaker) State() State {
	return State(cb.state.Load())
}
