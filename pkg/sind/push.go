package sind

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	docker "github.com/docker/docker/client"
	"github.com/jlevesy/sind/pkg/sind/internal"
)

// PushImageRefs pushes given refs to all node of a cluster.
func PushImageRefs(ctx context.Context, hostClient *docker.Client, clusterName string, refs []string) error {
	imagesFile, err := ioutil.TempFile(os.TempDir(), "sind_images")
	if err != nil {
		return fmt.Errorf("unable to create a temporary archive file: %v", err)
	}

	defer os.Remove(imagesFile.Name())
	defer imagesFile.Close()

	if err = internal.SaveImages(ctx, hostClient, imagesFile, refs); err != nil {
		return fmt.Errorf("unable to save images to file: %v", err)
	}

	return PushImageFile(ctx, hostClient, clusterName, imagesFile)
}

// PushImageFile pushes a given image archive file on all the nodes of a given Cluster.
func PushImageFile(ctx context.Context, hostClient *docker.Client, clusterName string, file *os.File) error {
	containers, err := internal.ListContainers(ctx, hostClient, clusterName)
	if err != nil {
		return fmt.Errorf("unable to list cluster %q containers: %v", clusterName, err)
	}

	archiveFile, err := ioutil.TempFile(os.TempDir(), "sind_archive")
	if err != nil {
		return fmt.Errorf("unable to create a temporary archive file: %v", err)
	}

	defer os.Remove(archiveFile.Name())
	defer archiveFile.Close()

	if err = internal.TarFile(file, archiveFile); err != nil {
		return fmt.Errorf("unable to tar file: %v", err)
	}

	if err = internal.CopyToContainers(ctx, hostClient, containers, archiveFile.Name(), "/"); err != nil {
		return fmt.Errorf("unable to copy content to containers: %v", err)
	}

	err = internal.ExecContainers(
		ctx,
		hostClient,
		containers,
		[]string{
			"docker",
			"load",
			"-i",
			filepath.Join("/", filepath.Base(file.Name())),
		},
	)
	if err != nil {
		return fmt.Errorf("unable to load image on nodes daemons: %v", err)
	}

	return nil
}
