package health

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go_loadbalancer/lb/internal/backend"
	"go_loadbalancer/lb/internal/registry"
)

type HealthChecker struct {
	registry         *registry.BackendRegistry
	interval         time.Duration
	failThreshold    int
	successThreshold int
	client           *http.Client
}

func NewHealthChecker(r *registry.BackendRegistry, interval time.Duration, failThreshold int, successThreshold int) *HealthChecker {
	return &HealthChecker{
		registry:         r,
		interval:         interval,
		failThreshold:    failThreshold,
		successThreshold: successThreshold,
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

func (hc *HealthChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(hc.interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				hc.check()
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

func (hc *HealthChecker) check() {
	backends := hc.registry.List()

	for _, b := range backends {
		go func(backend *backend.Backend) {
			alive := hc.ping(backend)

			if alive {
				backend.ResetFailCount()
				backend.IncrementSuccessCount()

				if backend.SuccessCount() >= int32(hc.successThreshold) {
					backend.MarkAlive()
				}
			} else {
				backend.ResetSuccessCount()
				backend.IncrementFailCount()

				if backend.FailCount() >= int32(hc.failThreshold) {
					backend.MarkDead()
				}
			}
		}(b)
	}
}

func (hc *HealthChecker) ping(b *backend.Backend) bool {
	req, _ := http.NewRequest("GET", b.URL.String(), nil)
	resp, err := hc.client.Do(req)

	fmt.Println("Pinging", b.URL.String(), "Alive?", err == nil)

	if err != nil {
		return false
	}

	defer resp.Body.Close()

	return resp.StatusCode < 500
}
