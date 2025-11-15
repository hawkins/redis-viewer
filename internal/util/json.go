package util

import (
	jsoniter "github.com/json-iterator/go"
	"regexp"

	"github.com/charmbracelet/lipgloss"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

var (
	jsonKeyStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#42B2F9")) // Blue
	jsonStringStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#73C991")) // Green
	jsonNumberStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")) // Yellow
	jsonBooleanStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF69B4")) // Hot Pink (Magenta-like)
	jsonNullStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6347")) // Tomato (Red-like)

	// Regex to match different JSON tokens
	jsonKeyRegex    = regexp.MustCompile(`("([^"\\]*(?:\\.[^"\\]*)*)")\s*:`)
	jsonStringRegex = regexp.MustCompile(`"([^"\\]*(?:\\.[^"\\]*)*)"`)
	jsonNumberRegex  = regexp.MustCompile(`\b(-?\d+(?:\.\d+)?(?:[eE][+\-]?\d+)?)\b`)
	jsonBooleanRegex = regexp.MustCompile(`\b(true|false)\b`)
	jsonNullRegex    = regexp.MustCompile(`\b(null)\b`)

	ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)
)

func TryPrettyJSON(input string) string {
	var raw interface{}
	if err := json.Unmarshal([]byte(input), &raw); err != nil {
		// Not a valid JSON, return original string
		return input
	}
	// It's a valid JSON, pretty print it
	pretty, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return input
	}

	// Apply syntax highlighting
	highlighted := string(pretty)

	highlighted = jsonNumberRegex.ReplaceAllStringFunc(highlighted, func(s string) string {
		return jsonNumberStyle.Render(s)
	})

	highlighted = jsonBooleanRegex.ReplaceAllStringFunc(highlighted, func(s string) string {
		return jsonBooleanStyle.Render(s)
	})

	highlighted = jsonNullRegex.ReplaceAllStringFunc(highlighted, func(s string) string {
		return jsonNullStyle.Render(s)
	})

	// Highlight keys first
	highlighted = jsonKeyRegex.ReplaceAllStringFunc(highlighted, func(s string) string {
		parts := jsonKeyRegex.FindStringSubmatch(s)
		if len(parts) > 1 {
			// parts[1] is the quoted key, parts[2] is the actual key name
			return jsonKeyStyle.Render(parts[1]) + ":"
		}
		return s
	})

	// Highlight string values (which are not keys)
	highlighted = jsonStringRegex.ReplaceAllStringFunc(highlighted, func(s string) string {
		// If the string already contains ANSI codes, it's likely a key that has been colored.
		if ansiRegex.MatchString(s) {
			return s
		}
		return jsonStringStyle.Render(s)
	})

	return highlighted
}

func JsonMarshal(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

func JsonMarshalIndent(data interface{}) ([]byte, error) {
	return json.MarshalIndent(data, "", "  ")
}

func JsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
