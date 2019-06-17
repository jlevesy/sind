package internal

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

const (
	imageFilterReference = "reference"
)

type imageLister interface {
	ImageList(context.Context, types.ImageListOptions) ([]types.ImageSummary, error)
}

// ImageExists returns true if given image ref exists on the docker host.
func ImageExists(ctx context.Context, docker imageLister, imageRef string) (bool, error) {
	imageList, err := docker.ImageList(ctx, types.ImageListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg(imageFilterReference, imageRef)),
	})
	if err != nil {
		return false, fmt.Errorf("unable to list images: %v", err)
	}

	if len(imageList) == 0 {
		return false, nil
	}

	return true, nil
}

type imagePuller interface {
	ImagePull(context.Context, string, types.ImagePullOptions) (io.ReadCloser, error)
}

// PullImage pulls given image ref.
func PullImage(ctx context.Context, docker imagePuller, imageRef string) error {
	out, err := docker.ImagePull(ctx, imageRef, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("unable to pull %q: %v", imageRef, err)
	}
	defer out.Close()

	if _, err = io.Copy(ioutil.Discard, out); err != nil {
		return fmt.Errorf("unable to pull %q: %v", imageRef, err)
	}

	return nil
}

type imageSaver interface {
	ImageSave(ctx context.Context, refs []string) (io.ReadCloser, error)
}

// SaveImages saves all given images refs to given target.
func SaveImages(ctx context.Context, hostClient imageSaver, dest io.WriteSeeker, refs []string) error {
	imgReader, err := hostClient.ImageSave(ctx, refs)
	if err != nil {
		return fmt.Errorf("unable to save the images: %v", err)
	}
	defer imgReader.Close()

	var bytes int64
	if bytes, err = io.Copy(dest, imgReader); err != nil {
		return fmt.Errorf("unable to save the images (copied %d): %v", bytes, err)
	}

	if _, err = dest.Seek(0, 0); err != nil {
		return fmt.Errorf("unable to seek the image: %v", err)
	}

	return nil
}
