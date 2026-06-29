package config

import (
	"testing"
)

func TestConfig_FilePath(t *testing.T) {
	c := &Config{DataDir: "/data"}
	got := c.FilePath("f_abc123")
	want := "/data/files/f_abc123"
	if got != want {
		t.Errorf("FilePath = %q, want %q", got, want)
	}
}

func TestConfig_FilesDir(t *testing.T) {
	c := &Config{DataDir: "/data"}
	got := c.FilesDir()
	want := "/data/files"
	if got != want {
		t.Errorf("FilesDir = %q, want %q", got, want)
	}
}

func TestConfig_DBPath(t *testing.T) {
	c := &Config{DataDir: "/data"}
	got := c.DBPath()
	want := "/data/metadata.db"
	if got != want {
		t.Errorf("DBPath = %q, want %q", got, want)
	}
}

func TestConfig_UploadURL(t *testing.T) {
	c := &Config{BaseURL: "http://localhost:8080"}
	got := c.UploadURL("f_123")
	want := "http://localhost:8080/upload/f_123"
	if got != want {
		t.Errorf("UploadURL = %q, want %q", got, want)
	}
}

func TestConfig_DownloadURL(t *testing.T) {
	c := &Config{BaseURL: "http://localhost:8080"}
	got := c.DownloadURL("abc12345")
	want := "http://localhost:8080/f/abc12345"
	if got != want {
		t.Errorf("DownloadURL = %q, want %q", got, want)
	}
}

func TestConfig_DownloadURL_ExternalURL(t *testing.T) {
	c := &Config{
		BaseURL:     "http://localhost:8080",
		ExternalURL: "http://aiderdesk.lan:3200",
	}
	got := c.DownloadURL("abc12345")
	want := "http://aiderdesk.lan:3200/f/abc12345"
	if got != want {
		t.Errorf("DownloadURL = %q, want %q", got, want)
	}
}

func TestConfig_UploadURL_ExternalURLDoesNotAffect(t *testing.T) {
	c := &Config{
		BaseURL:     "http://localhost:8080",
		ExternalURL: "http://aiderdesk.lan:3200",
	}
	got := c.UploadURL("f_123")
	want := "http://localhost:8080/upload/f_123"
	if got != want {
		t.Errorf("UploadURL = %q, want %q", got, want)
	}
}
