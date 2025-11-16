package dialogs

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SearchDialog handles search pattern input
type SearchDialog struct {
	input    textinput.Model
	onSubmit func(pattern string) tea.Cmd
	onCancel func() tea.Cmd
}

// NewSearchDialog creates a new search dialog
func NewSearchDialog() SearchDialog {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "Search Key"
	ti.PlaceholderStyle = lipgloss.NewStyle()

	return SearchDialog{
		input: ti,
	}
}

// SetCallbacks sets the submit and cancel callbacks
func (d *SearchDialog) SetCallbacks(onSubmit func(string) tea.Cmd, onCancel func() tea.Cmd) {
	d.onSubmit = onSubmit
	d.onCancel = onCancel
}

// Focus focuses the dialog
func (d *SearchDialog) Focus() tea.Cmd {
	return d.input.Focus()
}

// Blur blurs the dialog
func (d *SearchDialog) Blur() {
	d.input.Blur()
}

// Reset resets the dialog state
func (d *SearchDialog) Reset() {
	d.input.Reset()
}

// Value returns the current input value
func (d SearchDialog) Value() string {
	return d.input.Value()
}

// Update handles messages
func (d SearchDialog) Update(msg tea.Msg) (SearchDialog, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			d.input.Blur()
			d.input.Reset()
			if d.onCancel != nil {
				return d, d.onCancel()
			}
			return d, nil
		case tea.KeyEnter:
			value := d.input.Value()
			d.input.Blur()
			d.input.Reset()
			if d.onSubmit != nil {
				return d, d.onSubmit(value)
			}
			return d, nil
		}
	}

	d.input, cmd = d.input.Update(msg)
	return d, cmd
}

// View renders the dialog
func (d SearchDialog) View() string {
	return d.input.View()
}
