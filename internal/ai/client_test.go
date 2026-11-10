package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// sseServer replies to successive chat completion requests with the given
// pre-baked SSE bodies and records the decoded requests.
func sseServer(t *testing.T, replies ...string) (*httptest.Server, *[]chatRequest, *[]http.Header) {
	t.Helper()
	var requests []chatRequest
	var headers []http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}
		requests = append(requests, req)
		headers = append(headers, r.Header.Clone())

		reply := replies[min(len(requests), len(replies))-1]
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, reply)
	}))
	t.Cleanup(server.Close)
	return server, &requests, &headers
}

func sse(chunks ...string) string {
	var b strings.Builder
	for _, chunk := range chunks {
		b.WriteString("data: " + chunk + "\n\n")
	}
	b.WriteString("data: [DONE]\n\n")
	return b.String()
}

const toolCallReply = `{"choices":[{"delta":{"role":"assistant","reasoning_content":"need totals"}}]}` + "\x00" +
	`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"summarize_actions","arguments":"{\"type\":"}}]}}]}` + "\x00" +
	`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"expense\"}"}}]}}]}`

func TestRunExecutesToolCallsAndStreamsAnswer(t *testing.T) {
	server, requests, headers := sseServer(t,
		sse(strings.Split(toolCallReply, "\x00")...),
		sse(
			`{"choices":[{"delta":{"content":"You spent "}}]}`,
			`{"choices":[{"delta":{"content":"42.00"}}]}`,
			`{"choices":[],"usage":{"total_tokens":10}}`,
		),
	)

	client := NewClient(server.URL+"/v1/", "secret", "test-model")

	var text strings.Builder
	var toolNames, toolArgs []string
	err := client.Run(context.Background(), "session-1",
		[]Message{{Role: "user", Content: "how much?"}},
		[]Tool{{Type: "function", Function: FunctionDef{Name: "summarize_actions", Parameters: json.RawMessage(`{}`)}}},
		func(_ context.Context, name, arguments string) string {
			toolArgs = append(toolArgs, arguments)
			return `{"totals":[]}`
		},
		func(s string) { text.WriteString(s) },
		func(name string) { toolNames = append(toolNames, name) },
	)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if got := text.String(); got != "You spent 42.00" {
		t.Errorf("streamed text = %q", got)
	}
	if len(toolNames) != 1 || toolNames[0] != "summarize_actions" {
		t.Errorf("tool names = %v", toolNames)
	}
	if len(toolArgs) != 1 || toolArgs[0] != `{"type":"expense"}` {
		t.Errorf("tool arguments = %v", toolArgs)
	}

	if len(*requests) != 2 {
		t.Fatalf("expected 2 upstream requests, got %d", len(*requests))
	}
	first, second := (*requests)[0], (*requests)[1]
	if first.Model != "test-model" || !first.Stream || len(first.Tools) != 1 {
		t.Errorf("unexpected first request: %+v", first)
	}

	// The follow-up must carry the assistant's tool call (with its reasoning)
	// and the tool result.
	if len(second.Messages) != 3 {
		t.Fatalf("expected 3 messages in follow-up, got %d", len(second.Messages))
	}
	assistant, tool := second.Messages[1], second.Messages[2]
	if assistant.Role != "assistant" || assistant.ReasoningContent != "need totals" ||
		len(assistant.ToolCalls) != 1 || assistant.ToolCalls[0].ID != "call_1" {
		t.Errorf("unexpected assistant message: %+v", assistant)
	}
	if tool.Role != "tool" || tool.ToolCallID != "call_1" || tool.Content != `{"totals":[]}` {
		t.Errorf("unexpected tool message: %+v", tool)
	}

	h := (*headers)[0]
	if h.Get("Authorization") != "Bearer secret" {
		t.Errorf("Authorization header = %q", h.Get("Authorization"))
	}
	if h.Get("x-opencode-session") != "session-1" {
		t.Errorf("session header = %q", h.Get("x-opencode-session"))
	}
}

func TestRunStopsAfterTooManyRounds(t *testing.T) {
	server, requests, _ := sseServer(t, sse(strings.Split(toolCallReply, "\x00")...))
	client := NewClient(server.URL+"/v1", "secret", "test-model")

	err := client.Run(context.Background(), "", []Message{{Role: "user", Content: "loop"}}, nil,
		func(context.Context, string, string) string { return "{}" },
		func(string) {}, func(string) {},
	)
	if !errors.Is(err, ErrTooManyRounds) {
		t.Fatalf("expected ErrTooManyRounds, got %v", err)
	}
	if len(*requests) != maxToolRounds {
		t.Errorf("expected %d upstream requests, got %d", maxToolRounds, len(*requests))
	}
}

func TestRunReportsUpstreamErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"bad key"}`, http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "secret", "test-model")
	err := client.Run(context.Background(), "", []Message{{Role: "user", Content: "hi"}}, nil,
		func(context.Context, string, string) string { return "" },
		func(string) {}, func(string) {},
	)
	if err == nil || !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "bad key") {
		t.Fatalf("expected a 401 error with detail, got %v", err)
	}
}
