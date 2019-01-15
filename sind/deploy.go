package sind

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

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

	archivePath, err := prepareArchive(ctx, hostClient, imgs)
	if err != nil {
		return fmt.Errorf("unable to prepare the archive: %v", err)
	}
	defer os.Remove(archivePath)

	containers, err := c.ContainerList(ctx)
	if err != nil {
		return fmt.Errorf("unable to get container list %v", err)
	}

	var errg errgroup.Group
	for _, container := range containers {
		cID := container.ID
		errg.Go(func() error {
			return deployImage(ctx, hostClient, archivePath, cID)
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
					archivePath,
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

	if err := client.CopyToContainer(ctx, containerID, filepath.Dir(filePath), file, types.CopyToContainerOptions{}); err != nil {
		return fmt.Errorf("unable to copy the image to container %s: %v", containerID, err)
	}

	return nil
}

func prepareArchive(ctx context.Context, hostClient *docker.Client, imgs []types.ImageSummary) (string, error) {
	imgsFile, err := ioutil.TempFile("", "img_sind")
	if err != nil {
		return "", fmt.Errorf("unable to create the image file: %v", err)
	}

	defer func() {
		imgsFile.Close()
		os.Remove(imgsFile.Name())
	}()

	imgReader, err := hostClient.ImageSave(ctx, toImageIDs(imgs))
	if err != nil {
		return "", fmt.Errorf("unable to save the images to disk: %v", err)
	}
	defer imgReader.Close()

	if bytes, err := io.Copy(imgsFile, imgReader); err != nil {
		return "", fmt.Errorf("unable to save the images to disk (copied %d): %v", bytes, err)
	}

	if _, err = imgsFile.Seek(0, 0); err != nil {
		return "", fmt.Errorf("unable to seek to the begining of the image file: %v", err)
	}

	tarImgsFile, err := ioutil.TempFile("", "tar_img_sind")
	if err != nil {
		return "", fmt.Errorf("unable to create the tar file: %v", err)
	}
	defer tarImgsFile.Close()

	imgsFileInfo, err := imgsFile.Stat()
	if err != nil {
		return "", fmt.Errorf("unabel to collect images file info: %v", err)
	}

	tarWriter := tar.NewWriter(tarImgsFile)

	err = tarWriter.WriteHeader(
		&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     imgsFileInfo.Name(),
			Size:     imgsFileInfo.Size(),
			Mode:     0664,
		},
	)
	if err != nil {
		return "", fmt.Errorf("unable to write tar file header: %v", err)
	}

	bytes, err := io.Copy(tarWriter, imgsFile)
	if err != nil {
		return "", fmt.Errorf("unable to tar image files (wrote %d): %v", bytes, err)
	}

	if err = tarWriter.Close(); err != nil {
		return "", fmt.Errorf("unable to close the tar writer properly (wrote %d): %v", bytes, err)
	}

	return tarImgsFile.Name(), nil
}

func toImageIDs(imgs []types.ImageSummary) []string {
	res := make([]string, len(imgs))

	for i, img := range imgs {
		res[i] = img.ID
	}

	return res
}
