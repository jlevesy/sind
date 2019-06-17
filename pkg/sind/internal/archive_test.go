package internal

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTarFile(t *testing.T) {
	file, err := ioutil.TempFile(os.TempDir(), "test_sind_tar_file")
	require.NoError(t, err)

	defer os.Remove(file.Name())
	defer file.Close()

	destFile, err := ioutil.TempFile(os.TempDir(), "test_sind_tar_file")
	require.NoError(t, err)

	defer os.Remove(destFile.Name())
	defer destFile.Close()

	content := []byte("test")

	_, err = file.Write(content)
	require.NoError(t, err)

	_, err = file.Seek(0, 0)
	require.NoError(t, err)

	require.NoError(t, TarFile(file, destFile))

	_, err = destFile.Seek(0, 0)
	require.NoError(t, err)

	fileInfo, err := file.Stat()
	require.NoError(t, err)

	tr := tar.NewReader(destFile)

	hdr, err := tr.Next()
	require.NoError(t, err)

	assert.Equal(t, fileInfo.Name(), hdr.Name)
	assert.Equal(t, fileInfo.Size(), hdr.Size)
	assert.Equal(t, int64(fileInfo.Mode()), hdr.Mode)
	if _, err := io.Copy(os.Stdout, tr); err != nil {
		log.Fatal(err)
	}
	fmt.Println()

	_, err = tr.Next()
	assert.Equal(t, io.EOF, err)
}
