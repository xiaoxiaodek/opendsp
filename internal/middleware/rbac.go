package middleware

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PermissionChecker interface {
	HasPermission(ctx context.Context, userID int64, resource, action string) bool
}

type RBACInterceptor struct {
	checker PermissionChecker
	rules   map[string][]string
}

func NewRBACInterceptor(checker PermissionChecker) *RBACInterceptor {
	return &RBACInterceptor{
		checker: checker,
		rules:   defaultRules(),
	}
}

func defaultRules() map[string][]string {
	return map[string][]string{
		"/admanager.v1.AdManager/CreateCampaign":       {"admin", "operator"},
		"/admanager.v1.AdManager/UpdateCampaign":       {"admin", "operator"},
		"/admanager.v1.AdManager/UpdateCampaignStatus": {"admin", "operator"},
		"/admanager.v1.AdManager/CreateAdGroup":        {"admin", "operator"},
		"/admanager.v1.AdManager/UpdateAdGroup":        {"admin", "operator"},
		"/admanager.v1.AdManager/UpdateAdGroupStatus":  {"admin", "operator"},
		"/admanager.v1.AdManager/CreateCreative":       {"admin", "operator"},
		"/admanager.v1.AdManager/SubmitAudit":          {"admin"},
		"/admanager.v1.AdManager/UpdateAdvertiser":     {"admin"},
		"/admanager.v1.AdManager/SubmitQualification":  {"admin", "operator"},
		"/admanager.v1.AdManager/AuditAdvertiser":      {"admin"},
		"/admanager.v1.AdManager/UploadProofMaterial":  {"admin", "operator"},
		"/admanager.v1.AdManager/Recharge":             {"admin"},
		"/admanager.v1.AdManager/CreateMedia":          {"admin"},
		"/admanager.v1.AdManager/UpdateMedia":          {"admin"},
		"/admanager.v1.AdManager/UpdateMediaStatus":    {"admin"},
		"/admanager.v1.AdManager/CreateAdPosition":     {"admin"},
		"/admanager.v1.AdManager/UpdateAdPosition":     {"admin"},
		"/admanager.v1.AdManager/ListUsers":            {"admin"},
		"/admanager.v1.AdManager/UpdateUserRole":       {"admin"},
		"/admanager.v1.AdManager/ListPendingAudits":    {"admin"},
		"/admanager.v1.AdManager/AuditCreative":        {"admin"},
		"/admanager.v1.AdManager/CreateAdvertiser":     {"admin"},
		"/admanager.v1.AdManager/DeleteAdvertiser":     {"admin"},
		"/admanager.v1.AdManager/CreateUser":           {"admin"},
	}
}

func (i *RBACInterceptor) UnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	allowedRoles, ok := i.rules[info.FullMethod]
	if !ok {
		return handler(ctx, req)
	}

	role, _ := ctx.Value("role").(string)
	if role == "" {
		role = "viewer"
	}

	for _, r := range allowedRoles {
		if r == role {
			return handler(ctx, req)
		}
	}

	return nil, status.Errorf(codes.PermissionDenied, "role %s not allowed for %s", role, info.FullMethod)
}
