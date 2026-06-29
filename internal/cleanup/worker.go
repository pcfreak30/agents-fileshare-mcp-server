package cleanup

import (
	"os"
	"time"

	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/config"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/osutil"
	"go.uber.org/zap"
)

type Worker struct {
	store filestore.FileStore
	cfg   *config.Config
	log   *zap.Logger
	fs    osutil.FileSystem
	done  chan struct{}
}

func NewWorker(store filestore.FileStore, cfg *config.Config, log *zap.Logger) *Worker {
	return &Worker{store: store, cfg: cfg, log: log, fs: osutil.RealFS{}, done: make(chan struct{})}
}

func (w *Worker) Start() {
	go w.run()
}

func (w *Worker) Stop() {
	close(w.done)
}

func (w *Worker) run() {
	ticker := time.NewTicker(w.cfg.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.done:
			return
		case <-ticker.C:
			w.sweep()
		}
	}
}

func (w *Worker) sweep() {
	n, err := w.store.ExpireFiles()
	if err != nil {
		w.log.Error("expire files error", zap.Error(err))
	} else if n > 0 {
		w.log.Info("expired files", zap.Int("count", n))
	}

	ghosts, err := w.store.ExpirePendingFiles(w.cfg.GhostFileTTL)
	if err != nil {
		w.log.Error("expire pending ghost files", zap.Error(err))
	} else if ghosts > 0 {
		w.log.Info("expired stale pending files", zap.Int("count", ghosts))
	}

	purged, err := w.store.PurgeStaleAgents(w.cfg.AgentTTL)
	if err != nil {
		w.log.Error("purge stale agents", zap.Error(err))
	} else if purged > 0 {
		w.log.Info("purged stale agents", zap.Int("count", purged))
	}

	ids, err := w.store.GetExpiredFileIDs()
	if err != nil {
		w.log.Error("query expired files", zap.Error(err))
		return
	}

	for _, fileID := range ids {
		path := w.cfg.FilePath(fileID)
		if err := w.fs.Remove(path); err != nil && !os.IsNotExist(err) {
			w.log.Error("remove expired file", zap.String("file_id", fileID), zap.Error(err))
		}
	}
}
