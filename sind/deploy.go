package sind

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/golang/sync/errgroup"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	docker "github.com/docker/docker/client"
)

// Errors.
var (
	ErrImageReferenceNotFound = errors.New("image reference not found")
)

// DeployImage deploys an image from the host to the cluster.
func (c *Cluster) DeployImage(ctx context.Context, ref string) error {
	hostClient, err := c.Host.Client()
	if err != nil {
		return fmt.Errorf("unable to get host client: %v", err)
	}

	imgs, err := hostClient.ImageList(
		ctx,
		types.ImageListOptions{Filters: filters.NewArgs(filters.Arg("reference", ref))},
	)
	if err != nil {
		return fmt.Errorf("unable to lookup image on host: %v", err)
	}

	if len(imgs) == 0 {
		return ErrImageReferenceNotFound
	}

	file, err := ioutil.TempFile("", "sind")
	if err != nil {
		return fmt.Errorf("unable to create the image file: %v", err)
	}

	tarWriter := tar.NewWriter(file)

	defer func() {
		file.Close()
		os.Remove(file.Name())
	}()

	imgReader, err := hostClient.ImageSave(ctx, toImageIDs(imgs))
	if err != nil {
		return fmt.Errorf("unable to save the image to disk: %v", err)
	}
	defer imgReader.Close()

	if _, err = io.Copy(tarWriter, imgReader); err != nil {
		return fmt.Errorf("unable to save the image to disk: %v", err)
	}

	if err = tarWriter.Close(); err != nil {
		return fmt.Errorf("unable to close the tar writer: %v", err)
	}

	containers, err := c.ContainerList(ctx)
	if err != nil {
		return fmt.Errorf("unable to get container list %v", err)
	}

	var errg errgroup.Group
	for _, container := range containers {
		cID := container.ID
		errg.Go(func() error {
			return deployImage(ctx, hostClient, file.Name(), cID)
		})
	}

	if err = errg.Wait(); err != nil {
		return fmt.Errorf("unable to deploy the image to host: %v", err)
	}

	errg = errgroup.Group{}
	for _, container := range containers {
		cID := container.ID
		errg.Go(func() error {
			return execContainer(
				ctx,
				hostClient,
				cID,
				[]string{
					"docker",
					"load",
					"-i",
					file.Name(),
				},
			)
		})
	}

	if err = errg.Wait(); err != nil {
		return fmt.Errorf("unable to load the image on the host: %v", err)
	}

	return nil
}

func deployImage(ctx context.Context, client *docker.Client, filePath, containerID string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("unable to open file to deploy: %v", err)
	}

	defer file.Close()

	if err := client.CopyToContainer(ctx, containerID, filePath, file, types.CopyToContainerOptions{}); err != nil {
		return fmt.Errorf("unable to copy the image to container %s: %v", containerID, err)
	}

	return nil
}

func toImageIDs(imgs []types.ImageSummary) []string {
	res := make([]string, len(imgs))

	for i, img := range imgs {
		res[i] = img.ID
	}

	return res
}
