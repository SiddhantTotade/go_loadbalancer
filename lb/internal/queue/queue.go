package queue

import (
	"net/http"
)

type Request struct {
	W    http.ResponseWriter
	R    *http.Request
	Done chan struct{}
}

type RequestQueue struct {
	queue chan *Request
}

func NewRequestQueue(maxQueueSize int) *RequestQueue {
	return &RequestQueue{
		queue: make(chan *Request, maxQueueSize),
	}
}

func (rq *RequestQueue) Enqueue(req *Request) bool {
	select {
	case rq.queue <- req:
		return true
	default:
		return false
	}
}

func (rq *RequestQueue) StartWorkers(workerCount int, handler func(r *Request)) {
	for i := 0; i < workerCount; i++ {
		go func() {
			for req := range rq.queue {
				handler(req)
				if req.Done != nil {
					close(req.Done)
				}
			}
		}()
	}
}
