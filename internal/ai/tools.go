package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/opendsp/opendsp/internal/biz"
)

type ToolHandler func(ctx context.Context, userID, advertiserID int64, role string, args json.RawMessage) (string, error)

type toolEntry struct {
	def     ToolDef
	handler ToolHandler
}

type ToolRegistry struct {
	tools          []toolEntry
	campaignRepo   biz.CampaignRepo
	adGroupRepo    biz.AdGroupRepo
	creativeRepo   biz.CreativeRepo
	reportRepo     biz.ReportRepo
	advertiserRepo biz.AdvertiserRepo
	balanceRepo    biz.BalanceRepo
	adminRepo      biz.AdminRepo
}

func NewToolRegistry(
	campaignRepo biz.CampaignRepo,
	adGroupRepo biz.AdGroupRepo,
	creativeRepo biz.CreativeRepo,
	reportRepo biz.ReportRepo,
	advertiserRepo biz.AdvertiserRepo,
	balanceRepo biz.BalanceRepo,
	adminRepo biz.AdminRepo,
) *ToolRegistry {
	r := &ToolRegistry{
		campaignRepo:   campaignRepo,
		adGroupRepo:    adGroupRepo,
		creativeRepo:   creativeRepo,
		reportRepo:     reportRepo,
		advertiserRepo: advertiserRepo,
		balanceRepo:    balanceRepo,
		adminRepo:      adminRepo,
	}
	r.registerTools()
	return r
}

func (r *ToolRegistry) registerTools() {
	r.tools = []toolEntry{
		r.makeTool("get_dashboard", "Get today's advertising dashboard summary including impressions, clicks, cost, CTR, balance, active campaigns and ad groups",
			map[string]any{
				"type":       "object",
				"properties": map[string]any{},
				"required":   []string{},
			}, r.handleGetDashboard),

		r.makeTool("get_report", "Get hourly advertising report data for a date range. Returns impressions, clicks, CTR, cost, CPM per hour.",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"campaign_id": map[string]any{"type": "integer", "description": "Optional campaign ID filter"},
					"adgroup_id":  map[string]any{"type": "integer", "description": "Optional ad group ID filter"},
					"start_time":  map[string]any{"type": "string", "description": "Start time in ISO 8601 format"},
					"end_time":    map[string]any{"type": "string", "description": "End time in ISO 8601 format"},
				},
				"required": []string{"start_time", "end_time"},
			}, r.handleGetReport),

		r.makeTool("list_campaigns", "List advertising campaigns with optional status filter",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"status": map[string]any{"type": "integer", "description": "Optional: 1=active, 2=paused"},
				},
			}, r.handleListCampaigns),

		r.makeTool("list_adgroups", "List ad groups, optionally filtered by campaign",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"campaign_id": map[string]any{"type": "integer", "description": "Optional campaign ID filter"},
				},
			}, r.handleListAdGroups),

		r.makeTool("list_creatives", "List creatives for an ad group",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"adgroup_id": map[string]any{"type": "integer", "description": "Ad group ID"},
				},
				"required": []string{"adgroup_id"},
			}, r.handleListCreatives),

		r.makeTool("get_campaign", "Get details of a single campaign",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "integer", "description": "Campaign ID"},
				},
				"required": []string{"id"},
			}, r.handleGetCampaign),

		r.makeTool("get_adgroup", "Get details of a single ad group",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "integer", "description": "Ad group ID"},
				},
				"required": []string{"id"},
			}, r.handleGetAdGroup),

		r.makeTool("get_balance", "Get advertiser account balance",
			map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			}, r.handleGetBalance),

		r.makeTool("list_transactions", "List recent balance transactions",
			map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			}, r.handleListTransactions),

		r.makeTool("update_campaign_budget", "Update a campaign's daily or total budget",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":           map[string]any{"type": "integer", "description": "Campaign ID"},
					"budget":       map[string]any{"type": "number", "description": "Total budget in yuan"},
					"daily_budget": map[string]any{"type": "number", "description": "Daily budget in yuan"},
				},
				"required": []string{"id"},
			}, r.handleUpdateCampaignBudget),

		r.makeTool("update_adgroup_bid", "Update an ad group's bid price",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":        map[string]any{"type": "integer", "description": "Ad group ID"},
					"bid_price": map[string]any{"type": "number", "description": "New bid price in yuan"},
				},
				"required": []string{"id", "bid_price"},
			}, r.handleUpdateAdGroupBid),

		r.makeTool("update_adgroup_status", "Enable or disable an ad group",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":     map[string]any{"type": "integer", "description": "Ad group ID"},
					"status": map[string]any{"type": "integer", "description": "1=active, 2=paused"},
				},
				"required": []string{"id", "status"},
			}, r.handleUpdateAdGroupStatus),

		r.makeTool("update_campaign_status", "Enable or disable a campaign",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":     map[string]any{"type": "integer", "description": "Campaign ID"},
					"status": map[string]any{"type": "integer", "description": "1=active, 2=paused"},
				},
				"required": []string{"id", "status"},
			}, r.handleUpdateCampaignStatus),

		r.makeTool("list_audit_queue", "List pending creative and advertiser audits",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"audit_type": map[string]any{"type": "integer", "description": "Optional: 1=creative, 2=advertiser"},
				},
			}, r.handleListAuditQueue),

		r.makeTool("audit_creative", "Approve or reject a creative",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":           map[string]any{"type": "integer", "description": "Creative ID"},
					"audit_status": map[string]any{"type": "integer", "description": "1=approved, 2=rejected"},
					"reason":       map[string]any{"type": "string", "description": "Audit reason"},
				},
				"required": []string{"id", "audit_status"},
			}, r.handleAuditCreative),
	}
}

func (r *ToolRegistry) makeTool(name, desc string, params map[string]any, handler ToolHandler) toolEntry {
	return toolEntry{
		def: ToolDef{
			Type: "function",
			Function: struct {
				Name        string         `json:"name"`
				Description string         `json:"description"`
				Parameters  map[string]any `json:"parameters"`
			}{Name: name, Description: desc, Parameters: params},
		},
		handler: handler,
	}
}

func (r *ToolRegistry) Definitions() []ToolDef {
	defs := make([]ToolDef, len(r.tools))
	for i, t := range r.tools {
		defs[i] = t.def
	}
	return defs
}

func (r *ToolRegistry) Execute(ctx context.Context, name string, userID, advertiserID int64, role string, args json.RawMessage) (string, error) {
	for _, t := range r.tools {
		if t.def.Function.Name == name {
			return t.handler(ctx, userID, advertiserID, role, args)
		}
	}
	return "", fmt.Errorf("unknown tool: %s", name)
}

func sanitize(s string) string {
	s = strings.ReplaceAll(s, "@", "[at]")
	return s
}

func (r *ToolRegistry) handleGetDashboard(ctx context.Context, userID, advertiserID int64, role string, args json.RawMessage) (string, error) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	reports, err := r.reportRepo.Query(ctx, advertiserID, nil, nil, todayStart, now)
	if err != nil {
		return "", fmt.Errorf("query dashboard: %w", err)
	}
	var imp, clicks int64
	var cost float64
	for _, rpt := range reports {
		imp += rpt.Impressions
		clicks += rpt.Clicks
		cost += rpt.Cost
	}
	ctr := 0.0
	if imp > 0 {
		ctr = float64(clicks) / float64(imp) * 100
	}
	balance, _, _ := r.balanceRepo.GetBalance(ctx, advertiserID)
	return fmt.Sprintf(`{"today_impressions":%d,"today_clicks":%d,"today_cost":%.2f,"today_ctr":%.2f,"balance":%.2f}`,
		imp, clicks, cost, ctr, balance), nil
}

func (r *ToolRegistry) handleGetReport(ctx context.Context, userID, advertiserID int64, role string, args json.RawMessage) (string, error) {
	var params struct {
		CampaignID *int64 `json:"campaign_id"`
		AdGroupID  *int64 `json:"adgroup_id"`
		StartTime  string `json:"start_time"`
		EndTime    string `json:"end_time"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}
	startTime, err := time.Parse(time.RFC3339, params.StartTime)
	if err != nil {
		return "", fmt.Errorf("parse start_time: %w", err)
	}
	endTime, err := time.Parse(time.RFC3339, params.EndTime)
	if err != nil {
		return "", fmt.Errorf("parse end_time: %w", err)
	}
	reports, err := r.reportRepo.Query(ctx, advertiserID, params.CampaignID, params.AdGroupID, startTime, endTime)
	if err != nil {
		return "", fmt.Errorf("query report: %w", err)
	}
	type row struct {
		Hour        string  `json:"hour"`
		Impressions int64   `json:"impressions"`
		Clicks      int64   `json:"clicks"`
		CTR         float64 `json:"ctr"`
		Cost        float64 `json:"cost"`
		CPM         float64 `json:"cpm"`
	}
	rows := make([]row, 0, len(reports))
	for _, rpt := range reports {
		ctr := 0.0
		if rpt.Impressions > 0 {
			ctr = float64(rpt.Clicks) / float64(rpt.Impressions) * 100
		}
		cpm := 0.0
		if rpt.Impressions > 0 {
			cpm = rpt.Cost / float64(rpt.Impressions) * 1000
		}
		rows = append(rows, row{
			Hour: rpt.Hour.Format(time.RFC3339), Impressions: rpt.Impressions,
			Clicks: rpt.Clicks, CTR: ctr, Cost: rpt.Cost, CPM: cpm,
		})
	}
	b, _ := json.Marshal(rows)
	return string(b), nil
}

func (r *ToolRegistry) handleListCampaigns(ctx context.Context, userID, advertiserID int64, role string, args json.RawMessage) (string, error) {
	var params struct {
		Status *int16 `json:"status"`
	}
	json.Unmarshal(args, &params)
	campaigns, _, err := r.campaignRepo.List(ctx, advertiserID, params.Status, 1, 50)
	if err != nil {
		return "", fmt.Errorf("list campaigns: %w", err)
	}
	type item struct {
		ID          int64    `json:"id"`
		Name        string   `json:"name"`
		Budget      *float64 `json:"budget"`
		DailyBudget *float64 `json:"daily_budget"`
		Status      int16    `json:"status"`
	}
	items := make([]item, len(campaigns))
	for i, c := range campaigns {
		items[i] = item{ID: c.ID, Name: c.Name, Budget: c.Budget, DailyBudget: c.DailyBudget, Status: c.Status}
	}
	b, _ := json.Marshal(items)
	return string(b), nil
}

func (r *ToolRegistry) handleListAdGroups(ctx context.Context, userID, advertiserID int64, role string, args json.RawMessage) (string, error) {
	var params struct {
		CampaignID int64 `json:"campaign_id"`
	}
	json.Unmarshal(args, &params)
	adGroups, _, err := r.adGroupRepo.List(ctx, params.CampaignID, nil, 1, 50)
	if err != nil {
		return "", fmt.Errorf("list adgroups: %w", err)
	}
	type item struct {
		ID          int64    `json:"id"`
		CampaignID  int64    `json:"campaign_id"`
		Name        string   `json:"name"`
		BidPrice    float64  `json:"bid_price"`
		DailyBudget *float64 `json:"daily_budget"`
		Status      int16    `json:"status"`
	}
	items := make([]item, len(adGroups))
	for i, ag := range adGroups {
		items[i] = item{ID: ag.ID, CampaignID: ag.CampaignID, Name: ag.Name, BidPrice: ag.BidPrice, DailyBudget: ag.DailyBudget, Status: ag.Status}
	}
	b, _ := json.Marshal(items)
	return string(b), nil
}

func (r *ToolRegistry) handleListCreatives(ctx context.Context, userID, advertiserID int64, role string, args json.RawMessage) (string, error) {
	var params struct {
		AdGroupID int64 `json:"adgroup_id"`
	}
	json.Unmarshal(args, &params)
	creatives, _, err := r.creativeRepo.ListByAdGroup(ctx, params.AdGroupID, 1, 50)
	if err != nil {
		return "", fmt.Errorf("list creatives: %w", err)
	}
	type item struct {
		ID          int64  `json:"id"`
		Name        string `json:"name"`
		AuditStatus int16  `json:"audit_status"`
	}
	items := make([]item, len(creatives))
	for i, cr := range creatives {
		items[i] = item{ID: cr.ID, Name: cr.Name, AuditStatus: cr.AuditStatus}
	}
	b, _ := json.Marshal(items)
	return string(b), nil
}

func (r *ToolRegistry) handleGetCampaign(ctx context.Context, userID, advertiserID int64, role string, args json.RawMessage) (string, error) {
	var params struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}
	c, err := r.campaignRepo.Get(ctx, params.ID)
	if err != nil {
		return "", fmt.Errorf("get campaign: %w", err)
	}
	b, _ := json.Marshal(map[string]any{
		"id": c.ID, "name": c.Name, "budget": c.Budget, "daily_budget": c.DailyBudget,
		"status": c.Status, "pacing": c.Pacing,
	})
	return string(b), nil
}

func (r *ToolRegistry) handleGetAdGroup(ctx context.Context, userID, advertiserID int64, role string, args json.RawMessage) (string, error) {
	var params struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}
	ag, err := r.adGroupRepo.Get(ctx, params.ID)
	if err != nil {
		return "", fmt.Errorf("get adgroup: %w", err)
	}
	b, _ := json.Marshal(map[string]any{
		"id": ag.ID, "campaign_id": ag.CampaignID, "name": ag.Name,
		"bid_price": ag.BidPrice, "daily_budget": ag.DailyBudget, "status": ag.Status,
	})
	return string(b), nil
}

func (r *ToolRegistry) handleGetBalance(ctx context.Context, userID, advertiserID int64, role string, args json.RawMessage) (string, error) {
	balance, frozen, err := r.balanceRepo.GetBalance(ctx, advertiserID)
	if err != nil {
		return "", fmt.Errorf("get balance: %w", err)
	}
	b, _ := json.Marshal(map[string]float64{"balance": balance, "frozen": frozen})
	return string(b), nil
}

func (r *ToolRegistry) handleListTransactions(ctx context.Context, userID, advertiserID int64, role string, args json.RawMessage) (string, error) {
	txs, _, err := r.balanceRepo.ListTransactions(ctx, advertiserID, 1, 10)
	if err != nil {
		return "", fmt.Errorf("list transactions: %w", err)
	}
	type item struct {
		Amount        float64 `json:"amount"`
		BalanceBefore float64 `json:"balance_before"`
		BalanceAfter  float64 `json:"balance_after"`
		Description   *string `json:"description"`
		CreatedAt     string  `json:"created_at"`
	}
	items := make([]item, len(txs))
	for i, tx := range txs {
		items[i] = item{
			Amount: tx.Amount, BalanceBefore: tx.BalanceBefore, BalanceAfter: tx.BalanceAfter,
			Description: tx.Description, CreatedAt: tx.CreatedAt.Format(time.RFC3339),
		}
	}
	b, _ := json.Marshal(items)
	return string(b), nil
}

func (r *ToolRegistry) handleUpdateCampaignBudget(ctx context.Context, userID, advertiserID int64, role string, args json.RawMessage) (string, error) {
	if role == "viewer" {
		return "", fmt.Errorf("permission denied: viewer cannot modify campaigns")
	}
	var params struct {
		ID          int64    `json:"id"`
		Budget      *float64 `json:"budget"`
		DailyBudget *float64 `json:"daily_budget"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}
	c, err := r.campaignRepo.Get(ctx, params.ID)
	if err != nil {
		return "", fmt.Errorf("get campaign: %w", err)
	}
	if params.Budget != nil {
		c.Budget = params.Budget
	}
	if params.DailyBudget != nil {
		c.DailyBudget = params.DailyBudget
	}
	if err := r.campaignRepo.Update(ctx, c); err != nil {
		return "", fmt.Errorf("update campaign: %w", err)
	}
	return fmt.Sprintf(`{"success":true,"message":"Campaign %d budget updated"}`, params.ID), nil
}

func (r *ToolRegistry) handleUpdateAdGroupBid(ctx context.Context, userID, advertiserID int64, role string, args json.RawMessage) (string, error) {
	if role == "viewer" {
		return "", fmt.Errorf("permission denied: viewer cannot modify ad groups")
	}
	var params struct {
		ID       int64   `json:"id"`
		BidPrice float64 `json:"bid_price"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}
	ag, err := r.adGroupRepo.Get(ctx, params.ID)
	if err != nil {
		return "", fmt.Errorf("get adgroup: %w", err)
	}
	ag.BidPrice = params.BidPrice
	if err := r.adGroupRepo.Update(ctx, ag); err != nil {
		return "", fmt.Errorf("update adgroup: %w", err)
	}
	return fmt.Sprintf(`{"success":true,"message":"Ad group %d bid price updated to ¥%.2f"}`, params.ID, params.BidPrice), nil
}

func (r *ToolRegistry) handleUpdateAdGroupStatus(ctx context.Context, userID, advertiserID int64, role string, args json.RawMessage) (string, error) {
	if role == "viewer" {
		return "", fmt.Errorf("permission denied: viewer cannot modify ad groups")
	}
	var params struct {
		ID     int64 `json:"id"`
		Status int16 `json:"status"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}
	if err := r.adGroupRepo.UpdateStatus(ctx, params.ID, params.Status); err != nil {
		return "", fmt.Errorf("update adgroup status: %w", err)
	}
	statusStr := "enabled"
	if params.Status == 2 {
		statusStr = "paused"
	}
	return fmt.Sprintf(`{"success":true,"message":"Ad group %d %s"}`, params.ID, statusStr), nil
}

func (r *ToolRegistry) handleUpdateCampaignStatus(ctx context.Context, userID, advertiserID int64, role string, args json.RawMessage) (string, error) {
	if role == "viewer" {
		return "", fmt.Errorf("permission denied: viewer cannot modify campaigns")
	}
	var params struct {
		ID     int64 `json:"id"`
		Status int16 `json:"status"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}
	if err := r.campaignRepo.UpdateStatus(ctx, params.ID, params.Status); err != nil {
		return "", fmt.Errorf("update campaign status: %w", err)
	}
	statusStr := "enabled"
	if params.Status == 2 {
		statusStr = "paused"
	}
	return fmt.Sprintf(`{"success":true,"message":"Campaign %d %s"}`, params.ID, statusStr), nil
}

func (r *ToolRegistry) handleListAuditQueue(ctx context.Context, userID, advertiserID int64, role string, args json.RawMessage) (string, error) {
	if role != "admin" {
		return "", fmt.Errorf("permission denied: admin only")
	}
	var params struct {
		AuditType *int32 `json:"audit_type"`
	}
	json.Unmarshal(args, &params)
	audits, _, err := r.adminRepo.ListPendingAudits(ctx, params.AuditType, 1, 20)
	if err != nil {
		return "", fmt.Errorf("list audits: %w", err)
	}
	type item struct {
		ID             int64  `json:"id"`
		AuditType      int32  `json:"audit_type"`
		Name           string `json:"name"`
		AdvertiserName string `json:"advertiser_name"`
		Status         int16  `json:"status"`
	}
	items := make([]item, len(audits))
	for i, a := range audits {
		items[i] = item{ID: a.ID, AuditType: a.AuditType, Name: a.Name, AdvertiserName: a.AdvertiserName, Status: a.Status}
	}
	b, _ := json.Marshal(items)
	return string(b), nil
}

func (r *ToolRegistry) handleAuditCreative(ctx context.Context, userID, advertiserID int64, role string, args json.RawMessage) (string, error) {
	if role != "admin" {
		return "", fmt.Errorf("permission denied: admin only")
	}
	var params struct {
		ID          int64  `json:"id"`
		AuditStatus int16  `json:"audit_status"`
		Reason      string `json:"reason"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}
	if err := r.creativeRepo.UpdateAuditStatus(ctx, params.ID, params.AuditStatus, params.Reason); err != nil {
		return "", fmt.Errorf("audit creative: %w", err)
	}
	statusStr := "approved"
	if params.AuditStatus == 2 {
		statusStr = "rejected"
	}
	return fmt.Sprintf(`{"success":true,"message":"Creative %d %s"}`, params.ID, statusStr), nil
}
