func (r *Router) SelectBackend(req *http.Request) *backend.Backend
func (r *Router) ApplySticky(req, rw)
func (r *Router) RouteByPath(req)