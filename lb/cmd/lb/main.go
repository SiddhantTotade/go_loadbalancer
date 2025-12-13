package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"go_loadbalancer/lb/internal/backend"
	"go_loadbalancer/lb/internal/handler"
	"go_loadbalancer/lb/internal/health"
	"go_loadbalancer/lb/internal/queue"
	"go_loadbalancer/lb/internal/ratelimit"
	"go_loadbalancer/lb/internal/registry"
	"go_loadbalancer/lb/internal/strategy/weightedroundrobin"
)

func main() {
	reg := registry.NewRegistry()

	b1, _ := backend.CreateNewBackend("http://localhost:8081", 3*time.Second)
	b2, _ := backend.CreateNewBackend("http://localhost:8082", 3*time.Second)

	reg.Add(b1)
	reg.Add(b2)

	strat := weightedroundrobin.NewWeightedRoundRobin(map[*backend.Backend]int{
		b1: 3,
		b2: 1,
	})

	q := queue.NewRequestQueue(100)
	limiter := ratelimit.NewTokenBucket(100, 50)

	h := handler.NewHandler(reg, strat, 3, q)
	h.Queue = q
	h.GlobalLimiter = limiter

	q.StartWorkers(5, func(req *queue.Request) {
		h.ServeBackend(req.W, req.R)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hc := health.NewHealthChecker(reg, 5*time.Second, 3, 2)
	hc.Start(ctx)

	log.Println("Load Balancer running on :8080")

	if err := http.ListenAndServe(":8080", h); err != nil {
		log.Fatal(err)
	}
}
