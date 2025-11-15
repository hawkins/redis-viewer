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
	focusIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F87")).
			Bold(true)

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
	style := listViewStyle
	if m.focused == listPane {
		style = style.BorderForeground(lipgloss.Color("#FF5F87"))
	} else {
		style = style.BorderForeground(lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"})
	}
	return style.Render(m.list.View())
}

func (m model) viewportContent() string {
	if it := m.list.SelectedItem(); it != nil {
		keyType := fmt.Sprintf("KeyType: %s", it.(item).keyType)
		width := m.viewport.Width
		wrappedKey := wordwrap.String(it.(item).key, width)
		key := fmt.Sprintf("Key: \n%s", wrappedKey)
		divider := dividerStyle.Render(strings.Repeat("-", width))

		formattedValue := util.TryPrettyJSON(it.(item).val)
		// Apply word wrapping if enabled
		if m.wordWrap {
			formattedValue = wordwrap.String(formattedValue, width)
		}
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
	content := m.viewport.View()

	// Add a focus indicator at the top of the viewport when focused
	if m.focused == viewportPane {
		indicator := focusIndicator.Render("▸ ")
		content = indicator + content
	}

	return content
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
		"  w         Toggle word wrap",
		"  i         View server statistics",
		"  x         Delete selected key",
		"  P         Purge database (delete all keys)",
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
	var statusKey, encoding, wrapIndicator, datetime string

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
		if m.wordWrap {
			wrapIndicator = statusNugget.Copy().Background(lipgloss.Color("#50FA7B")).Render("WRAP")
		}
		datetime = datetimeStyle.Render(m.now)

		// Calculate available width for the confirmation message
		fixedWidth := lipgloss.Width(statusKey) + lipgloss.Width(encoding) + lipgloss.Width(wrapIndicator) + lipgloss.Width(datetime)
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
	case confirmPurgeState:
		status = "DANGER"
		statusKey = statusStyle.Copy().Background(lipgloss.Color("#FF0000")).Render(status)
		encoding = encodingStyle.Render("UTF-8")
		if m.wordWrap {
			wrapIndicator = statusNugget.Copy().Background(lipgloss.Color("#50FA7B")).Render("WRAP")
		}
		datetime = datetimeStyle.Render(m.now)
		statusDesc = fmt.Sprintf("PURGE ALL KEYS in database %d? (y/n)", m.db)
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
	if wrapIndicator == "" && m.wordWrap {
		wrapIndicator = statusNugget.Copy().Background(lipgloss.Color("#50FA7B")).Render("WRAP")
	}
	if datetime == "" {
		datetime = datetimeStyle.Render(m.now)
	}

	// Calculate available width for status description
	availableWidth := m.width - lipgloss.Width(statusKey) - lipgloss.Width(encoding) - lipgloss.Width(wrapIndicator) - lipgloss.Width(datetime)
	if availableWidth < 0 {
		availableWidth = 0
	}

	statusVal := statusText.Copy().
		Width(availableWidth).
		Render(statusDesc)

	bar := lipgloss.JoinHorizontal(lipgloss.Top, statusKey, statusVal, encoding, wrapIndicator, datetime)

	return statusBarStyle.Width(m.width).Render(bar)
}

func (m model) statsView() string {
	if m.statsData == nil || m.statsData.loading {
		// Show loading state
		loadingMsg := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F87")).
			Bold(true).
			Render(m.spinner.View() + " Loading statistics...")

		return lipgloss.Place(
			m.width,
			m.height-lipgloss.Height(m.statusView()),
			lipgloss.Center,
			lipgloss.Center,
			loadingMsg,
		)
	}

	if m.statsData.err != nil {
		// Show error state
		errorMsg := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true).
			Render(fmt.Sprintf("Error loading stats: %v", m.statsData.err))

		return lipgloss.Place(
			m.width,
			m.height-lipgloss.Height(m.statusView()),
			lipgloss.Center,
			lipgloss.Center,
			errorMsg,
		)
	}

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF5F87")).
		MarginBottom(1)

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#A550DF")).
		MarginTop(1).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"}).
		Width(25)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#343433", Dark: "#C1C6B2"}).
		Bold(true)

	var sections []string

	// Title
	sections = append(sections, titleStyle.Render("Redis Server Statistics"))

	// Server Info Section
	if m.statsData.serverStats != nil {
		s := m.statsData.serverStats
		sections = append(sections, sectionStyle.Render("Server Information"))

		serverInfo := []string{
			labelStyle.Render("Redis Version:") + valueStyle.Render(s.Version),
			labelStyle.Render("Uptime:") + valueStyle.Render(formatUptime(s.UptimeSeconds)),
			labelStyle.Render("Connected Clients:") + valueStyle.Render(fmt.Sprintf("%d", s.ConnectedClients)),
			labelStyle.Render("Ops/sec:") + valueStyle.Render(fmt.Sprintf("%d", s.OpsPerSec)),
			labelStyle.Render("Total Commands:") + valueStyle.Render(formatNumber(s.TotalCommandsProcessed)),
		}
		sections = append(sections, lipgloss.JoinVertical(lipgloss.Left, serverInfo...))

		// Memory Section
		sections = append(sections, sectionStyle.Render("Memory Statistics"))

		usedMem := s.UsedMemory
		if usedMem == "" {
			usedMem = "N/A"
		}
		peakMem := s.UsedMemoryPeak
		if peakMem == "" {
			peakMem = "N/A"
		}

		memoryInfo := []string{
			labelStyle.Render("Used Memory:") + valueStyle.Render(usedMem),
			labelStyle.Render("Peak Memory:") + valueStyle.Render(peakMem),
			labelStyle.Render("Fragmentation Ratio:") + valueStyle.Render(fmt.Sprintf("%.2f", s.MemFragmentationRatio)),
			labelStyle.Render("Evicted Keys:") + valueStyle.Render(formatNumber(s.EvictedKeys)),
			labelStyle.Render("Expired Keys:") + valueStyle.Render(formatNumber(s.ExpiredKeys)),
		}
		sections = append(sections, lipgloss.JoinVertical(lipgloss.Left, memoryInfo...))
	}

	// Database Section
	if len(m.statsData.dbStats) > 0 {
		sections = append(sections, sectionStyle.Render("Database Statistics"))

		// Table header
		headerStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#6124DF")).
			Width(15)

		tableHeader := lipgloss.JoinHorizontal(
			lipgloss.Left,
			headerStyle.Copy().Width(10).Render("Database"),
			headerStyle.Copy().Width(15).Render("Keys"),
			headerStyle.Copy().Width(20).Render("Avg TTL"),
		)
		sections = append(sections, tableHeader)

		// Table rows
		rowStyle := lipgloss.NewStyle().Width(15)
		for _, db := range m.statsData.dbStats {
			avgTTL := db.AvgTTL
			if avgTTL == "" {
				avgTTL = "No TTL"
			}

			row := lipgloss.JoinHorizontal(
				lipgloss.Left,
				rowStyle.Copy().Width(10).Render(fmt.Sprintf("DB %d", db.DB)),
				rowStyle.Copy().Width(15).Render(formatNumber(db.Keys)),
				rowStyle.Copy().Width(20).Render(avgTTL),
			)
			sections = append(sections, row)
		}
	}

	// Footer
	sections = append(sections, "")
	sections = append(sections, lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"}).
		Render("Press 'i', 'q', or ESC to close | Press 'r' to reload"))

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Center the content
	return lipgloss.Place(
		m.width,
		m.height-lipgloss.Height(m.statusView()),
		lipgloss.Center,
		lipgloss.Center,
		lipgloss.NewStyle().
			Padding(2, 4).
			Render(content),
	)
}

func formatUptime(seconds int64) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

func formatNumber(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	// Add commas for thousands separator
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

func (m model) View() string {
	// TODO: refresh status view only
	var content string

	// Show stats page, help dialog, or main content
	if m.state == statsState {
		content = m.statsView()
	} else if m.state == helpState {
		content = m.helpView()
	} else {
		content = lipgloss.JoinHorizontal(lipgloss.Top, m.listView(), m.detailView())
	}

	return lipgloss.JoinVertical(lipgloss.Left, content, m.statusView())
}
