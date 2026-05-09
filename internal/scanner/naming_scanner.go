package scanner

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

// DetectNamingConventions walks featureRoot, counts file suffix patterns,
// and returns the majority-vote naming convention for each role.
func DetectNamingConventions(featureRoot, projectType string) (models.NamingConvention, error) {
	conv := models.NamingConvention{
		ClassCase: "PascalCase",
		FileCase:  "snake_case",
	}

	switch projectType {
	case "flutter", "dart":
		return detectFlutterNaming(featureRoot, conv)
	case "go":
		return detectGoNaming(featureRoot, conv)
	default:
		return conv, nil
	}
}

func detectFlutterNaming(featureRoot string, conv models.NamingConvention) (models.NamingConvention, error) {
	screenVotes := map[string]int{"Screen": 0, "Page": 0, "View": 0}
	blocVotes := map[string]int{"Bloc": 0, "Cubit": 0}
	repoVotes := map[string]int{"Repository": 0, "Repo": 0}
	usecaseVotes := map[string]int{"UseCase": 0}
	eventVotes := map[string]int{"Event": 0}
	stateVotes := map[string]int{"State": 0}

	_ = filepath.WalkDir(featureRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".dart") {
			return nil
		}
		base := strings.TrimSuffix(filepath.Base(path), ".dart")
		switch {
		case strings.HasSuffix(base, "_screen"):
			screenVotes["Screen"]++
		case strings.HasSuffix(base, "_page"):
			screenVotes["Page"]++
		case strings.HasSuffix(base, "_view"):
			screenVotes["View"]++
		case strings.HasSuffix(base, "_bloc"):
			blocVotes["Bloc"]++
		case strings.HasSuffix(base, "_cubit"):
			blocVotes["Cubit"]++
		case strings.HasSuffix(base, "_event"):
			eventVotes["Event"]++
		case strings.HasSuffix(base, "_state"):
			stateVotes["State"]++
		case strings.HasSuffix(base, "_repository"):
			repoVotes["Repository"]++
		case strings.HasSuffix(base, "_repo"):
			repoVotes["Repo"]++
		case strings.HasSuffix(base, "_usecase") || strings.HasSuffix(base, "_use_case"):
			usecaseVotes["UseCase"]++
		}
		return nil
	})

	conv.ScreenSuffix = majorityVote(screenVotes, "Screen")
	conv.BlocSuffix = majorityVote(blocVotes, "Bloc")
	conv.RepositorySuffix = majorityVote(repoVotes, "Repository")
	conv.UsecaseSuffix = majorityVote(usecaseVotes, "UseCase")
	conv.EventSuffix = majorityVote(eventVotes, "Event")
	conv.StateSuffix = majorityVote(stateVotes, "State")
	return conv, nil
}

func detectGoNaming(featureRoot string, conv models.NamingConvention) (models.NamingConvention, error) {
	handlerVotes := map[string]int{"Handler": 0, "Controller": 0}
	serviceVotes := map[string]int{"Service": 0, "UseCase": 0}
	repoVotes := map[string]int{"Repository": 0, "Repo": 0, "Store": 0}

	_ = filepath.WalkDir(featureRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		base := strings.TrimSuffix(filepath.Base(path), ".go")
		base = strings.TrimSuffix(base, "_test")
		switch {
		case strings.HasSuffix(base, "_handler"):
			handlerVotes["Handler"]++
		case strings.HasSuffix(base, "_controller"):
			handlerVotes["Controller"]++
		case strings.HasSuffix(base, "_service"):
			serviceVotes["Service"]++
		case strings.HasSuffix(base, "_usecase") || strings.HasSuffix(base, "_use_case"):
			serviceVotes["UseCase"]++
		case strings.HasSuffix(base, "_repository"):
			repoVotes["Repository"]++
		case strings.HasSuffix(base, "_repo"):
			repoVotes["Repo"]++
		case strings.HasSuffix(base, "_store"):
			repoVotes["Store"]++
		}
		return nil
	})

	conv.HandlerSuffix = majorityVote(handlerVotes, "Handler")
	conv.ServiceSuffix = majorityVote(serviceVotes, "Service")
	conv.RepositorySuffix = majorityVote(repoVotes, "Repository")
	return conv, nil
}

// majorityVote returns the key with the highest count, or defaultVal if all are zero.
func majorityVote(votes map[string]int, defaultVal string) string {
	best, bestCount := defaultVal, -1
	for k, v := range votes {
		if v > bestCount {
			best, bestCount = k, v
		}
	}
	if bestCount <= 0 {
		return defaultVal
	}
	return best
}
