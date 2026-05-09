package learner

import "github.com/ChonlakanSutthimatmongkhol/repox/internal/models"

// Learner compares generation snapshots with current files and extracts lessons.
type Learner interface {
	ReadDiffs(generation models.Generation, baseDir string) ([]DiffResult, error)
}
