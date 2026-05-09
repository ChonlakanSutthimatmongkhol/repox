package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/config"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

func TestPlanFeature_WithRolesAndAnatomy(t *testing.T) {
	setupInitDir(t)

	conv, err := config.Load[models.Convention](config.RepoxPath("conventions.json"))
	require.NoError(t, err)
	conv.FeaturesAnalysis.RecommendedPattern = "clean_architecture"
	conv.FeaturesAnalysis.RoleAnatomy = map[string]models.RoleAnatomy{
		"bloc": {
			FeatureCount: 4,
			BaseClasses:  []models.AnatomyVote{{Name: "BaseBloc", Count: 3, Percentage: 75}},
			Methods:      []models.AnatomyVote{{Name: "_trackLandingEvent", Count: 2, Percentage: 50}},
			Capabilities: []models.AnatomyVote{
				{Name: "firebase_tracking", Count: 3, Percentage: 75},
			},
		},
		"screen": {
			FeatureCount: 4,
			BaseClasses:  []models.AnatomyVote{{Name: "BaseStatefulWidget", Count: 3, Percentage: 75}},
		},
	}
	require.NoError(t, config.Save(config.RepoxPath("conventions.json"), conv))

	planRoles = "bloc,screen"
	planLike = ""
	defer func() {
		planRoles = ""
		planLike = ""
	}()

	buf := &bytes.Buffer{}
	planFeatureCmd.SetOut(buf)
	err = runPlanFeature(planFeatureCmd, []string{"investment/new_feature"})
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Feature plan: investment/new_feature")
	assert.Contains(t, out, "Roles:   bloc, screen")
	assert.Contains(t, out, "new_feature_bloc.dart")
	assert.Contains(t, out, "new_feature_screen.dart")
	assert.Contains(t, out, "base BaseBloc")
	assert.Contains(t, out, "firebase_tracking")
	assert.NotContains(t, out, "new_feature_repository.dart")
}
