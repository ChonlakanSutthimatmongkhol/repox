package mapgen

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

func TestGenerateProjectMarkdown(t *testing.T) {
	conv := models.Convention{
		ProjectType:      "flutter",
		FeatureRoot:      "lib/features",
		FeatureStructure: "grouped",
		FeaturesAnalysis: models.FeaturesAnalysis{
			Features: []models.FeatureAnalysis{
				{Path: "lib/features/investment/fund_list", Files: map[string]string{"bloc": "fund_list_bloc.dart", "screen": "fund_list_screen.dart"}},
			},
		},
	}

	out := GenerateProjectMarkdown(conv)

	assert.Contains(t, out, "# Repox Project Map")
	assert.Contains(t, out, "Project type: flutter")
	assert.Contains(t, out, "lib/features/investment/fund_list")
}

func TestGenerateFeatureMarkdown(t *testing.T) {
	conv := models.Convention{
		FeatureRoot: "lib/features",
		FeaturesAnalysis: models.FeaturesAnalysis{
			Features: []models.FeatureAnalysis{
				{Name: "fund_list", Path: "lib/features/investment/fund_list", Files: map[string]string{"bloc": "lib/features/investment/fund_list/fund_list_bloc.dart"}},
			},
		},
	}

	name, out, ok := GenerateFeatureMarkdown(conv, "investment/fund_list")

	assert.True(t, ok)
	assert.Equal(t, "investment_fund_list", name)
	assert.Contains(t, out, "Feature Map")
	assert.Contains(t, out, "bloc")
}
