package main

import (
	"context"
	"log"

	"github.com/jlevesy/sind/pkg/sind"
)

func main() {
	ctx := context.Background()
	params := sind.ClusterConfiguration{
		ClusterName: "test",
		NetworkName: "test",

		Managers: 1,
		Workers:  1,
	}
	_, err := sind.CreateCluster(ctx, params)

	if err != nil {
		log.Fatalf("unable to create cluster: %v", err)
	}
}
