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
	Gateway GatewayConfig `yaml:"gateway"`
	Upstreams map[string]Upstream `yaml:"upstreams"`
	Endpoints []Endpoint `yaml:"endpoint"`
}

type GatewayConfig struct {
	Port int `yaml:"port"`
	Timeout int `yaml:"timeout_seconds"`
}

type Upstream struct {
	Targets []Target `yaml:"targets"`
}

type Endpoint struct {
	Path string `yaml:"path"`
	Method string `yaml:"method"`
	Upstream string `yaml:"upstream"`
}

type Target struct {
	Host string `yaml:"host"`
	TargetPort int `yaml:"target_port"`
	Protocol string `yaml:"protocol"`
}

func main() {

	// yaml parsing

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

	upstreams := config.Upstreams
	upstreamIPs := []string{}

	for _, ups := range(upstreams) {
		for _, targets := range(ups.Targets) {
			ip := targets.Protocol + "://" + targets.Host + ":" + strconv.Itoa(targets.TargetPort)
			upstreamIPs = append(upstreamIPs, ip)
		}
	}

	mux, err := router.SetupRoutes(upstreamIPs)
	if err != nil {
		log.Fatal("Server failed: ", err)
	}

	server := &http.Server{
		Addr: PORT,
		Handler: mux,
		ReadTimeout: time.Duration(config.Gateway.Timeout/2) * time.Second,
		WriteTimeout: time.Duration(config.Gateway.Timeout/2) * time.Second,
	}

	log.Println("API Gateway running on :8080")

	log.Fatal(server.ListenAndServe())

}