package dialogs

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FilterDialog handles fuzzy/strict filter input
type FilterDialog struct {
	input       textinput.Model
	strictMode  bool
	onSubmit    func(pattern string) tea.Cmd
	onCancel    func() tea.Cmd
	onToggleMode func() tea.Cmd
}

// NewFilterDialog creates a new filter dialog
func NewFilterDialog() FilterDialog {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "Fuzzy Filter"
	ti.PlaceholderStyle = lipgloss.NewStyle()

	return FilterDialog{
		input:      ti,
		strictMode: false,
	}
}

// SetCallbacks sets the callbacks
func (d *FilterDialog) SetCallbacks(onSubmit func(string) tea.Cmd, onCancel func() tea.Cmd, onToggleMode func() tea.Cmd) {
	d.onSubmit = onSubmit
	d.onCancel = onCancel
	d.onToggleMode = onToggleMode
}

// Focus focuses the dialog
func (d *FilterDialog) Focus() tea.Cmd {
	return d.input.Focus()
}

// Blur blurs the dialog
func (d *FilterDialog) Blur() {
	d.input.Blur()
}

// Reset resets the dialog state
func (d *FilterDialog) Reset() {
	d.input.Reset()
}

// SetValue sets the input value
func (d *FilterDialog) SetValue(value string) {
	d.input.SetValue(value)
}

// Value returns the current input value
func (d FilterDialog) Value() string {
	return d.input.Value()
}

// SetStrictMode sets the strict mode flag
func (d *FilterDialog) SetStrictMode(strict bool) {
	d.strictMode = strict
}

// StrictMode returns whether strict mode is enabled
func (d FilterDialog) StrictMode() bool {
	return d.strictMode
}

// Update handles messages
func (d FilterDialog) Update(msg tea.Msg) (FilterDialog, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			// Let quit handler take over
			return d, nil
		case tea.KeyCtrlF:
			// Toggle between fuzzy and strict mode
			if d.onToggleMode != nil {
				return d, d.onToggleMode()
			}
			return d, nil
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
func (d FilterDialog) View() string {
	return d.input.View()
}
