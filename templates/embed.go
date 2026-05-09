// Package templates embeds the scaffold template files.
package templates

import "embed"

//go:embed flutter_bloc_feature go_clean_feature
var FS embed.FS
