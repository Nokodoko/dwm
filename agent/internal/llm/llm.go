// Package llm speaks the OpenAI chat-completions dialect, which both vLLM
// (Laguna on monty) and llama.cpp (the offline fallback on lewis) implement.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Role values for a chat message.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

// Message is one turn in the conversation.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

// ToolCall is a model request to invoke one skill.
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"` // JSON-encoded object
	} `json:"function"`
}

// Tool is a skill advertised to the model.
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction describes one callable skill in JSON Schema terms.
type ToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

// Backend is one hot-swappable model endpoint.
type Backend struct {
	Name    string // short handle used by /model
	BaseURL string // e.g. http://monty:8094/v1
	Model   string // model id the server expects
	APIKey  string // optional; vLLM and llama.cpp accept any value
	Note    string // shown in /model listings
}

// Client talks to whichever Backend is currently selected.
type Client struct {
	backend Backend
	http    *http.Client
}

// New builds a client for a backend.
func New(b Backend) *Client {
	return &Client{
		backend: b,
		// Local inference on a cold cache can be slow; a short timeout would
		// abort valid generations, so allow a generous ceiling.
		http: &http.Client{Timeout: 180 * time.Second},
	}
}

// Backend reports the active backend.
func (c *Client) Backend() Backend { return c.backend }

// SetBackend hot-swaps the endpoint. Conversation state lives in the caller,
// so a swap mid-conversation keeps the history and only changes who answers.
func (c *Client) SetBackend(b Backend) { c.backend = b }

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools,omitempty"`
	ToolChoice  string    `json:"tool_choice,omitempty"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// Chat sends the conversation and returns the assistant message, which may
// carry tool calls instead of prose.
func (c *Client) Chat(ctx context.Context, msgs []Message, tools []Tool) (*Message, error) {
	req := chatRequest{
		Model:       c.backend.Model,
		Messages:    msgs,
		Tools:       tools,
		Temperature: 0.2, // window management should be predictable, not creative
		MaxTokens:   2048,
	}
	if len(tools) > 0 {
		req.ToolChoice = "auto"
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("llm: marshal request: %w", err)
	}

	url := strings.TrimSuffix(c.backend.BaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.backend.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.backend.APIKey)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("llm: %s unreachable: %w", c.backend.Name, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("llm: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("llm: %s returned %s: %s",
			c.backend.Name, resp.Status, truncate(string(data), 300))
	}

	var cr chatResponse
	if err := json.Unmarshal(data, &cr); err != nil {
		return nil, fmt.Errorf("llm: decode response: %w", err)
	}
	if cr.Error != nil {
		return nil, fmt.Errorf("llm: %s: %s", c.backend.Name, cr.Error.Message)
	}
	if len(cr.Choices) == 0 {
		return nil, fmt.Errorf("llm: %s returned no choices", c.backend.Name)
	}
	return &cr.Choices[0].Message, nil
}

// Probe reports whether the backend answers, for the status line and for
// automatic fallback when monty is off-network.
func (c *Client) Probe(ctx context.Context) error {
	url := strings.TrimSuffix(c.backend.BaseURL, "/") + "/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", resp.Status)
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
