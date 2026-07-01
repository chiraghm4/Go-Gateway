package main

import (
	"api-gateway/internal/router"
	"log"
	"net/http"
	"os"
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

	mux, err := router.SetupRoutes()
	if err != nil {
		log.Fatal("Server failed: ", err)
	}

	server := &http.Server{
		Addr: ":8080",
		Handler: mux,
		ReadTimeout: 5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Println("API Gateway running on :8080")

	log.Fatal(server.ListenAndServe())

}