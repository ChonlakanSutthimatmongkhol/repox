package learner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

// ── ReadDiffs ─────────────────────────────────────────────────────────────────

func TestReadDiffs_NoSnapshotDir(t *testing.T) {
	gen := models.Generation{ID: "gen_1", SnapshotDir: ""}
	_, err := ReadDiffs(gen, t.TempDir())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no snapshot")
}

func TestReadDiffs_FileUnchanged(t *testing.T) {
	base := t.TempDir()
	snapDir := filepath.Join(base, ".repox", "snapshots", "gen_1")
	relPath := "lib/features/home/home_bloc.dart"
	content := "class HomeBloc {}"

	writeFile(t, filepath.Join(snapDir, relPath), content)
	writeFile(t, filepath.Join(base, relPath), content)

	gen := models.Generation{
		ID:          "gen_1",
		SnapshotDir: snapDir,
		Files:       []string{relPath},
	}
	diffs, err := ReadDiffs(gen, base)
	require.NoError(t, err)
	require.Len(t, diffs, 1)
	assert.False(t, diffs[0].Changed)
	assert.Empty(t, diffs[0].Diff)
}

func TestReadDiffs_FileChanged(t *testing.T) {
	base := t.TempDir()
	snapDir := filepath.Join(base, ".repox", "snapshots", "gen_1")
	relPath := "lib/features/home/home_bloc.dart"

	writeFile(t, filepath.Join(snapDir, relPath), "class HomeBloc {}")
	writeFile(t, filepath.Join(base, relPath), "class HomeBloc extends BaseBloc {}")

	gen := models.Generation{
		ID:          "gen_1",
		SnapshotDir: snapDir,
		Files:       []string{relPath},
	}
	diffs, err := ReadDiffs(gen, base)
	require.NoError(t, err)
	require.Len(t, diffs, 1)
	assert.True(t, diffs[0].Changed)
	assert.NotEmpty(t, diffs[0].Diff)
}

func TestReadDiffs_MissingCurrentFile(t *testing.T) {
	base := t.TempDir()
	snapDir := filepath.Join(base, ".repox", "snapshots", "gen_1")
	relPath := "lib/features/home/home_bloc.dart"

	writeFile(t, filepath.Join(snapDir, relPath), "class HomeBloc {}")
	// current file NOT created — simulates deletion

	gen := models.Generation{
		ID:          "gen_1",
		SnapshotDir: snapDir,
		Files:       []string{relPath},
	}
	diffs, err := ReadDiffs(gen, base)
	require.NoError(t, err)
	assert.Empty(t, diffs) // deleted files are skipped
}

func TestReadDiffs_MissingSnapshot(t *testing.T) {
	base := t.TempDir()
	relPath := "lib/features/home/home_bloc.dart"
	writeFile(t, filepath.Join(base, relPath), "class HomeBloc {}")

	gen := models.Generation{
		ID:          "gen_1",
		SnapshotDir: filepath.Join(base, "nonexistent_snapshot"),
		Files:       []string{relPath},
	}
	diffs, err := ReadDiffs(gen, base)
	require.NoError(t, err)
	assert.Empty(t, diffs) // snapshot missing — skipped
}

// ── unifiedDiff ───────────────────────────────────────────────────────────────

func TestUnifiedDiff_ShowsChanges(t *testing.T) {
	diff := unifiedDiff("a.dart", "line1\nline2", "line1\nline3")
	assert.Contains(t, diff, "-line2")
	assert.Contains(t, diff, "+line3")
	assert.Contains(t, diff, "a.dart")
}

func TestUnifiedDiff_NoChanges(t *testing.T) {
	diff := unifiedDiff("a.dart", "same", "same")
	assert.Contains(t, diff, " same")
	assert.NotContains(t, diff, "-same")
	assert.NotContains(t, diff, "+same")
}
