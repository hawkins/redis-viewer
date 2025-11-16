package valueview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hawkins/redis-viewer/internal/styles"
	"github.com/hawkins/redis-viewer/internal/ui/components/keylist"
	"github.com/hawkins/redis-viewer/internal/util"
	"github.com/muesli/reflow/wordwrap"
)

// Model represents the value view component
type Model struct {
	viewport viewport.Model
	focused  bool
	wordWrap bool
	width    int
	height   int
}

// New creates a new valueview model
func New(width, height int) Model {
	vp := viewport.New(width, height)
	vp.MouseWheelEnabled = true

	return Model{
		viewport: vp,
		width:    width,
		height:   height,
		wordWrap: false,
	}
}

// Init initializes the component
func (m Model) Init() tea.Cmd {
	return nil
}

// SetSize updates the component size
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
}

// SetFocus sets whether this component is focused
func (m *Model) SetFocus(focused bool) {
	m.focused = focused
}

// Focused returns whether this component is focused
func (m Model) Focused() bool {
	return m.focused
}

// ToggleWordWrap toggles word wrapping
func (m *Model) ToggleWordWrap() {
	m.wordWrap = !m.wordWrap
}

// WordWrap returns whether word wrap is enabled
func (m Model) WordWrap() bool {
	return m.wordWrap
}

// GotoTop scrolls to the top
func (m *Model) GotoTop() {
	m.viewport.GotoTop()
}

// SetContent sets the viewport content
func (m *Model) SetContent(content string) {
	m.viewport.SetContent(content)
}

// FormatContent formats content for a keylist item
func (m Model) FormatContent(item keylist.Item) string {
	keyType := fmt.Sprintf("KeyType: %s", item.KeyType)
	width := m.width
	wrappedKey := wordwrap.String(item.Key, width)
	key := fmt.Sprintf("%s", wrappedKey)
	divider := styles.DividerStyle.Render(strings.Repeat("-", width))

	var value string
	if !item.Loaded {
		// Value not loaded yet - show loading message
		value = styles.LoadingStyle.Render("Loading value...")
	} else {
		formattedValue := util.TryPrettyJSON(item.Val)
		// Apply word wrapping if enabled
		if m.wordWrap {
			formattedValue = wordwrap.String(formattedValue, width)
		}
		value = fmt.Sprintf("%s", formattedValue)
	}

	content := []string{keyType}
	if item.TTLSeconds > 0 {
		ttlFormatted := formatTTLSeconds(item.TTLSeconds)
		content = append(content, fmt.Sprintf("TTL: %s (%d seconds)", ttlFormatted, item.TTLSeconds))
	}

	content = append(content, divider, key, divider, value)

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

// formatTTLSeconds formats TTL in seconds to a human-readable format
func formatTTLSeconds(seconds int64) string {
	if seconds <= 0 {
		return "" // No expiration or already expired
	}

	days := seconds / 86400
	remaining := seconds % 86400
	hours := remaining / 3600
	remaining %= 3600
	minutes := remaining / 60
	secs := remaining % 60

	var parts []string
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 || len(parts) > 0 {
		parts = append(parts, fmt.Sprintf("%02dh", hours))
	}
	if minutes > 0 || len(parts) > 0 {
		parts = append(parts, fmt.Sprintf("%02dm", minutes))
	}
	if secs > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%02ds", secs))
	}

	return strings.Join(parts, " ")
}
