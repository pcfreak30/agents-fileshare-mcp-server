package cleanup

import (
	"errors"
	"os"
	"testing"

	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/testutil"
)

func TestWorker_Sweep_ExpireAndRemove(t *testing.T) {
	store := testutil.NewTestStore()
	var expiredIDs []string
	store.ExpireFilesFn = func() (int, error) { return 2, nil }
	store.GetExpiredFileIDsFn = func() ([]string, error) {
		expiredIDs = []string{"f_001", "f_002"}
		return expiredIDs, nil
	}

	removedFiles := []string{}
	memFS := testutil.NewMemFS()
	memFS.RemoveFn = func(name string) error {
		removedFiles = append(removedFiles, name)
		return nil
	}

	w := &Worker{
		store: store,
		cfg:   testutil.NewTestConfig(),
		log:   testutil.NewTestLogger(),
		fs:    memFS,
	}

	w.sweep()

	if len(removedFiles) != 2 {
		t.Fatalf("expected 2 removes, got %d", len(removedFiles))
	}
}

func TestWorker_Sweep_ExpireError(t *testing.T) {
	store := testutil.NewTestStore()
	store.ExpireFilesFn = func() (int, error) { return 0, errors.New("db error") }

	w := &Worker{
		store: store,
		cfg:   testutil.NewTestConfig(),
		log:   testutil.NewTestLogger(),
		fs:    testutil.NewMemFS(),
	}

	w.sweep()

}

func TestWorker_Sweep_GetExpiredIDsError(t *testing.T) {
	store := testutil.NewTestStore()
	store.ExpireFilesFn = func() (int, error) { return 0, nil }
	store.GetExpiredFileIDsFn = func() ([]string, error) { return nil, errors.New("db error") }

	w := &Worker{
		store: store,
		cfg:   testutil.NewTestConfig(),
		log:   testutil.NewTestLogger(),
		fs:    testutil.NewMemFS(),
	}

	w.sweep()

}

func TestWorker_Sweep_RemoveFileNotExist(t *testing.T) {
	store := testutil.NewTestStore()
	store.ExpireFilesFn = func() (int, error) { return 1, nil }
	store.GetExpiredFileIDsFn = func() ([]string, error) { return []string{"f_gone"}, nil }

	removeCalled := false
	memFS := testutil.NewMemFS()
	memFS.RemoveFn = func(name string) error {
		removeCalled = true
		return os.ErrNotExist
	}

	w := &Worker{
		store: store,
		cfg:   testutil.NewTestConfig(),
		log:   testutil.NewTestLogger(),
		fs:    memFS,
	}

	w.sweep()

	if !removeCalled {
		t.Error("expected Remove to be called")
	}
}

func TestWorker_Sweep_NoExpiredFiles(t *testing.T) {
	store := testutil.NewTestStore()
	store.ExpireFilesFn = func() (int, error) { return 0, nil }
	store.GetExpiredFileIDsFn = func() ([]string, error) { return nil, nil }

	w := &Worker{
		store: store,
		cfg:   testutil.NewTestConfig(),
		log:   testutil.NewTestLogger(),
		fs:    testutil.NewMemFS(),
	}

	w.sweep()

}
