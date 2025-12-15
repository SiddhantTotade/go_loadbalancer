package gateway

import (
	"go_loadbalancer/lb/internal/handler"
	"net/http"
)

type Gateway struct {
	Routes  []*Route
	Handler *handler.LBHandler
}

func NewGateway(h *handler.LBHandler) *Gateway {
	return &Gateway{
		Routes:  make([]*Route, 0),
		Handler: h,
	}
}

func (g *Gateway) Register(r *Route) {
	g.Routes = append(g.Routes, r)
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, route := range g.Routes {
		if route.Match(r.URL.Path) {
			r.URL.Path = route.Rewrite(r.URL.Path)
			g.Handler.Registry = route.Registry
			g.Handler.Strategy = route.Strategy
			g.Handler.ServeBackend(w, r)
			return
		}
	}

	http.Error(w, "route not found", http.StatusNotFound)
}
