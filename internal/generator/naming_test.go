package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToSnakeCase(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"watchList", "watch_list"},
		{"WatchList", "watch_list"},
		{"watch-list", "watch_list"},
		{"watch_list", "watch_list"},
		{"watchlist", "watchlist"},
		{"Watchlist", "watchlist"},
		{"", ""},
		{"MyFeatureName", "my_feature_name"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.want, ToSnakeCase(tc.input))
		})
	}
}

func TestToPascalCase(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"watchlist", "Watchlist"},
		{"watch-list", "WatchList"},
		{"watch_list", "WatchList"},
		{"watchList", "WatchList"},
		{"WatchList", "WatchList"},
		{"", ""},
		{"my_feature_name", "MyFeatureName"},
		{"my-feature-name", "MyFeatureName"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.want, ToPascalCase(tc.input))
		})
	}
}

func TestToCamelCase(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"watchlist", "watchlist"},
		{"watch-list", "watchList"},
		{"watch_list", "watchList"},
		{"watchList", "watchList"},
		{"WatchList", "watchList"},
		{"", ""},
		{"my_feature_name", "myFeatureName"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.want, ToCamelCase(tc.input))
		})
	}
}
