package kubelet

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"strconv"
	"testing"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/server/v3/embed"
	"gokube/pkg/api"
	"gokube/pkg/api/server"
	"gokube/pkg/storage"
)

func TestGokubeEndToEnd(t *testing.T) {
	cluster := setupTestCluster(t)
	defer cluster.Cleanup()

	// Wait for the kubeletes to be registered with the api server
	err := waitForKubeletRegistration(cluster.APIServerURL, 3)
	assert.NoError(t, err)

}

type TestCluster struct {
	EtcdServer   *embed.Etcd
	EtcdClient   *clientv3.Client
	Storage      *storage.EtcdStorage
	APIServer    *server.APIServer
	APIServerURL string
	Kubelets     []*Kubelet
}

func setupTestCluster(t *testing.T) *TestCluster {

	// Start embedded etcd
	etcdServer, _, err := storage.StartEmbeddedEtcd()
	if err != nil {
		t.Fatalf("Failed to start embedded etcd: %v", err)
	}

	// Setup etcd client
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdServer.Config().ListenClientUrls[0].String()},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create etcd client: %v", err)
	}

	// Create storage and registries
	etcdStorage := storage.NewEtcdStorage(etcdClient)
	// Create API server
	apiServer := server.NewAPIServer(etcdStorage)

	// Start the API server
	port, err := storage.PickAvailableRandomPort()
	if err != nil {
		t.Fatalf("Failed to pick available random port: %v", err)
	}

	serverURL := "localhost:" + strconv.Itoa(port)
	//TODO: Is this the idiomatic way to handle errors in goroutines?
	go func() {
		err := apiServer.Start(serverURL)
		if err != nil {
			t.Errorf("Failed to start API server: %v", err)
		}
	}()
	// Wait for the API server to be ready
	if err := waitForAPIServer(serverURL); err != nil {
		t.Fatalf("API server failed to start: %v", err)
	}
	t.Log("API Server started at:", serverURL)

	kubelets, err := startKubelets(serverURL, 3, t)
	if err != nil {
		t.Fatalf("Failed to start kubelets: %v", err)
	}

	return &TestCluster{
		EtcdServer:   etcdServer,
		EtcdClient:   etcdClient,
		Storage:      etcdStorage,
		APIServer:    apiServer,
		Kubelets:     kubelets,
		APIServerURL: serverURL,
	}
}

func startKubelets(apiServerIPAndPort string, count int, t *testing.T) ([]*Kubelet, error) {
	var kubelets []*Kubelet
	for i := 0; i < count; i++ {
		nodeName := fmt.Sprintf("node-%d", i)
		k, err := NewKubelet(nodeName, apiServerIPAndPort)
		if err != nil {
			return nil, fmt.Errorf("failed to create Kubelet %s: %v", nodeName, err)
		}
		go func() {
			err := k.Start()
			if err != nil {
				t.Errorf("Failed to start Kubelet %s: %v", nodeName, err)
			}
		}()
		kubelets = append(kubelets, k)
	}
	return kubelets, nil
}

func waitForKubeletRegistration(apiServerURL string, expectedCount int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for Kubelets to register")
		default:
			resp, err := http.Get("http://" + apiServerURL + "/api/v1/nodes")
			if err != nil {
				return fmt.Errorf("failed to list nodes: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			}

			var nodeList []api.Node
			if err := json.NewDecoder(resp.Body).Decode(&nodeList); err != nil {
				return fmt.Errorf("failed to decode node list: %v", err)
			}

			readyCount := 0
			for _, node := range nodeList {
				if node.Status == api.NodeReady {
					readyCount++
				}
			}

			if readyCount == expectedCount {
				return nil
			}

			time.Sleep(1 * time.Second)
		}
	}
}

func (tc *TestCluster) Cleanup() {
	tc.EtcdClient.Close()
	storage.StopEmbeddedEtcd(tc.EtcdServer)

}

func waitForAPIServer(url string) error {
	for i := 0; i < 30; i++ {
		resp, err := http.Get("http://" + url + "/api/v1/healthz")
		if err == nil && resp.StatusCode == http.StatusOK {

			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("API server did not become ready in time")
}
