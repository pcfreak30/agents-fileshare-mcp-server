package config

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/knadh/koanf/providers/env/v2"
	koanfv2 "github.com/knadh/koanf/v2"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore/model"
)

type Config struct {
	ListenAddr        string        `koanf:"LISTEN_ADDR"`
	DataDir           string        `koanf:"DATA_DIR"`
	MaxFileSize       int64         `koanf:"MAX_FILE_SIZE"`
	DefaultTTL        time.Duration `koanf:"DEFAULT_TTL"`
	MaxTTL            time.Duration `koanf:"MAX_TTL"`
	CleanupInterval   time.Duration `koanf:"CLEANUP_INTERVAL"`
	ShareIDLength     int           `koanf:"SHARE_ID_LENGTH"`
	BaseURL           string        `koanf:"BASE_URL"`
	ExternalURL       string        `koanf:"EXTERNAL_URL"`
	UploadRateLimit   int           `koanf:"UPLOAD_RATE_LIMIT"`
	VerifyContentType bool          `koanf:"VERIFY_CONTENT_TYPE"`
}

func Load() *Config {
	k := koanfv2.NewWithConf(koanfv2.Conf{
		Delim:       ".",
		StrictMerge: true,
	})

	defaults := map[string]any{
		"listen_addr":         ":8080",
		"data_dir":            "/data",
		"max_file_size":       int64(536870912),
		"default_ttl":         "72h",
		"max_ttl":            "168h",
		"cleanup_interval":   "1m",
		"share_id_length":    8,
		"base_url":           "http://localhost:8080",
		"external_url":       "",
		"upload_rate_limit":  10,
		"verify_content_type": false,
	}

	if err := k.Load(env.Provider(".", env.Opt{
		TransformFunc: func(k, v string) (string, any) { return strings.ToLower(k), v },
	}), nil); err != nil {
		panic(fmt.Sprintf("load env config: %v", err))
	}

	for key, val := range defaults {
		if k.Get(key) == nil {
			k.Set(key, val)
		}
	}

	c := &Config{
		ListenAddr:        k.String("listen_addr"),
		DataDir:           k.String("data_dir"),
		MaxFileSize:       k.Int64("max_file_size"),
		DefaultTTL:        k.Duration("default_ttl"),
		MaxTTL:            k.Duration("max_ttl"),
		CleanupInterval:   k.Duration("cleanup_interval"),
		ShareIDLength:     k.Int("share_id_length"),
		BaseURL:           k.String("base_url"),
		ExternalURL:       k.String("external_url"),
		UploadRateLimit:   k.Int("upload_rate_limit"),
		VerifyContentType: k.Bool("verify_content_type"),
	}

	return c
}

func (c *Config) FilePath(id string) string {
	return filepath.Join(c.DataDir, "files", id)
}

func (c *Config) FilesDir() string {
	return filepath.Join(c.DataDir, "files")
}

func (c *Config) DBPath() string {
	return filepath.Join(c.DataDir, "metadata.db")
}

func (c *Config) UploadURL(fileID string) string {
	return c.BaseURL + model.RouteUpload + fileID
}

func (c *Config) DownloadURL(shareID string) string {
	base := c.ExternalURL
	if base == "" {
		base = c.BaseURL
	}
	return base + model.RouteFile + shareID
}
