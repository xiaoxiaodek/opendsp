package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	Name       string     `json:"name,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type ToolDef struct {
	Type     string `json:"type"`
	Function struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Parameters  map[string]any `json:"parameters"`
	} `json:"function"`
}

type StreamEvent struct {
	Delta    string
	ToolCall *ToolCall
	Done     bool
	Error    error
}

type LLMClient interface {
	Chat(ctx context.Context, messages []Message, tools []ToolDef) (<-chan StreamEvent, error)
}

type openaiClient struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

func NewLLMClient() LLMClient {
	apiKey := os.Getenv("LLM_API_KEY")
	baseURL := os.Getenv("LLM_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &openaiClient{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

type chatReq struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Tools    []ToolDef `json:"tools,omitempty"`
	Stream   bool      `json:"stream"`
}

type chatStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

func (c *openaiClient) Chat(ctx context.Context, messages []Message, tools []ToolDef) (<-chan StreamEvent, error) {
	body := chatReq{
		Model:    c.model,
		Messages: messages,
		Tools:    tools,
		Stream:   true,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("llm api error %d: %s", resp.StatusCode, string(body))
	}

	ch := make(chan StreamEvent, 10)
	go func() {
		defer resp.Body.Close()
		defer close(ch)

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		var toolCalls map[int]*ToolCall

		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				ch <- StreamEvent{Done: true}
				return
			}

			var chunk chatStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			var finishReason string
			for _, choice := range chunk.Choices {
				if choice.FinishReason != "" {
					finishReason = choice.FinishReason
				}
				if choice.Delta.Content != "" {
					ch <- StreamEvent{Delta: choice.Delta.Content}
				}
				for _, tc := range choice.Delta.ToolCalls {
					if toolCalls == nil {
						toolCalls = make(map[int]*ToolCall)
					}
					if _, ok := toolCalls[tc.Index]; !ok {
						toolCalls[tc.Index] = &ToolCall{
							ID:   tc.ID,
							Type: "function",
						}
						toolCalls[tc.Index].Function.Name = tc.Function.Name
					}
					if tc.ID != "" {
						toolCalls[tc.Index].ID = tc.ID
					}
					toolCalls[tc.Index].Function.Arguments += tc.Function.Arguments
				}
			}

			if finishReason == "tool_calls" && toolCalls != nil {
				keys := make([]int, 0, len(toolCalls))
				for k := range toolCalls {
					keys = append(keys, k)
				}
				sort.Ints(keys)
				for _, k := range keys {
					ch <- StreamEvent{ToolCall: toolCalls[k]}
				}
				ch <- StreamEvent{Done: true}
				return
			}

			if finishReason == "stop" || finishReason == "length" || finishReason == "content_filter" {
				ch <- StreamEvent{Done: true}
				return
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- StreamEvent{Error: fmt.Errorf("stream read error: %w", err)}
			return
		}
		ch <- StreamEvent{Done: true}
	}()

	return ch, nil
}
