package download

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/config"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore/model"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/httputil"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/osutil"
	"go.uber.org/zap"
)

type Handler struct {
	store filestore.FileStore
	cfg   *config.Config
	log   *zap.Logger
	fs    osutil.FileSystem
}

func NewHandler(store filestore.FileStore, cfg *config.Config, log *zap.Logger) *Handler {
	return &Handler{store: store, cfg: cfg, log: log, fs: osutil.RealFS{}}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	shareID := strings.TrimPrefix(r.URL.Path, model.RouteFile)
	if shareID == "" {
		http.Error(w, "missing share_id", http.StatusBadRequest)
		return
	}

	f, err := h.store.GetFileByShareID(shareID)
	if err != nil {
		httputil.ServerError(w, h.log, "internal error", "file lookup error", err)
		return
	}
	if f == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	switch f.Status {
	case model.StatusExpired:
		http.Error(w, "file expired", http.StatusGone)
		return
	case model.StatusDeleted:
		http.Error(w, "not found", http.StatusNotFound)
		return
	case model.StatusPending:
		http.Error(w, "file not yet uploaded", http.StatusNotFound)
		return
	case model.StatusReady:
	default:
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	switch f.Visibility {
	case model.VisPublic:
	case model.VisAgent:
		http.Error(w, model.ErrForbidden, http.StatusForbidden)
		return
	case model.VisToken:
		token := r.URL.Query().Get("token")
		if token == "" || token != f.DownloadToken {
			http.Error(w, model.ErrForbidden, http.StatusForbidden)
			return
		}
	default:
		http.Error(w, model.ErrForbidden, http.StatusForbidden)
		return
	}

	filePath := h.cfg.FilePath(f.FileID)
	info, err := h.fs.Stat(filePath)
	if err != nil {
		h.log.Error("file stat error", zap.String("file_id", f.FileID), zap.Error(err))
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", f.ContentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+f.Filename+"\"")
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))

	http.ServeFile(w, r, filePath)
}
