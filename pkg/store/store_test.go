package store

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/jlevesy/sind/pkg/sind"
	"github.com/stretchr/testify/require"
)

func prepareStore() (*Store, func(), error) {
	tmpStorage, err := ioutil.TempDir("", "sind_test")
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create tmp dir: %v", err)
	}

	filePath := filepath.Join(tmpStorage, "clusters.json")

	if err = initStorageFile(filePath); err != nil {
		return nil, nil, fmt.Errorf("unable to setup storage file: %v", err)
	}

	return &Store{filePath: filePath}, func() { os.Remove(filePath) }, nil
}

func TestStore(t *testing.T) {
	st, cleanup, err := prepareStore()
	require.NoError(t, err)
	defer cleanup()

	cluster := sind.Cluster{Name: "test_0", Host: sind.Docker{Host: "unix://var/run/docker.sock"}}

	require.NoError(t, st.Save(cluster))

	err = st.Exists(cluster.Name)
	require.Error(t, err)
	require.Equal(t, err.Error(), ErrAlreadyExists)

	readCluster, err := st.Load(cluster.Name)
	require.NoError(t, err)

	require.Equal(t, *readCluster, cluster)

	clusters, err := st.List()
	require.NoError(t, err)
	require.Len(t, clusters, 1)
	require.Equal(t, clusters[0], cluster)

	require.NoError(t, st.Delete(cluster.Name))

	clusters, err = st.List()
	require.NoError(t, err)

	require.Len(t, clusters, 0)
}
