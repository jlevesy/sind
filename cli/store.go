package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/jlevesy/go-sind/sind"
)

// Errors.
const (
	ErrAlreadyExists      = "cluster already exists"
	ErrMissingClusterName = "missing cluster name"
	ErrClusterNotFound    = "cluster not found"
)

// Store is in charge of storing and retrieving clusters.
type Store struct {
	filePath string
}

type clusters map[string]sind.Cluster

// NewStore creates and initializes a store.
func NewStore() (*Store, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("unable to initialize configuration store: %v", err)
	}

	path := filepath.Join(usr.HomeDir, ".config", "sind")

	if err = os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("unable to initialize configuration store: %v", err)
	}

	filePath := filepath.Join(path, "clusters.json")

	_, err = os.Stat(filePath)
	if os.IsNotExist(err) {
		err = initStorageFile(filePath)
	}
	if err != nil {
		return nil, fmt.Errorf("unable to init the configuration file: %v", err)
	}

	return &Store{filePath: filePath}, nil
}

func initStorageFile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("unable to create the configuration file: %v", err)
	}
	defer file.Close()

	clusters := make(clusters)
	if err = json.NewEncoder(file).Encode(clusters); err != nil {
		return fmt.Errorf("unable to initialize configuration file: %v", err)
	}

	return nil
}

// ValidateName will return true if a cluster already have this name.
func (s *Store) ValidateName(clusterName string) error {
	if clusterName == "" {
		return errors.New(ErrMissingClusterName)
	}

	clusters, err := s.readAll()
	if err != nil {
		return fmt.Errorf("unable to read existing clusters: %v", err)
	}

	if _, ok := clusters[clusterName]; ok {
		return errors.New(ErrAlreadyExists)
	}

	return nil
}

// Save will persist a new cluster.
func (s *Store) Save(cluster sind.Cluster) error {
	clusters, err := s.readAll()
	if err != nil {
		return fmt.Errorf("unable to read existing clusters: %v", err)
	}

	clusters[cluster.Name] = cluster

	return s.writeAll(clusters)
}

// Load will return a cluster according to its name.
func (s *Store) Load(clusterName string) (*sind.Cluster, error) {
	clusters, err := s.readAll()
	if err != nil {
		return nil, fmt.Errorf("unable to read existing clusters: %v", err)
	}

	cluster, ok := clusters[clusterName]
	if !ok {
		return nil, errors.New(ErrClusterNotFound)
	}

	return &cluster, nil
}

// Delete will delete a cluster from configuration.
func (s *Store) Delete(clusterName string) error {
	clusters, err := s.readAll()
	if err != nil {
		return fmt.Errorf("unable to read existing clusters: %v", err)
	}

	_, ok := clusters[clusterName]
	if !ok {
		return errors.New(ErrClusterNotFound)
	}

	delete(clusters, clusterName)

	return s.writeAll(clusters)
}

// List will return all existing clusters.
func (s *Store) List() ([]sind.Cluster, error) {
	clusters, err := s.readAll()
	if err != nil {
		return nil, fmt.Errorf("unable to read existing clusters: %v", err)
	}

	result := make([]sind.Cluster, 0, len(clusters))
	for _, cluster := range clusters {
		result = append(result, cluster)
	}
	return result, nil
}

func (s *Store) writeAll(clusters clusters) error {
	file, err := os.OpenFile(s.filePath, os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("unable to open the clusters file: %v", err)
	}
	defer file.Close()

	if err = json.NewEncoder(file).Encode(clusters); err != nil {
		return fmt.Errorf("unable to encode clusters file: %v", err)
	}

	return nil
}

func (s *Store) readAll() (clusters, error) {
	file, err := os.Open(s.filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to open the clusters file: %v", err)
	}
	defer file.Close()
	var clusters clusters
	if err := json.NewDecoder(file).Decode(&clusters); err != nil {
		return nil, fmt.Errorf("unable to decode clusters file: %v", err)
	}

	return clusters, nil
}
