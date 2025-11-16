package keylist

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles messages for the keylist component
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	// Only handle updates if focused or if it's a window size message
	switch msg.(type) {
	case tea.WindowSizeMsg:
		// Always handle window size
		m.list, cmd = m.list.Update(msg)
	default:
		if m.focused {
			m.list, cmd = m.list.Update(msg)
		}
	}

	return m, cmd
}
