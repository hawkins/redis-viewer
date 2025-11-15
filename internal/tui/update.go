// @file: update.go
// @date: 2022/02/08

package tui

import (
	"fmt"

	"github.com/saltfishpr/redis-viewer/internal/constant"
	"github.com/spf13/cast"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// global msg handling
	switch msg := msg.(type) {
	case deleteMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to delete key: %v", msg.err)
		} else {
			m.statusMessage = fmt.Sprintf("Key '%s' deleted successfully", msg.key)
			m.ready = false
			cmds = append(cmds, m.scanCmd(), m.countCmd())
		}
	case errMsg:
		m.statusMessage = msg.err.Error()
		// TODO: log error
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		statusBarHeight := lipgloss.Height(m.statusView())
		height := m.height - statusBarHeight

		listViewWidth := cast.ToInt(constant.ListProportion * float64(m.width))
		listWidth := listViewWidth - listViewStyle.GetHorizontalFrameSize()
		m.list.SetSize(listWidth, height)

		detailViewWidth := m.width - listViewWidth
		m.viewport = viewport.New(detailViewWidth, height)
		m.viewport.MouseWheelEnabled = true
		m.viewport.SetContent(m.viewportContent())
	case tickMsg:
		m.now = msg.t
		cmds = append(cmds, m.tickCmd())
	case scanMsg:
		m.list.SetItems(msg.items)
	case countMsg:
		if msg.count > constant.MaxScanCount {
			m.statusMessage = fmt.Sprintf("%d+ keys found", constant.MaxScanCount)
		} else {
			m.statusMessage = fmt.Sprintf("%d keys found", msg.count)
		}
		m.ready = true
	}

	switch m.state {
	case defaultState:
		cmds = append(cmds, m.handleDefaultState(msg))
	case searchState:
		cmds = append(cmds, m.handleSearchState(msg))
	case confirmDeleteState:
		cmds = append(cmds, m.handleConfirmDeleteState(msg))
	case helpState:
		cmds = append(cmds, m.handleHelpState(msg))
	}

	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *model) handleDefaultState(msg tea.Msg) tea.Cmd {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.MouseMsg:
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyRunes:
			switch {
			case key.Matches(msg, m.keyMap.search):
				m.state = searchState
				m.textinput.Focus()
				return textinput.Blink
			case key.Matches(msg, m.keyMap.reload):
				m.ready = false
				cmds = append(cmds, m.scanCmd(), m.countCmd())
			case key.Matches(msg, m.keyMap.delete):
				// Get the selected item
				if selectedItem := m.list.SelectedItem(); selectedItem != nil {
					if i, ok := selectedItem.(item); ok {
						m.keyToDelete = i.key
						m.state = confirmDeleteState
					}
				}
			case key.Matches(msg, m.keyMap.help):
				m.state = helpState
			}
		case tea.KeyCtrlC:
			cmd = tea.Quit
			cmds = append(cmds, cmd)
		case tea.KeyUp, tea.KeyDown, tea.KeyLeft, tea.KeyRight:
			m.list, cmd = m.list.Update(msg)
			cmds = append(cmds, cmd)
			m.viewport.GotoTop()
			m.viewport.SetContent(m.viewportContent())
		}
	default:
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)

		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return tea.Batch(cmds...)
}

func (m *model) handleSearchState(msg tea.Msg) tea.Cmd {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			m.textinput.Blur()
			m.textinput.Reset()
			m.state = defaultState
			// Don't update textinput after state change
			return tea.Batch(cmds...)
		case tea.KeyEnter:
			m.searchValue = m.textinput.Value()

			m.textinput.Blur()
			m.textinput.Reset()
			m.state = defaultState

			m.ready = false
			cmds = append(cmds, m.scanCmd(), m.countCmd())
			// Don't update textinput after state change
			return tea.Batch(cmds...)
		}
	}

	m.textinput, cmd = m.textinput.Update(msg)
	cmds = append(cmds, cmd)

	return tea.Batch(cmds...)
}

func (m *model) handleConfirmDeleteState(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			// Confirm deletion
			m.state = defaultState
			cmds = append(cmds, m.deleteCmd(m.keyToDelete))
		case "n", "N", "esc":
			// Cancel deletion
			m.state = defaultState
			m.keyToDelete = ""
		}
	}

	return tea.Batch(cmds...)
}

func (m *model) handleHelpState(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "?", "esc":
			// Close help dialog
			m.state = defaultState
		}
	}

	return tea.Batch(cmds...)
}
