package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"wallet-monitor/internal/config"
	grpcclient "wallet-monitor/internal/grpc"
	"wallet-monitor/internal/subscriber"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	cfg := config.Parse()
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		fmt.Fprintln(os.Stderr, "Usage: wallet-monitor --endpoint <url> --wallet <address> [--token <token>] [--username <user> --password <pass>] [--insecure]")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Printf("Connecting to %s ...", cfg.Endpoint)
	conn, err := grpcclient.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	log.Println("Connected")

	sub := subscriber.New(conn, cfg)
	if err := sub.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("Subscription error: %v", err)
	}
}
