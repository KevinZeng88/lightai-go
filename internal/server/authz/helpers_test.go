package authz

import (
	"net/http"
	"net/http/httptest"

	"lightai-go/internal/server/auth"
)

func newTestRequest(userID, tenantID string, isAdmin bool) *http.Request {
	r := httptest.NewRequest("GET", "/", nil)
	ctx := auth.NewContextWithSessionInfo(r.Context(), &auth.SessionInfo{
		UserID:          userID,
		TenantID:        tenantID,
		IsPlatformAdmin: isAdmin,
	})
	return r.WithContext(ctx)
}

func newTestRequestNoSession() *http.Request {
	return httptest.NewRequest("GET", "/", nil)
}
