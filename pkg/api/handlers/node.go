package handlers

import (
	"context"
	"errors"
	"net/http"

	"gokube/pkg/api"
	"gokube/pkg/registry"

	"github.com/emicklei/go-restful/v3"
)

// NodeHandler handles Node-related HTTP requests
type NodeHandler struct {
	nodeRegistry *registry.NodeRegistry
}

// NewNodeHandler creates a new NodeHandler
func NewNodeHandler(nodeRegistry *registry.NodeRegistry) *NodeHandler {
	return &NodeHandler{nodeRegistry: nodeRegistry}
}

// CreateNode handles POST requests to create a new Node
func (h *NodeHandler) CreateNode(request *restful.Request, response *restful.Response) {
	node := new(api.Node)
	if err := request.ReadEntity(node); err != nil {
		api.WriteError(response, http.StatusBadRequest, err)
		return
	}

	err := h.nodeRegistry.CreateNode(context.Background(), node)
	if err != nil {
		switch {
		case errors.Is(err, registry.ErrNodeInvalid):
			api.WriteError(response, http.StatusBadRequest, err)
		case errors.Is(err, registry.ErrNodeAlreadyExists):
			api.WriteError(response, http.StatusConflict, err)
		default:
			api.WriteError(response, http.StatusInternalServerError, err)
		}
		return
	}

	api.WriteResponse(response, http.StatusCreated, node)
}

// GetNode handles GET requests to retrieve a Node
func (h *NodeHandler) GetNode(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")

	node, err := h.nodeRegistry.GetNode(request.Request.Context(), name)
	if err != nil {
		switch {
		case errors.Is(err, registry.ErrNodeNotFound):
			api.WriteError(response, http.StatusNotFound, err)
		case errors.Is(err, registry.ErrNodeInvalid):
			api.WriteError(response, http.StatusBadRequest, err)
		default:
			api.WriteError(response, http.StatusInternalServerError, err)
		}
		return
	}

	api.WriteResponse(response, http.StatusOK, node)
}

// UpdateNode handles PUT requests to update a Node
func (h *NodeHandler) UpdateNode(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	node := new(api.Node)
	if err := request.ReadEntity(node); err != nil {
		api.WriteError(response, http.StatusBadRequest, err)
		return
	}

	// Verify that the name in the URL matches the name in the body
	if name != node.Name {
		api.WriteError(response, http.StatusBadRequest, errors.New("node name in URL must match name in body"))
		return
	}

	err := h.nodeRegistry.UpdateNode(request.Request.Context(), node)
	if err != nil {
		switch {
		case errors.Is(err, registry.ErrNodeNotFound):
			api.WriteError(response, http.StatusNotFound, err)
		case errors.Is(err, registry.ErrNodeInvalid):
			api.WriteError(response, http.StatusBadRequest, err)
		case errors.Is(err, registry.ErrInternal):
			api.WriteError(response, http.StatusInternalServerError, err)
		default:
			api.WriteError(response, http.StatusInternalServerError, err)
		}
		return
	}

	api.WriteResponse(response, http.StatusOK, node)
}

// DeleteNode handles DELETE requests to remove a Node
func (h *NodeHandler) DeleteNode(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")

	err := h.nodeRegistry.DeleteNode(request.Request.Context(), name)
	if err != nil {
		switch {
		case errors.Is(err, registry.ErrNodeInvalid):
			api.WriteError(response, http.StatusBadRequest, err)
		case errors.Is(err, registry.ErrInternal):
			api.WriteError(response, http.StatusInternalServerError, err)
		default:
			api.WriteError(response, http.StatusInternalServerError, err)
		}
		return
	}

	api.WriteResponse(response, http.StatusNoContent, nil)
}

// ListNodes handles GET requests to list all Nodes
func (h *NodeHandler) ListNodes(request *restful.Request, response *restful.Response) {
	nodes, err := h.nodeRegistry.ListNodes(request.Request.Context())
	if err != nil {
		switch {
		case errors.Is(err, registry.ErrListNodesFailed):
			api.WriteError(response, http.StatusInternalServerError, err)
		case errors.Is(err, registry.ErrInternal):
			api.WriteError(response, http.StatusInternalServerError, err)
		default:
			api.WriteError(response, http.StatusInternalServerError, err)
		}
		return
	}

	api.WriteResponse(response, http.StatusOK, nodes)
}

// RegisterNodeRoutes registers Node routes with the WebService
func RegisterNodeRoutes(ws *restful.WebService, handler *NodeHandler) {
	ws.Route(ws.POST("/nodes").To(handler.CreateNode))
	ws.Route(ws.GET("/nodes").To(handler.ListNodes))
	ws.Route(ws.GET("/nodes/{name}").To(handler.GetNode))
	ws.Route(ws.PUT("/nodes/{name}").To(handler.UpdateNode))
	ws.Route(ws.DELETE("/nodes/{name}").To(handler.DeleteNode))
}
