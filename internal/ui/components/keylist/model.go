package keylist

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// Model represents the key list component
type Model struct {
	list    list.Model
	focused bool
	width   int
	height  int
}

// New creates a new keylist model
func New(width, height int) Model {
	l := list.New(nil, list.NewDefaultDelegate(), width, height)
	l.Title = "Redis Viewer"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowFilter(false)
	l.SetFilteringEnabled(false)

	return Model{
		list:   l,
		width:  width,
		height: height,
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
	m.list.SetSize(width, height)
}

// SetFocus sets whether this component is focused
func (m *Model) SetFocus(focused bool) {
	m.focused = focused
}

// Focused returns whether this component is focused
func (m Model) Focused() bool {
	return m.focused
}

// SetItems sets the list items
func (m *Model) SetItems(items []list.Item) {
	m.list.SetItems(items)
}

// Items returns the current list items
func (m Model) Items() []list.Item {
	return m.list.Items()
}

// SelectedItem returns the currently selected item
func (m Model) SelectedItem() list.Item {
	return m.list.SelectedItem()
}

// Index returns the currently selected index
func (m Model) Index() int {
	return m.list.Index()
}
