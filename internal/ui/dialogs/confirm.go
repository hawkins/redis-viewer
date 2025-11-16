package dialogs

import (
	tea "github.com/charmbracelet/bubbletea"
)

// ConfirmType represents different types of confirmation
type ConfirmType int

const (
	ConfirmDelete ConfirmType = iota
	ConfirmPurge
)

// ConfirmDialog handles yes/no confirmation
type ConfirmDialog struct {
	confirmType ConfirmType
	data        interface{} // Can store key name, db number, etc.
	onConfirm   func() tea.Cmd
	onCancel    func() tea.Cmd
}

// NewConfirmDialog creates a new confirmation dialog
func NewConfirmDialog(confirmType ConfirmType, data interface{}) ConfirmDialog {
	return ConfirmDialog{
		confirmType: confirmType,
		data:        data,
	}
}

// SetCallbacks sets the confirm and cancel callbacks
func (d *ConfirmDialog) SetCallbacks(onConfirm func() tea.Cmd, onCancel func() tea.Cmd) {
	d.onConfirm = onConfirm
	d.onCancel = onCancel
}

// Type returns the confirmation type
func (d ConfirmDialog) Type() ConfirmType {
	return d.confirmType
}

// Data returns the associated data
func (d ConfirmDialog) Data() interface{} {
	return d.data
}

// Update handles messages
func (d ConfirmDialog) Update(msg tea.Msg) (ConfirmDialog, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			// Confirm
			if d.onConfirm != nil {
				return d, d.onConfirm()
			}
			return d, nil
		case "n", "N", "esc":
			// Cancel
			if d.onCancel != nil {
				return d, d.onCancel()
			}
			return d, nil
		}
	}

	return d, nil
}

// View renders the dialog - this is just a placeholder, actual rendering
// happens in the status bar
func (d ConfirmDialog) View() string {
	return ""
}
