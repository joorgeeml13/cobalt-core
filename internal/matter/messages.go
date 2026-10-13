package matter

import (
	"encoding/json"
)

// CommandRequest representa un comando enviado a python-matter-server.
type CommandRequest struct {
	MessageID string         `json:"message_id"`
	Command   string         `json:"command"`
	Args      map[string]any `json:"args,omitempty"`
}

// CommandResponse representa la respuesta a un comando previo identificado por MessageID.
type CommandResponse struct {
	MessageID string          `json:"message_id"`
	ErrorCode *int            `json:"error_code,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
}

// EventMessage representa un evento emitido asíncronamente por python-matter-server.
type EventMessage struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// NodeEventData describe la información emitida en eventos de nodos Matter.
type NodeEventData struct {
	NodeID     uint64         `json:"node_id"`
	EndpointID *uint          `json:"endpoint_id,omitempty"`
	ClusterID  *uint          `json:"cluster_id,omitempty"`
	Available  *bool          `json:"available,omitempty"`
	Attributes map[string]any `json:"attributes,omitempty"`
}
