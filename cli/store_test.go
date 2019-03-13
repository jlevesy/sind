package cli

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jlevesy/go-sind/sind"
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
	if err != nil {
		t.Fatalf("unable to prepare test store: %v", err)
	}
	defer cleanup()

	cluster := sind.Cluster{Name: "test_0", Host: sind.Docker{Host: "unix://var/run/docker.sock"}}

	if err = st.Save(cluster); err != nil {
		t.Fatalf("unable to save cluster: %v", err)
	}

	if err = st.ValidateName(cluster.Name); err.Error() != ErrAlreadyExists {
		t.Fatalf("unexpected error while validating cluster name: %v", err)
	}

	readCluster, err := st.Load(cluster.Name)
	if err != nil {
		t.Fatalf("unable to load cluster by name: %v", err)
	}

	if !reflect.DeepEqual(*readCluster, cluster) {
		t.Fatalf("read cluster setup differs from intitial config, %+v, %+v", *readCluster, cluster)
	}

	clusters, err := st.List()
	if err != nil {
		t.Fatalf("unable to list clusters: %v", err)
	}

	if len(clusters) != 1 {
		t.Fatalf("unexpected count of clusters, expected 1 got: %d", len(clusters))
	}

	if !reflect.DeepEqual(clusters[0], cluster) {
		t.Fatalf("read cluster setup differs from intitial config, %+v, %+v", clusters[0], cluster)
	}

	if err = st.Delete(cluster.Name); err != nil {
		t.Fatalf("unable to delete cluster: %v", err)
	}

	clusters, err = st.List()
	if err != nil {
		t.Fatalf("unable to list clusters: %v", err)
	}

	if len(clusters) != 0 {
		t.Fatalf("unexpected count of clusters, expected 0 got: %d", len(clusters))
	}
}
