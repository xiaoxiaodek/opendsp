package ai

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/opendsp/opendsp/internal/biz"
)

type mockCampaignRepo struct{}

func (m *mockCampaignRepo) Create(ctx context.Context, c *biz.Campaign) error { return nil }
func (m *mockCampaignRepo) Get(ctx context.Context, id int64) (*biz.Campaign, error) {
	b := float64(1000)
	return &biz.Campaign{ID: id, Name: "Test Campaign", Budget: &b}, nil
}
func (m *mockCampaignRepo) Update(ctx context.Context, c *biz.Campaign) error { return nil }
func (m *mockCampaignRepo) UpdateStatus(ctx context.Context, id int64, status int16) error { return nil }
func (m *mockCampaignRepo) List(ctx context.Context, advertiserID int64, status *int16, page, pageSize int32) ([]biz.Campaign, int64, error) {
	b := float64(1000)
	return []biz.Campaign{{ID: 1, Name: "Test", Budget: &b, Status: 1}}, 1, nil
}

func TestToolRegistry_GetCampaign(t *testing.T) {
	r := &ToolRegistry{campaignRepo: &mockCampaignRepo{}}
	r.registerTools()

	args, _ := json.Marshal(map[string]any{"id": 1})
	result, err := r.Execute(context.Background(), "get_campaign", 1, 1, "admin", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty result")
	}
}

func TestToolRegistry_WriteToolViewerDenied(t *testing.T) {
	r := &ToolRegistry{campaignRepo: &mockCampaignRepo{}}
	r.registerTools()

	args, _ := json.Marshal(map[string]any{"id": 1, "status": 2})
	_, err := r.Execute(context.Background(), "update_campaign_status", 1, 1, "viewer", args)
	if err == nil {
		t.Fatal("expected permission denied error for viewer")
	}
}

func TestToolRegistry_UnknownTool(t *testing.T) {
	r := &ToolRegistry{}
	r.registerTools()

	_, err := r.Execute(context.Background(), "nonexistent_tool", 1, 1, "admin", nil)
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestToolRegistry_Definitions(t *testing.T) {
	r := &ToolRegistry{}
	r.registerTools()

	defs := r.Definitions()
	if len(defs) == 0 {
		t.Fatal("expected non-empty tool definitions")
	}
	for _, d := range defs {
		if d.Function.Name == "" {
			t.Fatal("tool definition missing name")
		}
	}
}
