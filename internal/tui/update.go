// @file: update.go
// @date: 2022/02/08

package tui

import (
	"fmt"

	"github.com/go-redis/redis/v8"
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
	case statsMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to load stats: %v", msg.err)
			m.statsData = &statsData{loading: false, err: msg.err}
		} else {
			m.statsData = &statsData{
				serverStats: msg.serverStats,
				dbStats:     msg.dbStats,
				loading:     false,
				err:         nil,
			}
		}
	case deleteMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to delete key: %v", msg.err)
		} else {
			m.statusMessage = fmt.Sprintf("Key '%s' deleted successfully", msg.key)
			m.ready = false
			cmds = append(cmds, m.scanCmd(), m.countCmd())
		}
	case purgeMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to purge database: %v", msg.err)
		} else {
			m.statusMessage = fmt.Sprintf("Database %d purged successfully", msg.db)
			m.ready = false
			cmds = append(cmds, m.scanCmd(), m.countCmd())
		}
	case switchDBMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to switch database: %v", msg.err)
		} else {
			// Close the old client
			if m.rdb != nil {
				_ = m.rdb.Close()
			}
			// Set the new client
			m.rdb = msg.newRdb.(redis.UniversalClient)
			m.db = msg.db
			m.statusMessage = fmt.Sprintf("Switched to database %d", msg.db)
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
			m.statusMessage = fmt.Sprintf("DB %d: %d+ keys found", m.db, constant.MaxScanCount)
		} else {
			m.statusMessage = fmt.Sprintf("DB %d: %d keys found", m.db, msg.count)
		}
		m.ready = true
	}

	switch m.state {
	case defaultState:
		cmds = append(cmds, m.handleDefaultState(msg))
	case searchState:
		cmds = append(cmds, m.handleSearchState(msg))
	case fuzzySearchState:
		cmds = append(cmds, m.handleFuzzySearchState(msg))
	case switchDBState:
		cmds = append(cmds, m.handleSwitchDBState(msg))
	case confirmDeleteState:
		cmds = append(cmds, m.handleConfirmDeleteState(msg))
	case confirmPurgeState:
		cmds = append(cmds, m.handleConfirmPurgeState(msg))
	case helpState:
		cmds = append(cmds, m.handleHelpState(msg))
	case statsState:
		cmds = append(cmds, m.handleStatsState(msg))
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
			case key.Matches(msg, m.keyMap.fuzzySearch):
				m.state = fuzzySearchState
				m.fuzzyInput.SetValue(m.fuzzyFilter)
				m.fuzzyInput.Focus()
				return textinput.Blink
			case key.Matches(msg, m.keyMap.switchDB):
				m.state = switchDBState
				m.dbInput.Focus()
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
			case key.Matches(msg, m.keyMap.purge):
				// Enter purge confirmation state
				m.state = confirmPurgeState
			case key.Matches(msg, m.keyMap.toggleWrap):
				m.wordWrap = !m.wordWrap
				if m.wordWrap {
					m.statusMessage = "Word wrap enabled"
				} else {
					m.statusMessage = "Word wrap disabled"
				}
				// Update viewport content to reflect the change
				m.viewport.SetContent(m.viewportContent())
			case key.Matches(msg, m.keyMap.help):
				m.state = helpState
			case key.Matches(msg, m.keyMap.stats):
				// Enter stats state and start loading stats
				m.state = statsState
				m.statsData = &statsData{loading: true}
				cmds = append(cmds, m.statsCmd())
			}
		case tea.KeyCtrlC:
			cmd = tea.Quit
			cmds = append(cmds, cmd)
		case tea.KeyCtrlF:
			// Toggle between fuzzy and strict mode
			m.fuzzyStrict = !m.fuzzyStrict
			// Update status message to reflect mode change
			if m.fuzzyStrict {
				m.statusMessage = "Switched to strict mode"
			} else {
				m.statusMessage = "Switched to fuzzy mode"
			}
			// Re-scan if there's an active filter
			if m.fuzzyFilter != "" {
				m.ready = false
				cmds = append(cmds, m.scanCmd(), m.countCmd())
			}
		case tea.KeyLeft:
			// Switch focus to list pane
			m.focused = listPane
		case tea.KeyRight:
			// Switch focus to viewport pane
			m.focused = viewportPane
		case tea.KeyUp, tea.KeyDown:
			// Handle up/down based on which pane is focused
			if m.focused == listPane {
				// Navigate the list
				m.list, cmd = m.list.Update(msg)
				cmds = append(cmds, cmd)
				m.viewport.GotoTop()
				m.viewport.SetContent(m.viewportContent())
			} else {
				// Scroll the viewport
				m.viewport, cmd = m.viewport.Update(msg)
				cmds = append(cmds, cmd)
			}
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

func (m *model) handleFuzzySearchState(msg tea.Msg) tea.Cmd {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			cmd = tea.Quit
			cmds = append(cmds, cmd)
			return tea.Batch(cmds...)
		case tea.KeyCtrlF:
			// Toggle between fuzzy and strict mode
			m.fuzzyStrict = !m.fuzzyStrict
			// Don't update fuzzyInput when toggling mode
			return tea.Batch(cmds...)
		case tea.KeyEscape:
			m.fuzzyInput.Blur()
			m.fuzzyInput.Reset()
			m.state = defaultState
			// Don't update fuzzyInput after state change
			return tea.Batch(cmds...)
		case tea.KeyEnter:
			m.fuzzyFilter = m.fuzzyInput.Value()

			m.fuzzyInput.Blur()
			m.state = defaultState

			m.ready = false
			cmds = append(cmds, m.scanCmd(), m.countCmd())
			// Don't update fuzzyInput after state change
			return tea.Batch(cmds...)
		}
	}

	m.fuzzyInput, cmd = m.fuzzyInput.Update(msg)
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

func (m *model) handleConfirmPurgeState(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			// Confirm purge
			m.state = defaultState
			cmds = append(cmds, m.purgeCmd())
		case "n", "N", "esc":
			// Cancel purge
			m.state = defaultState
		}
	}

	return tea.Batch(cmds...)
}

func (m *model) handleSwitchDBState(msg tea.Msg) tea.Cmd {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			m.dbInput.Blur()
			m.dbInput.Reset()
			m.state = defaultState
			return tea.Batch(cmds...)
		case tea.KeyEnter:
			dbStr := m.dbInput.Value()

			m.dbInput.Blur()
			m.dbInput.Reset()
			m.state = defaultState

			// Parse the database number
			db := cast.ToInt(dbStr)
			if dbStr == "" || db < 0 {
				m.statusMessage = "Invalid database number"
				return tea.Batch(cmds...)
			}

			m.ready = false
			cmds = append(cmds, m.switchDBCmd(db))
			return tea.Batch(cmds...)
		}
	}

	m.dbInput, cmd = m.dbInput.Update(msg)
	cmds = append(cmds, cmd)

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

func (m *model) handleStatsState(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "i", "esc", "q":
			// Close stats page
			m.state = defaultState
		case "r":
			// Reload stats
			m.statsData = &statsData{loading: true}
			cmds = append(cmds, m.statsCmd())
		}
	}

	return tea.Batch(cmds...)
}
