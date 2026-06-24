package filestore

import (
	"testing"
	"time"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal.txt", "normal.txt"},
		{"path/to/file.txt", "path_to_file.txt"},
		{"path\\to\\file.txt", "path_to_file.txt"},
		{"../secret", "__secret"},
		{"a/../b", "a___b"},
		{"", ""},
	}

	for _, tt := range tests {
		got := SanitizeFilename(tt.input)
		if got != tt.want {
			t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSanitizeFilename_MaxLength(t *testing.T) {
	long := string(make([]byte, 300))
	for i := range long {
		long = long[:i] + "a" + long[i+1:]
	}
	got := SanitizeFilename(long)
	if len(got) != 255 {
		t.Errorf("SanitizeFilename length = %d, want 255", len(got))
	}
}

func TestParseTTL(t *testing.T) {
	tests := []struct {
		input      string
		defaultTTL time.Duration
		maxTTL     time.Duration
		want       time.Duration
		wantErr    bool
	}{
		{"", 72 * time.Hour, 168 * time.Hour, 72 * time.Hour, false},
		{"24h", 72 * time.Hour, 168 * time.Hour, 24 * time.Hour, false},
		{"30m", 72 * time.Hour, 168 * time.Hour, 30 * time.Minute, false},
		{"200h", 72 * time.Hour, 168 * time.Hour, 0, true},
		{"-1h", 72 * time.Hour, 168 * time.Hour, 0, true},
		{"notaduration", 72 * time.Hour, 168 * time.Hour, 0, true},
		{"0s", 72 * time.Hour, 168 * time.Hour, 0, true},
	}

	for _, tt := range tests {
		got, err := ParseTTL(tt.input, tt.defaultTTL, tt.maxTTL)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseTTL(%q) expected error, got nil", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseTTL(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseTTL(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseVisibility(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"", "public", false},
		{"public", "public", false},
		{"agent", "agent", false},
		{"token", "token", false},
		{"private", "", true},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		got, err := ParseVisibility(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseVisibility(%q) expected error, got nil", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseVisibility(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if string(got) != tt.want {
			t.Errorf("ParseVisibility(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSanitizeFTSQuery(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"hello", "hello*"},
		{"hello world", "hello* world*"},
		{"hello  world", "hello* world*"},
		{`"quoted"`, "quoted*"},
		{"*wild", "wild*"},
		{"+plus", "plus*"},
		{"-minus", "minus*"},
		{"^caret", "caret*"},
	}

	for _, tt := range tests {
		got := sanitizeFTSQuery(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeFTSQuery(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestPrefixedColumns(t *testing.T) {
	got := prefixedColumns("f")
	want := "f.file_id, f.share_id, f.agent_id, f.filename, f.content_type, f.size, f.sha256, f.visibility, f.download_token, f.status, f.ttl_seconds, f.uploaded_at, f.expires_at, f.created_at"
	if got != want {
		t.Errorf("prefixedColumns(\"f\") = %q, want %q", got, want)
	}

	got2 := prefixedColumns("files")
	want2 := "files.file_id, files.share_id, files.agent_id, files.filename, files.content_type, files.size, files.sha256, files.visibility, files.download_token, files.status, files.ttl_seconds, files.uploaded_at, files.expires_at, files.created_at"
	if got2 != want2 {
		t.Errorf("prefixedColumns(\"files\") = %q, want %q", got2, want2)
	}
}
