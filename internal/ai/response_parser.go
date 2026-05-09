package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseResponse parses the raw AI text output into a GenerateResponse.
// It strips markdown code fences if present and validates each file entry.
func ParseResponse(raw string) (*GenerateResponse, error) {
	cleaned := stripFences(strings.TrimSpace(raw))

	var resp GenerateResponse
	if err := json.Unmarshal([]byte(cleaned), &resp); err != nil {
		return nil, fmt.Errorf("response parser: invalid JSON: %w\nraw: %.300s", err, raw)
	}

	for i, f := range resp.Files {
		if f.Path == "" {
			return nil, fmt.Errorf("response parser: file[%d] has empty path", i)
		}
		if f.Content == "" {
			return nil, fmt.Errorf("response parser: file[%d] (%s) has empty content", i, f.Path)
		}
	}

	if len(resp.Files) == 0 {
		return nil, fmt.Errorf("response parser: no files in response")
	}

	return &resp, nil
}

// stripFences removes leading/trailing ```json ... ``` or ``` ... ``` wrappers.
func stripFences(s string) string {
	// Remove leading fence
	for _, fence := range []string{"```json", "```"} {
		if strings.HasPrefix(s, fence) {
			s = strings.TrimPrefix(s, fence)
			// Remove trailing fence
			if idx := strings.LastIndex(s, "```"); idx != -1 {
				s = s[:idx]
			}
			return strings.TrimSpace(s)
		}
	}
	return s
}
