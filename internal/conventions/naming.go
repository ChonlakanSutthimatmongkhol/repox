package conventions

import "strings"

func ToSnakeCase(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	var out []rune
	var prev rune
	for i, r := range s {
		if r == '/' || r == '\\' {
			out = append(out, r)
			prev = r
			continue
		}
		if r >= 'A' && r <= 'Z' {
			if i > 0 && prev != '_' && prev != '/' && prev != '\\' {
				out = append(out, '_')
			}
			r = r + ('a' - 'A')
		}
		out = append(out, r)
		prev = r
	}
	return strings.ToLower(strings.Trim(string(out), "_"))
}

func ToPascalCase(s string) string {
	parts := splitWords(s)
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, "")
}

func ToCamelCase(s string) string {
	pascal := ToPascalCase(s)
	if pascal == "" {
		return ""
	}
	return strings.ToLower(pascal[:1]) + pascal[1:]
}

func splitWords(s string) []string {
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	return strings.FieldsFunc(s, func(r rune) bool { return r == '_' })
}
