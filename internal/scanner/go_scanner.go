package scanner

import (
	"fmt"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

// GoScanner implements Scanner for Go projects (stub for v0.2.0).
type GoScanner struct{}

// Scan is not yet implemented for Go projects.
func (s *GoScanner) Scan(_ string) (*models.Convention, error) {
	return nil, fmt.Errorf("go scanner not yet implemented")
}
