package filestore

import (
	"database/sql"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore/model"
	"golang.org/x/crypto/bcrypt"
	"go.uber.org/zap"
)

func tokenLookup(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	log := zap.NewNop()
	s, err := NewStore(dbPath, log)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestStore_RegisterAndGetAgent(t *testing.T) {
	s := newTestStore(t)

	err := s.RegisterAgent("agent_01", "hashedtoken", "lookup_agent_01", "sess_abc")
	if err != nil {
		t.Fatalf("RegisterAgent: %v", err)
	}

	a, err := s.GetAgentBySession("sess_abc")
	if err != nil {
		t.Fatalf("GetAgentBySession: %v", err)
	}
	if a == nil {
		t.Fatal("expected agent, got nil")
	}
	if a.AgentID != "agent_01" {
		t.Errorf("AgentID = %q, want %q", a.AgentID, "agent_01")
	}
	if a.SessionID != "sess_abc" {
		t.Errorf("SessionID = %q, want %q", a.SessionID, "sess_abc")
	}
}

func TestStore_GetAgentBySession_NotFound(t *testing.T) {
	s := newTestStore(t)
	a, err := s.GetAgentBySession("nonexistent")
	if err != nil {
		t.Fatalf("GetAgentBySession: %v", err)
	}
	if a != nil {
		t.Errorf("expected nil, got %+v", a)
	}
}

func TestStore_GetAgentByID(t *testing.T) {
	s := newTestStore(t)
	s.RegisterAgent("agent_02", "hash2", "lookup_agent_02", "sess_def")

	a, err := s.GetAgentByID("agent_02")
	if err != nil {
		t.Fatalf("GetAgentByID: %v", err)
	}
	if a == nil {
		t.Fatal("expected agent, got nil")
	}
	if a.AgentID != "agent_02" {
		t.Errorf("AgentID = %q, want %q", a.AgentID, "agent_02")
	}
}

func TestStore_VerifyAgentToken(t *testing.T) {
	s := newTestStore(t)
	token := "secret123"
	hash, _ := hashBcrypt(token)
	s.RegisterAgent("agent_03", hash, "lookup_agent_03", "sess_ghi")

	ok, err := s.VerifyAgentToken("agent_03", token)
	if err != nil {
		t.Fatalf("VerifyAgentToken: %v", err)
	}
	if !ok {
		t.Error("expected token to verify")
	}

	ok, err = s.VerifyAgentToken("agent_03", "wrong")
	if err != nil {
		t.Fatalf("VerifyAgentToken wrong: %v", err)
	}
	if ok {
		t.Error("expected token to fail verification")
	}

	ok, err = s.VerifyAgentToken("nonexistent", token)
	if err != nil {
		t.Fatalf("VerifyAgentToken nonexistent: %v", err)
	}
	if ok {
		t.Error("expected nonexistent agent to fail")
	}
}

func TestStore_VerifyAnyToken(t *testing.T) {
	s := newTestStore(t)
	token1 := "token_one"
	hash1, _ := hashBcrypt(token1)
	s.RegisterAgent("agent_a", hash1, "lookup_agent_a", "sess_1")

	token2 := "token_two"
	hash2, _ := hashBcrypt(token2)
	s.RegisterAgent("agent_b", hash2, "lookup_agent_b", "sess_2")

	a, err := s.VerifyAnyToken(token1)
	if err != nil {
		t.Fatalf("VerifyAnyToken: %v", err)
	}
	if a == nil || a.AgentID != "agent_a" {
		t.Errorf("expected agent_a, got %+v", a)
	}

	a, err = s.VerifyAnyToken(token2)
	if err != nil {
		t.Fatalf("VerifyAnyToken: %v", err)
	}
	if a == nil || a.AgentID != "agent_b" {
		t.Errorf("expected agent_b, got %+v", a)
	}

	a, err = s.VerifyAnyToken("invalid")
	if err != nil {
		t.Fatalf("VerifyAnyToken invalid: %v", err)
	}
	if a != nil {
		t.Errorf("expected nil for invalid token, got %+v", a)
	}
}

func TestStore_UpdateAgentSession(t *testing.T) {
	s := newTestStore(t)
	s.RegisterAgent("agent_04", "hash4", "lookup_agent_04", "sess_old")

	err := s.UpdateAgentSession("agent_04", "sess_new")
	if err != nil {
		t.Fatalf("UpdateAgentSession: %v", err)
	}

	a, _ := s.GetAgentBySession("sess_new")
	if a == nil || a.AgentID != "agent_04" {
		t.Errorf("expected agent_04 on new session, got %+v", a)
	}

	a, _ = s.GetAgentBySession("sess_old")
	if a != nil {
		t.Error("expected nil on old session")
	}
}

func TestStore_TouchAgent(t *testing.T) {
	s := newTestStore(t)
	s.RegisterAgent("agent_05", "hash5", "lookup_agent_05", "sess_5")

	before, _ := s.GetAgentByID("agent_05")
	time.Sleep(10 * time.Millisecond)
	s.TouchAgent("agent_05")
	after, _ := s.GetAgentByID("agent_05")

	if !after.LastSeen.After(before.LastSeen) {
		t.Error("expected LastSeen to be updated after TouchAgent")
	}
}

func TestStore_FileCRUD(t *testing.T) {
	s := newTestStore(t)
	s.RegisterAgent("agent_f1", "hashf1", "lookup_agent_f1", "sess_f1")

	expiresAt := time.Now().UTC().Add(72 * time.Hour)
	err := s.CreateFile("f_001", "share_abc", "agent_f1", "test.txt", "text/plain", model.VisPublic, 259200, sql.NullString{}, expiresAt)
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}

	f, err := s.GetFileByID("f_001")
	if err != nil {
		t.Fatalf("GetFileByID: %v", err)
	}
	if f == nil {
		t.Fatal("expected file, got nil")
	}
	if f.FileID != "f_001" {
		t.Errorf("FileID = %q, want %q", f.FileID, "f_001")
	}
	if f.Status != model.StatusPending {
		t.Errorf("Status = %q, want %q", f.Status, model.StatusPending)
	}
	if f.Filename != "test.txt" {
		t.Errorf("Filename = %q, want %q", f.Filename, "test.txt")
	}

	f2, err := s.GetFileByShareID("share_abc")
	if err != nil {
		t.Fatalf("GetFileByShareID: %v", err)
	}
	if f2 == nil || f2.FileID != "f_001" {
		t.Errorf("expected f_001 via share_id, got %+v", f2)
	}
}

func TestStore_CompleteUpload(t *testing.T) {
	s := newTestStore(t)
	s.RegisterAgent("agent_f2", "hashf2", "lookup_agent_f2", "sess_f2")
	expiresAt := time.Now().UTC().Add(72 * time.Hour)
	s.CreateFile("f_002", "share_def", "agent_f2", "photo.jpg", "image/jpeg", model.VisPublic, 259200, sql.NullString{}, expiresAt)

	err := s.CompleteUpload("f_002", "abc123sha", 4096)
	if err != nil {
		t.Fatalf("CompleteUpload: %v", err)
	}

	f, _ := s.GetFileByID("f_002")
	if f == nil {
		t.Fatal("expected file, got nil")
	}
	if f.Status != model.StatusReady {
		t.Errorf("Status = %q, want %q", f.Status, model.StatusReady)
	}
	if f.SHA256 != "abc123sha" {
		t.Errorf("SHA256 = %q, want %q", f.SHA256, "abc123sha")
	}
	if f.Size != 4096 {
		t.Errorf("Size = %d, want %d", f.Size, 4096)
	}
	if !f.UploadedAt.Valid {
		t.Error("expected UploadedAt to be valid")
	}
}

func TestStore_RevertToPending(t *testing.T) {
	s := newTestStore(t)
	s.RegisterAgent("agent_f3", "hashf3", "lookup_agent_f3", "sess_f3")
	expiresAt := time.Now().UTC().Add(72 * time.Hour)
	s.CreateFile("f_003", "share_ghi", "agent_f3", "doc.pdf", "application/pdf", model.VisPublic, 259200, sql.NullString{}, expiresAt)
	s.CompleteUpload("f_003", "sha_revert", 1024)

	err := s.RevertToPending("f_003")
	if err != nil {
		t.Fatalf("RevertToPending: %v", err)
	}

	f, _ := s.GetFileByID("f_003")
	if f == nil {
		t.Fatal("expected file, got nil")
	}
	if f.Status != model.StatusPending {
		t.Errorf("Status = %q, want %q", f.Status, model.StatusPending)
	}
	if f.SHA256 != "" {
		t.Errorf("SHA256 = %q, want empty", f.SHA256)
	}
	if f.Size != 0 {
		t.Errorf("Size = %d, want 0", f.Size)
	}
}

func TestStore_DeleteFile(t *testing.T) {
	s := newTestStore(t)
	s.RegisterAgent("agent_f4", "hashf4", "lookup_agent_f4", "sess_f4")
	expiresAt := time.Now().UTC().Add(72 * time.Hour)
	s.CreateFile("f_004", "share_jkl", "agent_f4", "data.csv", "text/csv", model.VisPublic, 259200, sql.NullString{}, expiresAt)
	s.CompleteUpload("f_004", "sha_del", 512)

	deleted, err := s.DeleteFile("f_004", "agent_f4")
	if err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	if !deleted {
		t.Error("expected deleted=true")
	}

	f, _ := s.GetFileByID("f_004")
	if f == nil {
		t.Fatal("expected file (soft delete), got nil")
	}
	if f.Status != model.StatusDeleted {
		t.Errorf("Status = %q, want %q", f.Status, model.StatusDeleted)
	}
}

func TestStore_DeleteFile_WrongAgent(t *testing.T) {
	s := newTestStore(t)
	s.RegisterAgent("agent_f5", "hashf5", "lookup_agent_f5", "sess_f5")
	s.RegisterAgent("agent_f6", "hashf6", "lookup_agent_f6", "sess_f6")
	expiresAt := time.Now().UTC().Add(72 * time.Hour)
	s.CreateFile("f_005", "share_mno", "agent_f5", "file.txt", "text/plain", model.VisPublic, 259200, sql.NullString{}, expiresAt)
	s.CompleteUpload("f_005", "sha_wa", 100)

	deleted, err := s.DeleteFile("f_005", "agent_f6")
	if err != nil {
		t.Fatalf("DeleteFile wrong agent: %v", err)
	}
	if deleted {
		t.Error("expected deleted=false for wrong agent")
	}
}

func TestStore_DeleteFile_AlreadyDeleted(t *testing.T) {
	s := newTestStore(t)
	s.RegisterAgent("agent_f7", "hashf7", "lookup_agent_f7", "sess_f7")
	expiresAt := time.Now().UTC().Add(72 * time.Hour)
	s.CreateFile("f_006", "share_pqr", "agent_f7", "file2.txt", "text/plain", model.VisPublic, 259200, sql.NullString{}, expiresAt)
	s.CompleteUpload("f_006", "sha_ad", 200)
	s.DeleteFile("f_006", "agent_f7")

	deleted, err := s.DeleteFile("f_006", "agent_f7")
	if err != nil {
		t.Fatalf("DeleteFile already deleted: %v", err)
	}
	if deleted {
		t.Error("expected deleted=false for already-deleted file")
	}
}

func TestStore_ExpireFiles(t *testing.T) {
	s := newTestStore(t)
	s.RegisterAgent("agent_exp", "hashexp", "lookup_agent_exp", "sess_exp")

	pastExpiry := time.Now().UTC().Add(-1 * time.Hour)
	s.CreateFile("f_exp1", "share_exp1", "agent_exp", "old.txt", "text/plain", model.VisPublic, 259200, sql.NullString{}, pastExpiry)
	s.CompleteUpload("f_exp1", "sha_exp1", 50)

	futureExpiry := time.Now().UTC().Add(72 * time.Hour)
	s.CreateFile("f_exp2", "share_exp2", "agent_exp", "new.txt", "text/plain", model.VisPublic, 259200, sql.NullString{}, futureExpiry)
	s.CompleteUpload("f_exp2", "sha_exp2", 50)

	n, err := s.ExpireFiles()
	if err != nil {
		t.Fatalf("ExpireFiles: %v", err)
	}
	if n != 1 {
		t.Errorf("expired %d files, want 1", n)
	}

	f, _ := s.GetFileByID("f_exp1")
	if f == nil || f.Status != model.StatusExpired {
		t.Errorf("expected f_exp1 to be expired, got status=%q", f.Status)
	}

	f, _ = s.GetFileByID("f_exp2")
	if f == nil || f.Status != model.StatusReady {
		t.Errorf("expected f_exp2 to remain ready, got status=%q", f.Status)
	}
}

func TestStore_GetExpiredFileIDs(t *testing.T) {
	s := newTestStore(t)
	s.RegisterAgent("agent_eid", "hasheid", "lookup_agent_eid", "sess_eid")

	pastExpiry := time.Now().UTC().Add(-1 * time.Hour)
	s.CreateFile("f_eid1", "share_eid1", "agent_eid", "expired.txt", "text/plain", model.VisPublic, 259200, sql.NullString{}, pastExpiry)
	s.CompleteUpload("f_eid1", "sha_eid1", 10)
	s.ExpireFiles()

	ids, err := s.GetExpiredFileIDs()
	if err != nil {
		t.Fatalf("GetExpiredFileIDs: %v", err)
	}
	if len(ids) != 1 || ids[0] != "f_eid1" {
		t.Errorf("expected [f_eid1], got %v", ids)
	}
}

func TestStore_ListFiles(t *testing.T) {
	s := newTestStore(t)
	s.RegisterAgent("agent_lf", "hashlf", "lookup_agent_lf", "sess_lf")
	expiresAt := time.Now().UTC().Add(72 * time.Hour)

	s.CreateFile("f_lf1", "share_lf1", "agent_lf", "a.txt", "text/plain", model.VisPublic, 259200, sql.NullString{}, expiresAt)
	s.CompleteUpload("f_lf1", "sha_lf1", 100)
	s.CreateFile("f_lf2", "share_lf2", "agent_lf", "b.txt", "text/plain", model.VisPublic, 259200, sql.NullString{}, expiresAt)
	s.CompleteUpload("f_lf2", "sha_lf2", 200)

	files, total, err := s.ListFiles("agent_lf", model.StatusReady, 10, 0)
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(files) != 2 {
		t.Errorf("got %d files, want 2", len(files))
	}
}

func TestStore_ListFiles_Pagination(t *testing.T) {
	s := newTestStore(t)
	s.RegisterAgent("agent_pag", "hashpag", "lookup_agent_pag", "sess_pag")
	expiresAt := time.Now().UTC().Add(72 * time.Hour)

	for i := 0; i < 5; i++ {
		fid := fmt.Sprintf("f_pag%d", i)
		sid := fmt.Sprintf("share_pag%d", i)
		fn := fmt.Sprintf("file%d.txt", i)
		if err := s.CreateFile(fid, sid, "agent_pag", fn, "text/plain", model.VisPublic, 259200, sql.NullString{}, expiresAt); err != nil {
			t.Fatalf("CreateFile %s: %v", fid, err)
		}
		if err := s.CompleteUpload(fid, "sha_"+fid, int64(i*100)); err != nil {
			t.Fatalf("CompleteUpload %s: %v", fid, err)
		}
	}

	files, total, err := s.ListFiles("agent_pag", model.StatusReady, 2, 0)
	if err != nil {
		t.Fatalf("ListFiles page1: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(files) != 2 {
		t.Errorf("page 1: got %d files, want 2", len(files))
	}

	files2, total2, err := s.ListFiles("agent_pag", model.StatusReady, 2, 2)
	if err != nil {
		t.Fatalf("ListFiles page2: %v", err)
	}
	if total2 != 5 {
		t.Errorf("total2 = %d, want 5", total2)
	}
	if len(files2) != 2 {
		t.Errorf("page 2: got %d files, want 2", len(files2))
	}
}

func TestStore_SearchFiles(t *testing.T) {
	s := newTestStore(t)
	s.RegisterAgent("agent_search", "hashsearch", "lookup_agent_search", "sess_search")
	expiresAt := time.Now().UTC().Add(72 * time.Hour)

	s.CreateFile("f_s1", "share_s1", "agent_search", "quarterly_report.pdf", "application/pdf", model.VisPublic, 259200, sql.NullString{}, expiresAt)
	s.CompleteUpload("f_s1", "sha_s1", 1000)

	s.CreateFile("f_s2", "share_s2", "agent_search", "annual_report.docx", "application/docx", model.VisPublic, 259200, sql.NullString{}, expiresAt)
	s.CompleteUpload("f_s2", "sha_s2", 2000)

	s.CreateFile("f_s3", "share_s3", "agent_search", "photo.jpg", "image/jpeg", model.VisPublic, 259200, sql.NullString{}, expiresAt)
	s.CompleteUpload("f_s3", "sha_s3", 3000)

	files, err := s.SearchFiles("report", "agent_search", 10)
	if err != nil {
		t.Fatalf("SearchFiles: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("got %d results, want 2", len(files))
	}

	files, err = s.SearchFiles("photo", "agent_search", 10)
	if err != nil {
		t.Fatalf("SearchFiles photo: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("got %d results, want 1", len(files))
	}
}

func TestStore_SearchFiles_PendingNotReturned(t *testing.T) {
	s := newTestStore(t)
	s.RegisterAgent("agent_sp", "hashsp", "lookup_agent_sp", "sess_sp")
	expiresAt := time.Now().UTC().Add(72 * time.Hour)

	s.CreateFile("f_sp1", "share_sp1", "agent_sp", "pending_report.pdf", "application/pdf", model.VisPublic, 259200, sql.NullString{}, expiresAt)

	files, err := s.SearchFiles("report", "agent_sp", 10)
	if err != nil {
		t.Fatalf("SearchFiles: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 results for pending file, got %d", len(files))
	}
}

func TestStore_DownloadTokenVisibility(t *testing.T) {
	s := newTestStore(t)
	s.RegisterAgent("agent_vis", "hashvis", "lookup_agent_vis", "sess_vis")
	expiresAt := time.Now().UTC().Add(72 * time.Hour)

	dlToken := sql.NullString{String: "secret_token_123", Valid: true}
	s.CreateFile("f_vis", "share_vis", "agent_vis", "private.txt", "text/plain", model.VisToken, 259200, dlToken, expiresAt)
	s.CompleteUpload("f_vis", "sha_vis", 50)

	f, err := s.GetFileByDownloadToken("secret_token_123")
	if err != nil {
		t.Fatalf("GetFileByDownloadToken: %v", err)
	}
	if f == nil || f.FileID != "f_vis" {
		t.Errorf("expected f_vis, got %+v", f)
	}

	f, err = s.GetFileByDownloadToken("wrong_token")
	if err != nil {
		t.Fatalf("GetFileByDownloadToken wrong: %v", err)
	}
	if f != nil {
		t.Errorf("expected nil for wrong token, got %+v", f)
	}
}

func TestStore_ListAgents(t *testing.T) {
	s := newTestStore(t)
	s.RegisterAgent("agent_la1", "hashla1", "lookup_agent_la1", "sess_la1")
	s.RegisterAgent("agent_la2", "hashla2", "lookup_agent_la2", "sess_la2")
	expiresAt := time.Now().UTC().Add(72 * time.Hour)
	s.CreateFile("f_la1", "share_la1", "agent_la1", "file.txt", "text/plain", model.VisPublic, 259200, sql.NullString{}, expiresAt)
	s.CompleteUpload("f_la1", "sha_la1", 500)

	agents, err := s.ListAgents()
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}
	if len(agents) != 2 {
		t.Errorf("got %d agents, want 2", len(agents))
	}

	var found *AgentInfo
	for i := range agents {
		if agents[i].AgentID == "agent_la1" {
			found = &agents[i]
			break
		}
	}
	if found == nil {
		t.Fatal("agent_la1 not found")
	}
	if found.FileCount != 1 {
		t.Errorf("FileCount = %d, want 1", found.FileCount)
	}
	if found.TotalSize != 500 {
		t.Errorf("TotalSize = %d, want 500", found.TotalSize)
	}
}

func TestStore_RegisterAgent_DuplicateAgentID(t *testing.T) {
	s := newTestStore(t)
	s.RegisterAgent("agent_up", "hash_up1", "lookup_up1", "sess_up1")

	// Second insert with same agent_id should fail (UNIQUE constraint),
	// preserving the original agent and its token lookup.
	err := s.RegisterAgent("agent_up", "hash_up2", "lookup_up2", "sess_up2")
	if err == nil {
		t.Error("expected UNIQUE constraint error on duplicate agent_id")
	}

	a1, _ := s.GetAgentBySession("sess_up1")
	if a1 == nil || a1.AgentID != "agent_up" {
		t.Errorf("original agent should still exist, got %+v", a1)
	}
}

func TestStore_GetFileByID_NotFound(t *testing.T) {
	s := newTestStore(t)
	f, err := s.GetFileByID("nonexistent")
	if err != nil {
		t.Fatalf("GetFileByID: %v", err)
	}
	if f != nil {
		t.Errorf("expected nil, got %+v", f)
	}
}

func hashBcrypt(token string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.MinCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
