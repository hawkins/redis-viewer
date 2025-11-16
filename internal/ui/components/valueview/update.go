package valueview

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles messages for the valueview component
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	// Always handle mouse messages for viewport scrolling
	switch msg.(type) {
	case tea.MouseMsg, tea.WindowSizeMsg:
		m.viewport, cmd = m.viewport.Update(msg)
	default:
		// Only handle other updates if focused
		if m.focused {
			m.viewport, cmd = m.viewport.Update(msg)
		}
	}

	return m, cmd
}
