package loadbalancer

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"fmt"
	"time"
)

type Server struct {
	URL   *url.URL
	Proxy *httputil.ReverseProxy
	Alive bool
}

type RoundRobin struct {
	servers []*Server
	counter uint64
}

func NewRoundRobin(targets []string) (*RoundRobin, error) {

	var servers []*Server

	for _, t := range targets {

		u, err := url.Parse(t)
		if err != nil {
			return nil, err
		}

		proxy := httputil.NewSingleHostReverseProxy(u)

		servers = append(servers, &Server{
			URL:   u,
			Proxy: proxy,
			Alive: true,
		})
	}

	return &RoundRobin{
		servers: servers,
	}, nil
}

func (rr *RoundRobin) NextServer() *Server {

	for {
		i := atomic.AddUint64(&rr.counter, 1) - 1
		index := i % uint64(len(rr.servers))

		server := rr.servers[index]

		if server.Alive {
			return server
		}
	}
}

func (rr *RoundRobin) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	server := rr.NextServer()

	server.Proxy.ServeHTTP(w, r)
}

func (rr *RoundRobin) StartHealthCheck() {

	go func() {

		for {
			for _, server := range rr.servers {

				resp, err := http.Get(server.URL.String() + "/health")

				if err != nil || resp.StatusCode != 200 {
					server.Alive = false
					fmt.Println("Server DOWN:", server.URL)
				} else {
					server.Alive = true
					fmt.Println("Server UP:", server.URL)
				}

				if resp != nil {
					resp.Body.Close()
				}
			}

			time.Sleep(5 * time.Second)
		}
	}()
}