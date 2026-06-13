// Package events provides the Events resource for sending and receiving session events.
// See: https://docs.qoder.com/cloud-agents/api/sessions/create
package events

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	httpclient "github.com/futuretea/go-http-client"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/qoderhttp"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/types"
)

// Event type constants for sending events.
const (
	EventTypeUserMessage          = "user.message"
	EventTypeUserInterrupt        = "user.interrupt"
	EventTypeUserToolConfirmation = "user.tool_confirmation"
	EventTypeUserCustomToolResult = "user.custom_tool_result"
	EventTypeAgentMessage         = "agent.message"
	EventTypeAgentThinking        = "agent.thinking"
	EventTypeAgentToolUse         = "agent.tool_use"
	EventTypeAgentToolResult      = "agent.tool_result"
	EventTypeAgentCustomToolUse   = "agent.custom_tool_use"
	EventTypeAgentMCPToolUse      = "agent.mcp_tool_use"
	EventTypeAgentMCPToolResult   = "agent.mcp_tool_result"
	EventTypeAgentArtifact        = "agent.artifact_delivered"
	EventTypeSessionRunning       = "session.status_running"
	EventTypeSessionIdle          = "session.status_idle"
	EventTypeSessionError         = "session.error"
)

// UserMessageEvent creates a user message event for sending.
type UserMessageEvent struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// NewUserMessage creates a new user message event.
func NewUserMessage(content string) UserMessageEvent {
	return UserMessageEvent{Type: EventTypeUserMessage, Content: content}
}

// NewInterruptEvent creates a user interrupt event to stop a running session.
func NewInterruptEvent() UserMessageEvent {
	return UserMessageEvent{Type: EventTypeUserInterrupt}
}

// NewToolConfirmationEvent creates a tool confirmation event.
// toolUseID is the ID of the tool_use event being confirmed.
// Set approved to true to allow the tool call, false to deny it.
func NewToolConfirmationEvent(toolUseID string, approved bool) (UserMessageEvent, error) {
	payload, err := json.Marshal(map[string]any{
		"tool_use_id": toolUseID,
		"approved":    approved,
	})
	if err != nil {
		return UserMessageEvent{}, fmt.Errorf("events: encode tool confirmation: %w", err)
	}
	return UserMessageEvent{Type: EventTypeUserToolConfirmation, Content: string(payload)}, nil
}

// NewCustomToolResultEvent creates a custom tool result event.
// toolUseID is the ID of the custom_tool_use event.
// result contains the tool result data.
func NewCustomToolResultEvent(toolUseID string, result map[string]any) (UserMessageEvent, error) {
	payload, err := json.Marshal(map[string]any{
		"tool_use_id": toolUseID,
		"result":      result,
	})
	if err != nil {
		return UserMessageEvent{}, fmt.Errorf("events: encode custom tool result: %w", err)
	}
	return UserMessageEvent{Type: EventTypeUserCustomToolResult, Content: string(payload)}, nil
}

// SendEventRequest wraps events for sending to a session.
type SendEventRequest struct {
	Events []UserMessageEvent `json:"events"`
}

// NewSendRequest creates a request to send events to a session.
func NewSendRequest(events ...UserMessageEvent) *SendEventRequest {
	return &SendEventRequest{Events: events}
}

// StreamEvent represents a parsed SSE event from the event stream.
type StreamEvent = qoderhttp.SSEEvent

// API provides access to the Events resource.
type API struct {
	client httpclient.Client
}

// NewAPI creates a new Events API client.
func NewAPI(client httpclient.Client) *API {
	return &API{client: client}
}

// Send sends one or more user message events to a session.
func (a *API) Send(ctx context.Context, sessionID string, req *SendEventRequest) error {
	if err := qoderhttp.ValidateID(sessionID); err != nil {
		return err
	}
	return a.client.POST("/sessions/" + sessionID + "/events").WithJSON(req).WithContext(ctx).Do(nil)
}

// SendMessage is a convenience method to send a single user message.
func (a *API) SendMessage(ctx context.Context, sessionID, content string) error {
	return a.Send(ctx, sessionID, NewSendRequest(NewUserMessage(content)))
}

// List returns events for a session in chronological order.
func (a *API) List(ctx context.Context, sessionID string, params *types.ListParams) (*types.PaginatedResponse[map[string]interface{}], error) {
	if err := qoderhttp.ValidateID(sessionID); err != nil {
		return nil, err
	}
	req := qoderhttp.ApplyListParams(a.client.GET("/sessions/"+sessionID+"/events"), params)
	var result types.PaginatedResponse[map[string]interface{}]
	if err := req.WithContext(ctx).Do(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Stream opens an SSE event stream for a session and returns the raw response.
// Use qoderhttp.NewSSEStream(resp) to parse the SSE events.
// The caller is responsible for closing the response body via SSEStream.Close().
//
// An optional lastEventID can be provided for SSE resumption. When set, the
// after_id query parameter is sent and the server replays events that occurred
// after the event with the given ID.
//
// Usage:
//
//	resp, err := client.Events().Stream(ctx, sessionID)
//	if err != nil { ... }
//	stream := qoderhttp.NewSSEStream(resp)
//	defer stream.Close()
//	for {
//	    evt, err := stream.Next(ctx)
//	    if err == io.EOF { break }
//	    ...
//	}
//
// Resume from a specific event:
//
//	resp, err := client.Events().Stream(ctx, sessionID, "evt_123")
func (a *API) Stream(ctx context.Context, sessionID string, lastEventID ...string) (*http.Response, error) {
	if err := qoderhttp.ValidateID(sessionID); err != nil {
		return nil, err
	}
	path := "/sessions/" + sessionID + "/events/stream"
	req := a.client.GET(path).
		WithHeader("Accept", "text/event-stream").
		WithContext(ctx)
	if len(lastEventID) > 0 && lastEventID[0] != "" {
		req = req.WithQuery("after_id", lastEventID[0])
	}
	return req.DoWithResponse()
}
