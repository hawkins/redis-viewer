// @file: view.go
// @date: 2022/02/08

package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
	"github.com/saltfishpr/redis-viewer/internal/util"
)

var (
	listViewStyle = lipgloss.NewStyle().
			PaddingRight(1).
			MarginRight(1).
			Border(lipgloss.RoundedBorder(), false, true, false, false)
	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"})

	statusNugget   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFDF5")).Padding(0, 1)
	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#343433", Dark: "#C1C6B2"}).
			Background(lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#353533"})
	statusStyle = statusBarStyle.Copy().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#FF5F87")).
			Padding(0, 1).
			MarginRight(1)
	encodingStyle = statusNugget.Copy().Background(lipgloss.Color("#A550DF")).Align(lipgloss.Right)
	statusText    = statusBarStyle.Copy()
	datetimeStyle = statusNugget.Copy().Background(lipgloss.Color("#6124DF"))
)

func (m model) listView() string {
	return listViewStyle.Render(m.list.View())
}

func (m model) viewportContent() string {
	if it := m.list.SelectedItem(); it != nil {
		keyType := fmt.Sprintf("KeyType: %s", it.(item).keyType)
		width := m.viewport.Width
		wrappedKey := wordwrap.String(it.(item).key, width)
		key := fmt.Sprintf("Key: \n%s", wrappedKey)
		divider := dividerStyle.Render(strings.Repeat("-", width))

		formattedValue := util.TryPrettyJSON(it.(item).val)
		value := fmt.Sprintf("Value: \n%s", formattedValue)

		content := []string{keyType}
		if it.(item).expiration != "" {
			content = append(content, fmt.Sprintf("TTL: %s", it.(item).expiration))
		}

		content = append(content, divider, key, divider, value)

		finalContent := lipgloss.JoinVertical(lipgloss.Left, content...)

		return finalContent
	}

	return "No item selected"
}

func (m model) detailView() string {
	return m.viewport.View()
}

func (m model) helpView() string {
	helpTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF5F87")).
		Render("Keybindings")

	helpItems := []string{
		"",
		helpTitle,
		"",
		"  ↑/↓       Navigate keys",
		"  ←/→       Navigate panes",
		"  r         Reload keys",
		"  s         Search for keys",
		"  /         Fuzzy filter keys",
		"  Ctrl+F    Toggle fuzzy/strict mode",
		"  d         Switch database",
		"  x         Delete selected key",
		"  ?         Toggle this help",
		"  Ctrl+C    Quit application",
		"",
		"Press ? or ESC to close",
	}

	content := strings.Join(helpItems, "\n")

	// Calculate the height for the help content area
	// (same height as the main content area to prevent scrolling)
	statusBarHeight := lipgloss.Height(m.statusView())
	availableHeight := m.height - statusBarHeight

	// Use lipgloss.Place to center the content
	return lipgloss.Place(
		m.width,
		availableHeight,
		lipgloss.Center,
		lipgloss.Center,
		lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#343433", Dark: "#C1C6B2"}).
			Background(lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#353533"}).
			Padding(1, 2).
			Render(content),
	)
}

func (m model) statusView() string {
	var status string
	var statusDesc string

	// Pre-render fixed elements to get their widths
	var statusKey, encoding, datetime string

	switch m.state {
	case searchState:
		status = "Search"
		statusDesc = m.textinput.View()
	case fuzzySearchState:
		if m.fuzzyStrict {
			status = "Strict"
		} else {
			status = "Fuzzy"
		}
		statusDesc = m.fuzzyInput.View()
	case switchDBState:
		status = "Switch DB"
		statusDesc = m.dbInput.View()
	case confirmDeleteState:
		status = "Confirm"
		statusKey = statusStyle.Render(status)
		encoding = encodingStyle.Render("UTF-8")
		datetime = datetimeStyle.Render(m.now)

		// Calculate available width for the confirmation message
		fixedWidth := lipgloss.Width(statusKey) + lipgloss.Width(encoding) + lipgloss.Width(datetime)
		availableWidth := m.width - fixedWidth

		// Account for the message template: "Delete key ''? (y/n)" = ~23 chars
		messageOverhead := len("Delete key ''? (y/n)")
		maxKeyLen := availableWidth - messageOverhead - 5 // Extra padding for safety

		if maxKeyLen < 10 {
			maxKeyLen = 10 // Minimum readable length
		}

		keyName := m.keyToDelete
		if len(keyName) > maxKeyLen {
			keyName = keyName[:maxKeyLen] + "..."
		}
		statusDesc = fmt.Sprintf("Delete key '%s'? (y/n)", keyName)
	default:
		status = "Ready"
		statusDesc = m.statusMessage
		if !m.ready {
			status = m.spinner.View()
			statusDesc = "Loading..."
		}
		// Show active fuzzy filter in status
		if m.fuzzyFilter != "" {
			var modeLabel string
			if m.fuzzyStrict {
				modeLabel = "Strict Filter"
			} else {
				modeLabel = "Fuzzy Filter"
			}
			if m.statusMessage != "" {
				statusDesc = fmt.Sprintf("[%s: %s] %s", modeLabel, m.fuzzyFilter, m.statusMessage)
			} else {
				statusDesc = fmt.Sprintf("[%s: %s]", modeLabel, m.fuzzyFilter)
			}
		}
	}

	// Render fixed elements if not already done
	if statusKey == "" {
		statusKey = statusStyle.Render(status)
	}
	if encoding == "" {
		encoding = encodingStyle.Render("UTF-8")
	}
	if datetime == "" {
		datetime = datetimeStyle.Render(m.now)
	}

	// Calculate available width for status description
	availableWidth := m.width - lipgloss.Width(statusKey) - lipgloss.Width(encoding) - lipgloss.Width(datetime)
	if availableWidth < 0 {
		availableWidth = 0
	}

	statusVal := statusText.Copy().
		Width(availableWidth).
		Render(statusDesc)

	bar := lipgloss.JoinHorizontal(lipgloss.Top, statusKey, statusVal, encoding, datetime)

	return statusBarStyle.Width(m.width).Render(bar)
}

func (m model) View() string {
	// TODO: refresh status view only
	var content string

	// Show help dialog or main content
	if m.state == helpState {
		content = m.helpView()
	} else {
		content = lipgloss.JoinHorizontal(lipgloss.Top, m.listView(), m.detailView())
	}

	return lipgloss.JoinVertical(lipgloss.Left, content, m.statusView())
}
