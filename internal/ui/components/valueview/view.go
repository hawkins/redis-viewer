package valueview

import (
	"github.com/hawkins/redis-viewer/internal/styles"
)

// View renders the valueview component
func (m Model) View() string {
	content := m.viewport.View()

	// Add a focus indicator at the top when focused
	if m.focused {
		indicator := styles.FocusIndicator.Render("▸ ")
		content = indicator + content
	}

	return content
}
