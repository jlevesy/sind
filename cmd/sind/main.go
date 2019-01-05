package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jlevesy/sind/sind"
)

func main() {
	log.Println("Creating a new swarm cluster")

	createCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	params := sind.CreateClusterParams{
		NetworkName: "swarmynet",

		Masters: 3,
		Workers: 4,
	}

	cluster, err := sind.CreateCluster(createCtx, params)
	if err != nil {
		log.Fatalf("unable to create cluster %v", err)
	}

	defer func() {
		deleteCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cluster.Delete(deleteCtx)
	}()

	log.Println("success, press ctrl+C to stop")

	sig := make(chan os.Signal)

	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	<-sig
}
