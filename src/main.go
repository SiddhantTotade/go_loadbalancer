package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Server interface {
	Address() string
	IsAlive() bool
	Serve(rw http.ResponseWriter, r *http.Request)
}

type SimpleServer struct {
	addr  string
	proxy *httputil.ReverseProxy
}

func newSimpleServer(addr string) *SimpleServer {
	serverUrl, err := url.Parse(addr)

	handleErr(err)

	return &SimpleServer{
		addr:  addr,
		proxy: httputil.NewSingleHostReverseProxy(serverUrl),
	}
}

type LoadBalancer struct {
	port            string
	roundRobinCount int
	servers         []Server
}

func NewLoadBalancer(port string, servers []Server) *LoadBalancer {
	return &LoadBalancer{
		port:            port,
		roundRobinCount: 0,
		servers:         servers,
	}
}

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}

func (s *SimpleServer) Address() string { return s.addr }

func (s *SimpleServer) IsAlive() bool {
	resp, err := http.Get(s.addr)

	if err != nil {
		return false
	}

	return resp.StatusCode < 500
}

func (s *SimpleServer) Serve(rw http.ResponseWriter, req *http.Request) {
	s.proxy.ServeHTTP(rw, req)
}

func (lb *LoadBalancer) getNextAvailableServer() Server {
	for {

		server := lb.servers[lb.roundRobinCount%len(lb.servers)]
		lb.roundRobinCount++

		if server.IsAlive() {
			return server
		}

		fmt.Printf("server %s is DOWN, trying next...\n", server.Address())
	}
}

func (lb *LoadBalancer) serverProxy(rw http.ResponseWriter, req *http.Request) {
	targetServer := lb.getNextAvailableServer()
	fmt.Printf("forwarding request to address %q\n", targetServer.Address())
	targetServer.Serve(rw, req)
}

func main() {
	servers := []Server{
		newSimpleServer("https://www.facebook.com"),
		newSimpleServer("https://www.bing.com"),
		// newSimpleServer("https://www.duckduckgo.com"),
		newSimpleServer("https://www.netflix.com"),
	}

	lb := NewLoadBalancer("8000", servers)
	handleRedirect := func(rw http.ResponseWriter, req *http.Request) {
		lb.serverProxy(rw, req)
	}
	http.HandleFunc("/", handleRedirect)

	fmt.Printf("serving request at 'localhost:%s'\n", lb.port)

	http.ListenAndServe(":"+lb.port, nil)
}
