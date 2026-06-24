package model

import "testing"

func TestFileStatus_String(t *testing.T) {
	tests := []struct {
		s    FileStatus
		want string
	}{
		{StatusPending, "pending"},
		{StatusReady, "ready"},
		{StatusExpired, "expired"},
		{StatusDeleted, "deleted"},
	}
	for _, tt := range tests {
		if tt.s.String() != tt.want {
			t.Errorf("FileStatus(%q).String() = %q, want %q", tt.s, tt.s.String(), tt.want)
		}
	}
}

func TestVisibility_String(t *testing.T) {
	tests := []struct {
		v    Visibility
		want string
	}{
		{VisPublic, "public"},
		{VisAgent, "agent"},
		{VisToken, "token"},
	}
	for _, tt := range tests {
		if tt.v.String() != tt.want {
			t.Errorf("Visibility(%q).String() = %q, want %q", tt.v, tt.v.String(), tt.want)
		}
	}
}

func TestRouteConstants(t *testing.T) {
	if RouteUpload != "/upload/" {
		t.Errorf("RouteUpload = %q, want /upload/", RouteUpload)
	}
	if RouteFile != "/f/" {
		t.Errorf("RouteFile = %q, want /f/", RouteFile)
	}
	if RouteMCP != "/mcp" {
		t.Errorf("RouteMCP = %q, want /mcp", RouteMCP)
	}
	if RouteHealth != "/health" {
		t.Errorf("RouteHealth = %q, want /health", RouteHealth)
	}
}

func TestErrorConstants(t *testing.T) {
	if ErrNotAuthenticated != "not authenticated" {
		t.Errorf("ErrNotAuthenticated = %q, want %q", ErrNotAuthenticated, "not authenticated")
	}
	if ErrForbidden != "forbidden" {
		t.Errorf("ErrForbidden = %q, want %q", ErrForbidden, "forbidden")
	}
	if ErrFileNotFound != "file not found" {
		t.Errorf("ErrFileNotFound = %q, want %q", ErrFileNotFound, "file not found")
	}
}
