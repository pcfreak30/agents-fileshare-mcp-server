package config

import (
	"fmt"
	"path/filepath"
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
	UploadRateLimit   int           `koanf:"UPLOAD_RATE_LIMIT"`
	VerifyContentType bool          `koanf:"VERIFY_CONTENT_TYPE"`
}

func Load() *Config {
	k := koanfv2.NewWithConf(koanfv2.Conf{
		Delim:       ".",
		StrictMerge: true,
	})

	defaults := map[string]any{
		"LISTEN_ADDR":        ":8080",
		"DATA_DIR":           "/data",
		"MAX_FILE_SIZE":      int64(536870912),
		"DEFAULT_TTL":        "72h",
		"MAX_TTL":           "168h",
		"CLEANUP_INTERVAL":   "1m",
		"SHARE_ID_LENGTH":   8,
		"BASE_URL":          "http://localhost:8080",
		"UPLOAD_RATE_LIMIT": 10,
		"VERIFY_CONTENT_TYPE": false,
	}

	if err := k.Load(env.Provider(".", env.Opt{}), nil); err != nil {
		panic(fmt.Sprintf("load env config: %v", err))
	}

	for key, val := range defaults {
		if k.Get(key) == nil {
			k.Set(key, val)
		}
	}

	c := &Config{
		ListenAddr:        k.String("LISTEN_ADDR"),
		DataDir:           k.String("DATA_DIR"),
		MaxFileSize:       k.Int64("MAX_FILE_SIZE"),
		DefaultTTL:        k.Duration("DEFAULT_TTL"),
		MaxTTL:            k.Duration("MAX_TTL"),
		CleanupInterval:   k.Duration("CLEANUP_INTERVAL"),
		ShareIDLength:      k.Int("SHARE_ID_LENGTH"),
		BaseURL:           k.String("BASE_URL"),
		UploadRateLimit:    k.Int("UPLOAD_RATE_LIMIT"),
		VerifyContentType: k.Bool("VERIFY_CONTENT_TYPE"),
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
	return c.BaseURL + model.RouteFile + shareID
}
