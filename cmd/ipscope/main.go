package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"
	"time"

	"ipscope/internal/config"
	"ipscope/internal/geolocation"
	"ipscope/internal/metrics"
	"ipscope/internal/server"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

func main() {
	configPath := flag.String("config", "config.yml", "Path to YAML config file")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	resolver := geolocation.NewAPIClient(10 * time.Second)
	exporter, err := metrics.NewExporter(registry, cfg.Metrics.Prefix, resolver)
	if err != nil {
		log.Fatalf("failed to create metrics exporter: %v", err)
	}

	if err := exporter.Refresh(ctx, cfg.Nodes); err != nil {
		log.Printf("datacenter lookup finished with warnings: %v", err)
	}

	log.Printf("serving metrics on %s:%d", cfg.Server.Host, cfg.Server.Port)
	if err := server.Run(ctx, cfg.Server.Host, cfg.Server.Port, registry); err != nil {
		log.Fatalf("metrics server exited with error: %v", err)
	}
}
