package learner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/ai"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

// ── mock Caller ───────────────────────────────────────────────────────────────

type mockCaller struct {
	response string
	err      error
}

func (m *mockCaller) Call(_, _ string) (string, error) { return m.response, m.err }

var _ ai.Caller = (*mockCaller)(nil)

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

// ── ExtractLessons ────────────────────────────────────────────────────────────

func TestExtractLessons_NoChanges(t *testing.T) {
	diffs := []DiffResult{{Path: "a.dart", Changed: false}}
	mc := &mockCaller{response: `{"lessons":[]}`}
	lessons, err := ExtractLessons(diffs, "gen_1", "home", "flutter_bloc_feature", mc)
	require.NoError(t, err)
	assert.Empty(t, lessons)
}

func TestExtractLessons_WithChanges(t *testing.T) {
	diffs := []DiffResult{
		{Path: "a.dart", Changed: true, GeneratedCode: "class X {}", CurrentCode: "class X extends BaseBloc {}", Diff: "+extends BaseBloc"},
	}
	mc := &mockCaller{
		response: `{"lessons":[{"scope":"flutter_bloc_feature","lesson":"extend BaseBloc instead of nothing","confidence":0.9}]}`,
	}
	lessons, err := ExtractLessons(diffs, "gen_1", "home", "flutter_bloc_feature", mc)
	require.NoError(t, err)
	require.Len(t, lessons, 1)
	assert.Equal(t, "extend BaseBloc instead of nothing", lessons[0].Lesson)
	assert.Equal(t, 0.9, lessons[0].Confidence)
	assert.False(t, lessons[0].Approved)
	assert.Equal(t, "gen_1", lessons[0].Source.GenerationID)
}

func TestExtractLessons_InvalidJSON(t *testing.T) {
	diffs := []DiffResult{{Path: "a.dart", Changed: true, GeneratedCode: "x", CurrentCode: "y", Diff: "+y"}}
	mc := &mockCaller{response: "not json"}
	_, err := ExtractLessons(diffs, "gen_1", "home", "flutter_bloc_feature", mc)
	assert.Error(t, err)
}

// ── ParseExtractionJSON ───────────────────────────────────────────────────────

func TestParseExtractionJSON_Valid(t *testing.T) {
	raw := `{"lessons":[{"scope":"flutter_bloc_feature","lesson":"use BaseBloc","confidence":0.8}]}`
	resp, err := ParseExtractionJSON(raw)
	require.NoError(t, err)
	require.Len(t, resp.Lessons, 1)
	assert.Equal(t, "use BaseBloc", resp.Lessons[0].Lesson)
}

func TestParseExtractionJSON_WithFence(t *testing.T) {
	raw := "```json\n{\"lessons\":[{\"scope\":\"global\",\"lesson\":\"x\",\"confidence\":0.5}]}\n```"
	resp, err := ParseExtractionJSON(raw)
	require.NoError(t, err)
	assert.Len(t, resp.Lessons, 1)
}

func TestParseExtractionJSON_Invalid(t *testing.T) {
	_, err := ParseExtractionJSON("invalid")
	assert.Error(t, err)
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
