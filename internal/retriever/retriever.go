// Package retriever indexes existing features and ranks them by similarity.
package retriever

import "github.com/ChonlakanSutthimatmongkhol/repox/internal/models"

// Retriever finds and ranks existing features similar to a target feature.
type Retriever interface {
	Index(rootDir string, conv *models.Convention) ([]models.Example, error)
	FindSimilar(target string, examples []models.Example, topN int) []models.Example
}
