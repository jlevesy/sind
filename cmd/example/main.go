package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	docker "github.com/docker/docker/client"
	"github.com/jlevesy/sind/pkg/sind"
)

func main() {
	log.Println("Creating a new swarm cluster")

	createCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithVersion("1.39"))
	if err != nil {
		log.Fatalf("unable to create docker client: %v", err)
	}

	params := sind.ClusterConfiguration{
		ClusterName: "test",
		NetworkName: "swarmynet",

		Managers: 3,
		Workers:  2,
	}

	if err := sind.CreateCluster(createCtx, client, params); err != nil {
		log.Fatalf("unable to create cluster %v", err)
	}

	defer func() {
		deleteCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err = sind.DeleteCluster(deleteCtx, client, params.ClusterName); err != nil {
			log.Fatalf("unable to delete the cluster:  %v", err)
		}

		log.Println("Cluster deleted !")
	}()

	log.Println("success, press ctrl+C to stop")

	sig := make(chan os.Signal, 1)

	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	<-sig
}
