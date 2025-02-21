package registry

import (
	"context"
	"errors"
	"fmt"
	"path"

	"gokube/pkg/api"
	"gokube/pkg/storage"
)

const (
	nodePrefix = "/registry/nodes/"
)

var (
	ErrNodeNotFound      = errors.New("node not found")
	ErrNodeAlreadyExists = errors.New("node already exists")
	ErrListNodesFailed   = errors.New("failed to list nodes")
	ErrNodeInvalid       = errors.New("invalid node")
)

// NodeRegistry provides CRUD operations for Node objects
type NodeRegistry struct {
	storage storage.Storage
}

// NewNodeRegistry creates a new NodeRegistry
func NewNodeRegistry(storage storage.Storage) *NodeRegistry {
	return &NodeRegistry{storage: storage}
}

// generateKey generates the storage key for a given node name
func generateKey(prefix, name string) string {
	return path.Join(prefix, name)
}

// CreateNode stores a new Node
func (r *NodeRegistry) CreateNode(ctx context.Context, node *api.Node) error {
	if node == nil || node.Name == "" {
		return ErrNodeInvalid
	}
	if err := node.Validate(); err != nil {
		return ErrNodeInvalid
	}

	// Check if node already exists
	key := generateKey(nodePrefix, node.Name)
	existingNode := &api.Node{}
	err := r.storage.Get(ctx, key, existingNode)
	if errors.Is(err, storage.ErrNotFound) {
		return ErrNodeAlreadyExists
	}

	if err != nil {
		return fmt.Errorf("failed to check existing node: %w", err)
	}

	// Store the node
	if err := r.storage.Create(ctx, key, node); err != nil {
		return fmt.Errorf("failed to create node: %w", err)
	}

	return nil
}

// GetNode retrieves a Node by name
func (r *NodeRegistry) GetNode(ctx context.Context, name string) (*api.Node, error) {
	if name == "" {
		return nil, ErrNodeInvalid
	}

	key := generateKey(nodePrefix, name)
	node := &api.Node{}
	err := r.storage.Get(ctx, key, node)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, fmt.Errorf("node %s not found", name)
		}
		return nil, ErrInternal
	}

	return node, nil
}

// UpdateNode updates an existing Node
func (r *NodeRegistry) UpdateNode(ctx context.Context, node *api.Node) error {
	// Validate node
	if node == nil || node.Name == "" {
		return ErrNodeInvalid
	}
	if err := node.Validate(); err != nil {
		return ErrNodeInvalid
	}

	// Check if node exists
	key := generateKey(nodePrefix, node.Name)
	existingNode := &api.Node{}
	err := r.storage.Get(ctx, key, existingNode)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return ErrNodeNotFound
		}
		return fmt.Errorf("failed to check existing node: %w", err)
	}

	// Update the node
	if err := r.storage.Update(ctx, key, node); err != nil {
		return fmt.Errorf("failed to update node: %w", err)
	}

	return nil
}

// DeleteNode removes a Node by name
func (r *NodeRegistry) DeleteNode(ctx context.Context, name string) error {
	if name == "" {
		return ErrNodeInvalid
	}

	key := generateKey(nodePrefix, name)
	err := r.storage.Delete(ctx, key)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	return nil
}

// ListNodes retrieves all Nodes
func (r *NodeRegistry) ListNodes(ctx context.Context) ([]*api.Node, error) {
	var nodes []*api.Node
	err := r.storage.List(ctx, nodePrefix, &nodes)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrListNodesFailed, err)
	}

	return nodes, nil
}
