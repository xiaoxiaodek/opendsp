package ai

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"
)

var (
	aiChatRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "admanager_ai_chat_requests_total",
		Help: "Total AI chat requests.",
	}, []string{"role"})

	aiChatLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "admanager_ai_chat_latency_seconds",
		Help:    "AI chat request latency in seconds.",
		Buckets: []float64{.1, .5, 1, 2.5, 5, 10, 30},
	}, []string{"role"})

	aiToolCalls = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "admanager_ai_tool_calls_total",
		Help: "Total AI tool calls.",
	}, []string{"tool", "status"})

	aiLLMCalls = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "admanager_ai_llm_calls_total",
		Help: "Total LLM API calls.",
	}, []string{"status"})

	aiLLMLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "admanager_ai_llm_latency_seconds",
		Help:    "LLM API call latency in seconds.",
		Buckets: []float64{.1, .5, 1, 2.5, 5, 10, 30},
	})

	aiActiveSessions = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "admanager_ai_active_sessions",
		Help: "Current number of active AI chat sessions.",
	})
)

type ChatService struct {
	llm   LLMClient
	tools *ToolRegistry
	rdb   *redis.Client
	mu    sync.Mutex
}

func NewChatService(llm LLMClient, tools *ToolRegistry, rdb *redis.Client) *ChatService {
	return &ChatService{llm: llm, tools: tools, rdb: rdb}
}

type ChatSession struct {
	Messages       []Message       `json:"messages"`
	UserID         int64           `json:"user_id"`
	AdvertiserID   int64           `json:"advertiser_id"`
	Role           string          `json:"role"`
	PendingConfirm *PendingConfirm `json:"pending_confirm,omitempty"`
}

type PendingConfirm struct {
	ToolCallID string          `json:"tool_call_id"`
	ToolName   string          `json:"tool_name"`
	Args       json.RawMessage `json:"args"`
}

func (s *ChatService) NewSession(userID, advertiserID int64, role string) (string, *ChatSession) {
	id := generateSessionID()
	sess := &ChatSession{
		UserID:       userID,
		AdvertiserID: advertiserID,
		Role:         role,
		Messages: []Message{
			{Role: "system", Content: systemPrompt(userID, advertiserID, role)},
		},
	}
	s.saveSession(id, sess)
	aiActiveSessions.Inc()
	return id, sess
}

func (s *ChatService) GetSession(id string) (*ChatSession, error) {
	data, err := s.rdb.Get(context.Background(), "ai:chat:"+id).Result()
	if err != nil {
		return nil, fmt.Errorf("session not found")
	}
	var sess ChatSession
	if err := json.Unmarshal([]byte(data), &sess); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	return &sess, nil
}

func (s *ChatService) saveSession(id string, sess *ChatSession) {
	data, _ := json.Marshal(sess)
	s.rdb.Set(context.Background(), "ai:chat:"+id, data, 30*time.Minute)
}

func (s *ChatService) deleteSession(id string) {
	s.rdb.Del(context.Background(), "ai:chat:"+id)
	aiActiveSessions.Dec()
}

func (s *ChatService) Chat(ctx context.Context, sessionID, userMessage string) (<-chan StreamEvent, error) {
	sess, err := s.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	aiChatRequests.WithLabelValues(sess.Role).Inc()

	sess.Messages = append(sess.Messages, Message{Role: "user", Content: userMessage})
	s.saveSession(sessionID, sess)

	ch := make(chan StreamEvent, 20)
	go s.runConversation(ctx, sessionID, sess, ch)
	return ch, nil
}

func (s *ChatService) ConfirmTool(ctx context.Context, sessionID, toolCallID string, confirmed bool) (<-chan StreamEvent, error) {
	sess, err := s.GetSession(sessionID)
	if err != nil {
		return nil, err
	}
	if sess.PendingConfirm == nil || sess.PendingConfirm.ToolCallID != toolCallID {
		return nil, fmt.Errorf("no pending confirmation for tool call %s", toolCallID)
	}

	ch := make(chan StreamEvent, 20)

	if !confirmed {
		sess.Messages = append(sess.Messages, Message{
			Role: "tool", ToolCallID: toolCallID,
			Content: `{"cancelled":true,"message":"User cancelled the operation"}`,
		})
		sess.PendingConfirm = nil
		s.saveSession(sessionID, sess)
		go s.runConversation(ctx, sessionID, sess, ch)
		return ch, nil
	}

	pc := sess.PendingConfirm
	sess.PendingConfirm = nil

	result, err := s.tools.Execute(ctx, pc.ToolName, sess.UserID, sess.AdvertiserID, sess.Role, pc.Args)
	if err != nil {
		result = fmt.Sprintf(`{"error":"%s"}`, err.Error())
	}
	AuditLog(sess.UserID, sess.AdvertiserID, sess.Role, pc.ToolName, pc.Args, result, err)

	sess.Messages = append(sess.Messages, Message{
		Role: "tool", ToolCallID: toolCallID, Content: result,
	})
	s.saveSession(sessionID, sess)
	go s.runConversation(ctx, sessionID, sess, ch)
	return ch, nil
}

func (s *ChatService) runConversation(ctx context.Context, sessionID string, sess *ChatSession, ch chan<- StreamEvent) {
	defer close(ch)

	maxSteps := 5
	for step := 0; step < maxSteps; step++ {
		select {
		case <-ctx.Done():
			ch <- StreamEvent{Error: ctx.Err()}
			return
		default:
		}

		llmStart := time.Now()
		eventCh, err := s.llm.Chat(ctx, sess.Messages, s.tools.Definitions())
		if err != nil {
			aiLLMCalls.WithLabelValues("error").Inc()
			ch <- StreamEvent{Error: fmt.Errorf("LLM error: %w", err)}
			return
		}

		var toolCalls []ToolCall
		hasContent := false

		for event := range eventCh {
			if event.Error != nil {
				ch <- event
				return
			}
			if event.Delta != "" {
				hasContent = true
				ch <- event
			}
			if event.ToolCall != nil {
				toolCalls = append(toolCalls, *event.ToolCall)
			}
			if event.Done {
				break
			}
		}

		aiLLMLatency.Observe(time.Since(llmStart).Seconds())
		aiLLMCalls.WithLabelValues("ok").Inc()

		if len(toolCalls) == 0 {
			if hasContent {
				ch <- StreamEvent{Done: true}
			}
			return
		}

		assistantMsg := Message{Role: "assistant", ToolCalls: toolCalls}
		sess.Messages = append(sess.Messages, assistantMsg)

		for _, tc := range toolCalls {
			if s.isWriteTool(tc.Function.Name) {
				var args json.RawMessage
				if tc.Function.Arguments != "" {
					args = json.RawMessage(tc.Function.Arguments)
				}
				sess.PendingConfirm = &PendingConfirm{
					ToolCallID: tc.ID,
					ToolName:   tc.Function.Name,
					Args:       args,
				}
				s.saveSession(sessionID, sess)

				confirmEvent := StreamEvent{
					Delta: fmt.Sprintf(`{"action":"confirm_required","tool_call_id":"%s","tool":"%s","args":%s}`,
						tc.ID, tc.Function.Name, tc.Function.Arguments),
				}
				ch <- confirmEvent
				ch <- StreamEvent{Done: true}
				return
			}

			result, err := s.tools.Execute(ctx, tc.Function.Name, sess.UserID, sess.AdvertiserID, sess.Role, json.RawMessage(tc.Function.Arguments))
			toolStatus := "ok"
			if err != nil {
				toolStatus = "error"
				result = fmt.Sprintf(`{"error":"%s"}`, err.Error())
			}
			aiToolCalls.WithLabelValues(tc.Function.Name, toolStatus).Inc()
			AuditLog(sess.UserID, sess.AdvertiserID, sess.Role, tc.Function.Name, json.RawMessage(tc.Function.Arguments), result, err)
			sess.Messages = append(sess.Messages, Message{
				Role: "tool", ToolCallID: tc.ID, Content: result,
			})
		}
		s.saveSession(sessionID, sess)
	}

	ch <- StreamEvent{Delta: "(I've reached the maximum number of steps. Please try a more specific question.)"}
	ch <- StreamEvent{Done: true}
}

func (s *ChatService) isWriteTool(name string) bool {
	writeTools := map[string]bool{
		"update_campaign_budget": true,
		"update_adgroup_bid":     true,
		"update_adgroup_status":  true,
		"update_campaign_status": true,
		"audit_creative":         true,
	}
	return writeTools[name]
}

func generateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func AuditLog(userID, advertiserID int64, role, toolName string, args json.RawMessage, result string, err error) {
	status := "ok"
	if err != nil {
		status = "error: " + err.Error()
	}
	if len(result) > 500 {
		result = result[:500] + "..."
	}
	log.Printf("[AI-AUDIT] user=%d advertiser=%d role=%s tool=%s args=%s result=%s status=%s",
		userID, advertiserID, role, toolName, string(args), result, status)
}
