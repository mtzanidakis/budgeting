// Package ai is a minimal streaming client for OpenAI-compatible chat
// completions APIs, with a tool-calling loop on top.
package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mtzanidakis/budgeting/internal/version"
)

// maxToolRounds bounds how many times the model may call tools before it has
// to produce a final answer.
const maxToolRounds = 6

// ErrTooManyRounds is returned by Run when the model keeps calling tools
// without settling on an answer.
var ErrTooManyRounds = errors.New("model exceeded the tool-call round limit")

// Message is a chat message. Content is a string, or a []ContentPart for
// user messages that carry images.
type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
	// ReasoningContent is a non-standard field emitted by thinking models
	// (e.g. DeepSeek). Such providers expect it to be echoed back alongside
	// the tool calls it led to, so it is preserved within a tool loop.
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
}

type ContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Tool struct {
	Type     string      `json:"type"`
	Function FunctionDef `json:"function"`
}

type FunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ToolExecutor runs one tool call and returns the result handed back to the
// model. Failures are reported in the result so the model can correct itself.
type ToolExecutor func(ctx context.Context, name, arguments string) string

type Client struct {
	baseURL string
	apiKey  string
	model   string
	http    *http.Client
}

func NewClient(baseURL, apiKey, model string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		// No overall timeout: responses are streamed, so the caller's
		// context bounds the request instead.
		http: &http.Client{
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				ResponseHeaderTimeout: 90 * time.Second,
				IdleConnTimeout:       90 * time.Second,
			},
		},
	}
}

// Run drives a conversation to completion: it streams the model's reply,
// executes any tool calls it makes and feeds the results back, until the
// model answers without calling tools. onText receives answer text as it
// arrives; onTool is called before each tool execution.
func (c *Client) Run(ctx context.Context, sessionID string, messages []Message, tools []Tool, exec ToolExecutor, onText func(string), onTool func(name string)) error {
	wroteText := false
	for round := 0; round < maxToolRounds; round++ {
		roundStarted := false
		reply, err := c.stream(ctx, sessionID, messages, tools, func(text string) {
			// Separate text that the model emits on either side of a tool call.
			if !roundStarted && wroteText {
				onText("\n\n")
			}
			roundStarted = true
			wroteText = true
			onText(text)
		})
		if err != nil {
			return err
		}
		if len(reply.ToolCalls) == 0 {
			return nil
		}

		messages = append(messages, reply)
		for _, call := range reply.ToolCalls {
			onTool(call.Function.Name)
			messages = append(messages, Message{
				Role:       "tool",
				ToolCallID: call.ID,
				Content:    exec(ctx, call.Function.Name, call.Function.Arguments),
			})
		}
	}
	return ErrTooManyRounds
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Tools    []Tool    `json:"tools,omitempty"`
	Stream   bool      `json:"stream"`
}

type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
			ToolCalls        []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
	} `json:"choices"`
}

// stream performs one chat completion request and assembles the streamed
// assistant message, forwarding answer text to onText as it arrives.
func (c *Client) stream(ctx context.Context, sessionID string, messages []Message, tools []Tool, onText func(string)) (Message, error) {
	body, err := json.Marshal(chatRequest{Model: c.model, Messages: messages, Tools: tools, Stream: true})
	if err != nil {
		return Message{}, fmt.Errorf("failed to encode chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Message{}, fmt.Errorf("failed to build chat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", "budgeting/"+version.Get())
	if sessionID != "" {
		// Required by OpenCode's gateway to route a conversation
		// consistently; ignored by other providers.
		req.Header.Set("x-opencode-session", sessionID)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return Message{}, fmt.Errorf("failed to send chat request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return Message{}, fmt.Errorf("chat request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(detail)))
	}

	var content, reasoning strings.Builder
	var calls []ToolCall

	reader := bufio.NewReader(resp.Body)
	for {
		line, readErr := reader.ReadString('\n')
		if data, ok := strings.CutPrefix(strings.TrimSpace(line), "data:"); ok {
			data = strings.TrimSpace(data)
			if data == "[DONE]" {
				break
			}

			var chunk streamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				return Message{}, fmt.Errorf("failed to decode chat stream: %w", err)
			}
			for _, choice := range chunk.Choices {
				delta := choice.Delta
				reasoning.WriteString(delta.ReasoningContent)
				if delta.Content != "" {
					content.WriteString(delta.Content)
					onText(delta.Content)
				}
				for _, tc := range delta.ToolCalls {
					if tc.Index < 0 || tc.Index > 64 {
						return Message{}, fmt.Errorf("failed to decode chat stream: tool call index %d out of range", tc.Index)
					}
					for len(calls) <= tc.Index {
						calls = append(calls, ToolCall{Type: "function"})
					}
					call := &calls[tc.Index]
					if tc.ID != "" {
						call.ID = tc.ID
					}
					call.Function.Name += tc.Function.Name
					call.Function.Arguments += tc.Function.Arguments
				}
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return Message{}, fmt.Errorf("failed to read chat stream: %w", readErr)
		}
	}

	return Message{
		Role:             "assistant",
		Content:          content.String(),
		ReasoningContent: reasoning.String(),
		ToolCalls:        calls,
	}, nil
}
