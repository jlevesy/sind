package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jlevesy/sind/pkg/sind"
)

func main() {
	log.Println("Creating a new swarm cluster")

	createCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	params := sind.ClusterConfiguration{
		ClusterName: "test",
		NetworkName: "swarmynet",

		Managers: 3,
		Workers:  2,
	}

	if err := sind.CreateCluster(createCtx, params); err != nil {
		log.Fatalf("unable to create cluster %v", err)
	}

	defer func() {
		deleteCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = deleteCtx
		// TODO implement delete
	}()

	log.Println("success, press ctrl+C to stop")

	sig := make(chan os.Signal, 1)

	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	<-sig
}
