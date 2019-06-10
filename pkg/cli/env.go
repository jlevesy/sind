package cli

import (
	"context"
	"fmt"
	"os"
	"syscall"

	docker "github.com/docker/docker/client"
	"github.com/jlevesy/sind/pkg/cli/internal"
	"github.com/jlevesy/sind/pkg/sind"
	"github.com/spf13/cobra"
)

var (
	envCmd = &cobra.Command{
		Use:   "env",
		Short: "Sets up docker env variables.",
		Run:   runEnv,
	}
)

func init() {
	rootCmd.AddCommand(envCmd)
}

func runEnv(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ctx, cancel = internal.WithSignal(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	client, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithVersion("1.39"))
	if err != nil {
		fmt.Printf("unable to collect to the docker daemon: %v", err)
		os.Exit(1)
	}

	host, err := sind.ClusterHost(ctx, client, clusterName)
	if err != nil {
		fmt.Printf("unable to collect cluster information: %v", err)
		os.Exit(1)
	}

	fmt.Printf("export DOCKER_HOST=%s", host)
}
