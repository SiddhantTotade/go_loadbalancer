package handler

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"go_loadbalancer/lb/internal/queue"
	"go_loadbalancer/lb/internal/ratelimit"
	"go_loadbalancer/lb/internal/registry"
	"go_loadbalancer/lb/internal/retry"
	"go_loadbalancer/lb/internal/strategy"
)

type LBHandler struct {
	Registry      *registry.BackendRegistry
	Strategy      strategy.Strategy
	MaxRetries    int
	GlobalLimiter *ratelimit.TokenBucket
	Queue         *queue.RequestQueue
}

func NewHandler(r *registry.BackendRegistry, s strategy.Strategy, maxRetries int, q *queue.RequestQueue) *LBHandler {
	h := &LBHandler{
		Registry:   r,
		Strategy:   s,
		MaxRetries: maxRetries,
		Queue:      q,
	}

	return h
}

func (h *LBHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	reqWrap := &queue.Request{
		W:    w,
		R:    req,
		Done: make(chan struct{}),
	}

	if h.GlobalLimiter != nil && !h.GlobalLimiter.Allow() {
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return
	}

	if !h.Queue.Enqueue(reqWrap) {
		http.Error(w, "Server busy. Too many requests.", http.StatusServiceUnavailable)
		return
	}

	<-reqWrap.Done

}

func (h *LBHandler) processRequest(w http.ResponseWriter, req *http.Request) {
	var bodyBuf []byte
	if req.Body != nil {
		if b, err := io.ReadAll(req.Body); err == nil {
			bodyBuf = b
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBuf))
	}

	var lastErr error

	for attempt := 0; attempt < h.MaxRetries; attempt++ {

		alive := h.Registry.AliveBackends()
		if len(alive) == 0 {
			http.Error(w, "no backend available", http.StatusServiceUnavailable)
			return
		}

		backend := h.Strategy.Next(alive)
		if backend == nil {
			http.Error(w, "no backend selected", http.StatusServiceUnavailable)
			return
		}

		if backend.CB != nil {
			if ok := backend.CB.BeforeRequest(); !ok {
				log.Printf("circuit OPEN for %s → skipping", backend.URL)
				continue
			}
		}

		if bodyBuf != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBuf))
		}

		rec := retry.NewResponseRecorder()
		backend.Proxy.ServeHTTP(rec, req)

		if rec.Status < 500 {
			backend.RecordSuccess()

			for k, vv := range rec.HeaderMap {
				for _, v := range vv {
					w.Header().Add(k, v)
				}
			}

			w.WriteHeader(rec.Status)

			if rec.Body != nil {
				_, _ = io.Copy(w, rec.Body)
			}

			log.Printf(
				"backend SUCCESS: %s status=%d attempt=%d",
				backend.URL, rec.Status, attempt+1,
			)

			return
		}

		lastErr = fmt.Errorf("backend %s returned %d", backend.URL, rec.Status)
		backend.RecordFailure()

		log.Printf(
			"backend FAILURE: %s status=%d attempt=%d",
			backend.URL, rec.Status, attempt+1,
		)

		time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("all backend failed")
	}

	http.Error(w, lastErr.Error(), http.StatusBadGateway)
}

func (h *LBHandler) ServeBackend(w http.ResponseWriter, r *http.Request) {
	h.processRequest(w, r)
}
