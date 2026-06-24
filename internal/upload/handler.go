package upload

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/agent"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/config"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore/model"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/httputil"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/osutil"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type Handler struct {
	store  filestore.FileStore
	agents *agent.Manager
	cfg    *config.Config
	log    *zap.Logger
	fs     osutil.FileSystem
	limits map[string]*rate.Limiter
	mu     sync.Mutex
}

func NewHandler(store filestore.FileStore, agents *agent.Manager, cfg *config.Config, log *zap.Logger) *Handler {
	return &Handler{
		store:  store,
		agents: agents,
		cfg:    cfg,
		log:    log,
		fs:     osutil.RealFS{},
		limits: make(map[string]*rate.Limiter),
	}
}

func (h *Handler) getLimiter(agentID string) *rate.Limiter {
	h.mu.Lock()
	defer h.mu.Unlock()
	if l, ok := h.limits[agentID]; ok {
		return l
	}
	l := rate.NewLimiter(rate.Limit(h.cfg.UploadRateLimit)/60, h.cfg.UploadRateLimit)
	h.limits[agentID] = l
	return l
}

func (h *Handler) cleanupTmp(dst *os.File, tmpPath string) {
	dst.Close()
	h.fs.Remove(tmpPath)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")

	ag, err := h.agents.VerifyToken(token)
	if err != nil {
		httputil.ServerError(w, h.log, "internal error", "token verification error", err)
		return
	}
	if ag == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	limiter := h.getLimiter(ag.AgentID)
	if !limiter.Allow() {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	fileID := strings.TrimPrefix(r.URL.Path, model.RouteUpload)
	if fileID == "" {
		http.Error(w, "missing file_id", http.StatusBadRequest)
		return
	}

	f, err := h.store.GetFileByID(fileID)
	if err != nil {
		httputil.ServerError(w, h.log, "internal error", "file lookup error", err)
		return
	}
	if f == nil {
		http.Error(w, model.ErrFileNotFound, http.StatusNotFound)
		return
	}
	if f.Status != model.StatusPending {
		http.Error(w, "file already uploaded", http.StatusConflict)
		return
	}
	if f.AgentID != ag.AgentID {
		http.Error(w, model.ErrForbidden, http.StatusForbidden)
		return
	}

	filePath := h.cfg.FilePath(fileID)
	if err := h.fs.MkdirAll(h.cfg.FilesDir(), 0755); err != nil {
		httputil.ServerError(w, h.log, "internal error", "create files dir", err)
		return
	}

	tmpPath := filePath + ".tmp"
	dst, err := h.fs.Create(tmpPath)
	if err != nil {
		httputil.ServerError(w, h.log, "internal error", "create temp file", err)
		return
	}

	hasher := sha256.New()
	maxReader := io.LimitReader(r.Body, h.cfg.MaxFileSize+1)
	mw := io.MultiWriter(dst, hasher)

	written, err := io.Copy(mw, maxReader)
	if err != nil {
		h.cleanupTmp(dst, tmpPath)
		httputil.ServerError(w, h.log, "upload failed", "write file", err)
		return
	}

	if written > h.cfg.MaxFileSize {
		h.cleanupTmp(dst, tmpPath)
		http.Error(w, fmt.Sprintf("file exceeds maximum size of %d bytes", h.cfg.MaxFileSize), http.StatusRequestEntityTooLarge)
		return
	}

	if err := dst.Close(); err != nil {
		h.fs.Remove(tmpPath)
		httputil.ServerError(w, h.log, "upload failed", "close temp file", err)
		return
	}

	if err := h.fs.Rename(tmpPath, filePath); err != nil {
		h.fs.Remove(tmpPath)
		httputil.ServerError(w, h.log, "upload failed", "rename temp file", err)
		return
	}

	sha := hex.EncodeToString(hasher.Sum(nil))
	if err := h.store.CompleteUpload(fileID, sha, written); err != nil {
		h.fs.Remove(filePath)
		h.store.RevertToPending(fileID)
		httputil.ServerError(w, h.log, "upload failed", "complete upload metadata", err)
		return
	}

	h.log.Info("file uploaded",
		zap.String("file_id", fileID),
		zap.String("agent", ag.AgentID),
		zap.Int64("size", written),
		zap.String("sha256", sha),
	)
	w.WriteHeader(http.StatusNoContent)
}
