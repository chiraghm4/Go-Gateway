package main

import (
	"api-gateway/internal/router"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type Gateway struct {
	Gateway   GatewayConfig        `yaml:"gateway"`
	Upstreams map[string]Upstream  `yaml:"upstreams"`
	Endpoints []Endpoint           `yaml:"endpoints"`
}

type GatewayConfig struct {
	Port    int `yaml:"port"`
	Timeout int `yaml:"timeout_seconds"`
}

type Upstream struct {
	Targets []Target `yaml:"targets"`
}

type Endpoint struct {
	Path     string `yaml:"path"`
	Method   string `yaml:"method"`
	Upstream string `yaml:"upstream"`
}

type Target struct {
	Host       string `yaml:"host"`
	TargetPort int    `yaml:"target_port"`
	Protocol   string `yaml:"protocol"`
}

func main() {
	file, err := os.ReadFile("../gateway-config.yaml")
	if err != nil {
		log.Fatalf("cannot open config file - %v", err)
	}

	var config Gateway

	err = yaml.Unmarshal(file, &config)
	if err != nil {
		log.Fatalf("cannot parse config file - %v", err)
	}

	PORT := ":" + strconv.Itoa(config.Gateway.Port)

	upstreamTargets := make(map[string][]string)
	for name, ups := range config.Upstreams {
		var targets []string
		for _, t := range ups.Targets {
			ip := t.Protocol + "://" + t.Host + ":" + strconv.Itoa(t.TargetPort)
			targets = append(targets, ip)
		}
		upstreamTargets[name] = targets
	}

	endpoints := make([]router.EndpointConfig, len(config.Endpoints))
	for i, ep := range config.Endpoints {
		endpoints[i] = router.EndpointConfig{
			Path:     ep.Path,
			Method:   ep.Method,
			Upstream: ep.Upstream,
		}
	}

	mux, err := router.SetupRoutes(upstreamTargets, endpoints)
	if err != nil {
		log.Fatal("Server failed: ", err)
	}

	server := &http.Server{
		Addr:         PORT,
		Handler:      mux,
		ReadTimeout:  time.Duration(config.Gateway.Timeout/2) * time.Second,
		WriteTimeout: time.Duration(config.Gateway.Timeout/2) * time.Second,
	}

	log.Printf("API Gateway running on %s", PORT)

	log.Fatal(server.ListenAndServe())
}
