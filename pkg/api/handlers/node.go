package handlers

import (
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
	node := &api.Node{}
	if err := request.ReadEntity(node); err != nil {
		api.WriteError(response, http.StatusBadRequest, err)
		return
	}

	err := h.nodeRegistry.CreateNode(request.Request.Context(), node)
	h.handleNodeResponse(response, http.StatusCreated, node, err)
}

// GetNode handles GET requests to retrieve a Node
func (h *NodeHandler) GetNode(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	node, err := h.nodeRegistry.GetNode(request.Request.Context(), name)
	h.handleNodeResponse(response, http.StatusOK, node, err)
}

// UpdateNode handles PUT requests to update a Node
func (h *NodeHandler) UpdateNode(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	node := &api.Node{}
	if err := request.ReadEntity(node); err != nil {
		api.WriteError(response, http.StatusBadRequest, err)
		return
	}

	if name != node.Name {
		api.WriteError(response, http.StatusBadRequest, registry.ErrNodeInvalid)
		return
	}

	err := h.nodeRegistry.UpdateNode(request.Request.Context(), node)
	h.handleNodeResponse(response, http.StatusOK, node, err)
}

// handleNodeResponse processes the response for node operations, handling both success and error cases
func (h *NodeHandler) handleNodeResponse(response *restful.Response, successStatus int, result interface{}, err error) {
	if err != nil {
		switch {
		case errors.Is(err, registry.ErrNodeNotFound):
			api.WriteError(response, http.StatusNotFound, err)
		case errors.Is(err, registry.ErrNodeInvalid):
			api.WriteError(response, http.StatusBadRequest, err)
		case errors.Is(err, registry.ErrNodeAlreadyExists):
			api.WriteError(response, http.StatusConflict, err)
		case errors.Is(err, registry.ErrListNodesFailed):
			api.WriteError(response, http.StatusInternalServerError, err)
		case errors.Is(err, registry.ErrInternal):
			api.WriteError(response, http.StatusInternalServerError, err)
		default:
			api.WriteError(response, http.StatusInternalServerError, err)
		}
		return
	}

	api.WriteResponse(response, successStatus, result)
}

// DeleteNode handles DELETE requests to remove a Node
func (h *NodeHandler) DeleteNode(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	err := h.nodeRegistry.DeleteNode(request.Request.Context(), name)
	h.handleNodeResponse(response, http.StatusNoContent, name, err)
}

// ListNodes handles GET requests to list all Nodes
func (h *NodeHandler) ListNodes(request *restful.Request, response *restful.Response) {
	nodes, err := h.nodeRegistry.ListNodes(request.Request.Context())
	h.handleNodeResponse(response, http.StatusOK, nodes, err)
}

// RegisterNodeRoutes registers Node routes with the WebService
func RegisterNodeRoutes(ws *restful.WebService, handler *NodeHandler) {
	ws.Route(ws.POST("/nodes").To(handler.CreateNode))
	ws.Route(ws.GET("/nodes").To(handler.ListNodes))
	ws.Route(ws.GET("/nodes/{name}").To(handler.GetNode))
	ws.Route(ws.PUT("/nodes/{name}").To(handler.UpdateNode))
	ws.Route(ws.DELETE("/nodes/{name}").To(handler.DeleteNode))
}
