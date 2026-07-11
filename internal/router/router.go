package router

import (
	"api-gateway/internal/loadbalancer"
	"api-gateway/internal/middleware"
	"fmt"
	"log"
	"net/http"
)

type EndpointConfig struct {
	Path     string
	Method   string
	Upstream string
}

func SetupRoutes(upstreamTargets map[string][]string, endpoints []EndpointConfig) (*http.ServeMux, error) {
	mux := http.NewServeMux()

	rateLimiter := middleware.NewRateLimiter(250, 250)

	lbs := make(map[string]*loadbalancer.RoundRobin)
	startedHealthChecks := make(map[string]bool)

	for _, ep := range endpoints {
		targets, ok := upstreamTargets[ep.Upstream]
		if !ok {
			return nil, fmt.Errorf("upstream %q not found in config", ep.Upstream)
		}

		lb, ok := lbs[ep.Upstream]
		if !ok {
			var err error
			lb, err = loadbalancer.NewRoundRobin(targets)
			if err != nil {
				return nil, fmt.Errorf("failed to create load balancer for %q: %w", ep.Upstream, err)
			}
			lbs[ep.Upstream] = lb

			if !startedHealthChecks[ep.Upstream] {
				lb.StartHealthCheck()
				startedHealthChecks[ep.Upstream] = true
			}
		}

		handler := middleware.Chain(lb, rateLimiter.Middleware, middleware.LoggingMiddleware)
		mux.Handle(ep.Path, handler)
		log.Printf("registered route %s -> upstream %s", ep.Path, ep.Upstream)
	}

	return mux, nil
}
