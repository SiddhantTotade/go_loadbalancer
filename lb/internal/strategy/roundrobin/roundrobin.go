package roundrobin

import (
	"sync/atomic"

	"go_loadbalancer/lb/internal/strategy"

	"go_loadbalancer/lb/internal/backend"
)

type RoundRobin struct {
	counter atomic.Uint64
}

func New() strategy.Strategy {
	return &RoundRobin{}
}

func (rr *RoundRobin) Next(backends []*backend.Backend) *backend.Backend {
	if len(backends) == 0 {
		return nil
	}

	idx := (rr.counter.Add(1) - 1) % uint64(len(backends))
	return backends[int(idx)]
}
