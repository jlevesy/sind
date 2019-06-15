package internal

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
)

// TarFile writes the given file file to a tar archive.
func TarFile(file, dest *os.File) error {
	contentInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("unable to collect images file info: %v", err)
	}

	tarWriter := tar.NewWriter(dest)

	err = tarWriter.WriteHeader(
		&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     contentInfo.Name(),
			Size:     contentInfo.Size(),
			Mode:     int64(contentInfo.Mode()),
		},
	)
	if err != nil {
		return fmt.Errorf("unable to write tar file header: %v", err)
	}

	bytes, err := io.Copy(tarWriter, file)
	if err != nil {
		return fmt.Errorf("unable to tar image files (wrote %d): %v", bytes, err)
	}

	if err = tarWriter.Close(); err != nil {
		return fmt.Errorf("unable to close the tar writer properly (wrote %d): %v", bytes, err)
	}

	return nil
}
