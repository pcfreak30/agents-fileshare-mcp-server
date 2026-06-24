package download

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore/model"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/testutil"
)

func TestHandler_ServeHTTP_MethodNotAllowed(t *testing.T) {
	h := NewHandler(testutil.NewTestStore(), testutil.NewTestConfig(), testutil.NewTestLogger())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/f/share123", nil)
	h.ServeHTTP(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandler_ServeHTTP_MissingShareID(t *testing.T) {
	h := NewHandler(testutil.NewTestStore(), testutil.NewTestConfig(), testutil.NewTestLogger())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/f/", nil)
	h.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_ServeHTTP_FileNotFound(t *testing.T) {
	store := testutil.NewTestStore()
	store.GetFileByShareIDFn = func(shareID string) (*filestore.FileMetadata, error) {
		return nil, nil
	}

	h := NewHandler(store, testutil.NewTestConfig(), testutil.NewTestLogger())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/f/share404", nil)
	h.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandler_ServeHTTP_ExpiredFile(t *testing.T) {
	f := testutil.ReadyFile("f_1", "share_exp", "agent_1", "test.txt", "text/plain", model.VisPublic)
	f.Status = model.StatusExpired

	store := testutil.NewTestStore()
	store.GetFileByShareIDFn = func(shareID string) (*filestore.FileMetadata, error) {
		return f, nil
	}

	h := NewHandler(store, testutil.NewTestConfig(), testutil.NewTestLogger())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/f/share_exp", nil)
	h.ServeHTTP(w, r)

	if w.Code != http.StatusGone {
		t.Errorf("status = %d, want %d", w.Code, http.StatusGone)
	}
}

func TestHandler_ServeHTTP_DeletedFile(t *testing.T) {
	f := testutil.ReadyFile("f_1", "share_del", "agent_1", "test.txt", "text/plain", model.VisPublic)
	f.Status = model.StatusDeleted

	store := testutil.NewTestStore()
	store.GetFileByShareIDFn = func(shareID string) (*filestore.FileMetadata, error) {
		return f, nil
	}

	h := NewHandler(store, testutil.NewTestConfig(), testutil.NewTestLogger())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/f/share_del", nil)
	h.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandler_ServeHTTP_PendingFile(t *testing.T) {
	f := testutil.PendingFile("f_1", "share_pen", "agent_1", "test.txt", "text/plain", model.VisPublic)

	store := testutil.NewTestStore()
	store.GetFileByShareIDFn = func(shareID string) (*filestore.FileMetadata, error) {
		return f, nil
	}

	h := NewHandler(store, testutil.NewTestConfig(), testutil.NewTestLogger())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/f/share_pen", nil)
	h.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandler_ServeHTTP_AgentVisibility_Forbidden(t *testing.T) {
	f := testutil.ReadyFile("f_1", "share_ag", "agent_1", "test.txt", "text/plain", model.VisAgent)

	store := testutil.NewTestStore()
	store.GetFileByShareIDFn = func(shareID string) (*filestore.FileMetadata, error) {
		return f, nil
	}

	h := NewHandler(store, testutil.NewTestConfig(), testutil.NewTestLogger())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/f/share_ag", nil)
	h.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandler_ServeHTTP_TokenVisibility_MissingToken(t *testing.T) {
	f := testutil.ReadyFile("f_1", "share_tok", "agent_1", "test.txt", "text/plain", model.VisToken)
	f.DownloadToken = "secret123"

	store := testutil.NewTestStore()
	store.GetFileByShareIDFn = func(shareID string) (*filestore.FileMetadata, error) {
		return f, nil
	}

	h := NewHandler(store, testutil.NewTestConfig(), testutil.NewTestLogger())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/f/share_tok", nil)
	h.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandler_ServeHTTP_TokenVisibility_WrongToken(t *testing.T) {
	f := testutil.ReadyFile("f_1", "share_tok2", "agent_1", "test.txt", "text/plain", model.VisToken)
	f.DownloadToken = "secret123"

	store := testutil.NewTestStore()
	store.GetFileByShareIDFn = func(shareID string) (*filestore.FileMetadata, error) {
		return f, nil
	}

	h := NewHandler(store, testutil.NewTestConfig(), testutil.NewTestLogger())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/f/share_tok2?token=wrong", nil)
	h.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandler_ServeHTTP_PublicFile_DiskStatError(t *testing.T) {
	f := testutil.ReadyFile("f_1", "share_stat", "agent_1", "test.txt", "text/plain", model.VisPublic)

	store := testutil.NewTestStore()
	store.GetFileByShareIDFn = func(shareID string) (*filestore.FileMetadata, error) {
		return f, nil
	}

	memFS := testutil.NewMemFS()

	h := &Handler{
		store: store,
		cfg:   testutil.NewTestConfig(),
		log:   testutil.NewTestLogger(),
		fs:    memFS,
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/f/share_stat", nil)
	h.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d (file not on disk)", w.Code, http.StatusNotFound)
	}
}

func TestHandler_ServeHTTP_StoreLookupError(t *testing.T) {
	store := testutil.NewTestStore()
	store.GetFileByShareIDFn = func(shareID string) (*filestore.FileMetadata, error) {
		return nil, errors.New("db error")
	}

	h := NewHandler(store, testutil.NewTestConfig(), testutil.NewTestLogger())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/f/share_err", nil)
	h.ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
