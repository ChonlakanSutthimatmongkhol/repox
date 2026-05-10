package scanner

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

// InferRoleConventions learns role naming from scanned feature files.
func InferRoleConventions(features []models.FeatureAnalysis, naming models.NamingConvention) map[string]models.RoleConvention {
	type votes map[string]int
	fileVotes := map[string]votes{}
	addVote := func(role, suffix string) {
		if role == "" || suffix == "" {
			return
		}
		if fileVotes[role] == nil {
			fileVotes[role] = votes{}
		}
		fileVotes[role][suffix]++
	}

	for _, feature := range features {
		featureSnake := toSnakeCase(feature.Name)
		for role, path := range feature.Files {
			suffix := fileSuffixForFeature(path, featureSnake)
			addVote(role, suffix)
		}
		for role, anatomy := range feature.Anatomy {
			suffix := fileSuffixForFeature(anatomy.Path, featureSnake)
			addVote(role, suffix)
		}
	}

	roles := map[string]models.RoleConvention{}
	for role, suffix := range defaultClassSuffixes(naming) {
		roles[role] = models.RoleConvention{
			FileSuffix:  defaultFileSuffix(role, naming),
			ClassSuffix: suffix,
		}
	}
	for role, votes := range fileVotes {
		rc := roles[role]
		if suffix := majorityString(votes); suffix != "" {
			rc.FileSuffix = suffix
			if rc.ClassSuffix == "" {
				rc.ClassSuffix = toPascalCase(suffix)
			}
		}
		roles[role] = rc
	}
	return roles
}

func fileSuffixForFeature(path, featureSnake string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if base == "" {
		return ""
	}
	if featureSnake != "" && strings.HasPrefix(base, featureSnake+"_") {
		return strings.TrimPrefix(base, featureSnake+"_")
	}
	return base
}

func majorityString(votes map[string]int) string {
	best, bestCount := "", -1
	var keys []string
	for key := range votes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if votes[key] > bestCount {
			best, bestCount = key, votes[key]
		}
	}
	return best
}

func defaultClassSuffixes(naming models.NamingConvention) map[string]string {
	out := map[string]string{}
	add := func(role, suffix string) {
		if suffix != "" {
			out[role] = suffix
		}
	}
	add("screen", naming.ScreenSuffix)
	add("bloc", naming.BlocSuffix)
	add("event", naming.EventSuffix)
	add("state", naming.StateSuffix)
	add("repository", naming.RepositorySuffix)
	add("usecase", naming.UsecaseSuffix)
	add("handler", naming.HandlerSuffix)
	add("service", naming.ServiceSuffix)
	return out
}

func defaultFileSuffix(role string, naming models.NamingConvention) string {
	for suffix, mappedRole := range naming.SuffixRoles {
		if mappedRole == role {
			return strings.ToLower(suffix)
		}
	}
	return role
}

func toSnakeCase(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	var out []rune
	var prev rune
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 && prev != '_' {
				out = append(out, '_')
			}
			r = r + ('a' - 'A')
		}
		out = append(out, r)
		prev = r
	}
	return strings.ToLower(strings.Trim(string(out), "_"))
}

func toPascalCase(s string) string {
	parts := strings.FieldsFunc(strings.ReplaceAll(strings.ReplaceAll(s, "-", "_"), " ", "_"), func(r rune) bool {
		return r == '_'
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, "")
}
