package authz

import (
	"testing"
)

func TestTenantID_WithSession(t *testing.T) {
	r := newTestRequest("user1", "tenant1", false)
	if tenantID(r) != "tenant1" {
		t.Errorf("expected tenant1, got %s", tenantID(r))
	}
}

func TestTenantID_NoSession(t *testing.T) {
	r := newTestRequestNoSession()
	if tenantID(r) != "" {
		t.Errorf("expected empty tenant, got %s", tenantID(r))
	}
}

func TestIsPlatformAdmin_True(t *testing.T) {
	r := newTestRequest("admin1", "tenant1", true)
	if !isPlatformAdmin(r) {
		t.Error("expected admin")
	}
}

func TestIsPlatformAdmin_False(t *testing.T) {
	r := newTestRequest("user1", "tenant1", false)
	if isPlatformAdmin(r) {
		t.Error("expected non-admin")
	}
}

func TestIsPlatformAdmin_NoSession(t *testing.T) {
	r := newTestRequestNoSession()
	if isPlatformAdmin(r) {
		t.Error("expected non-admin for no session")
	}
}
