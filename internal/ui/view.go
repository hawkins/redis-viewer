package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hawkins/redis-viewer/internal/redis"
	"github.com/hawkins/redis-viewer/internal/styles"
)

// View renders the application
func (a App) View() string {
	var content string

	// Show stats page, help dialog, or main content
	if a.state == StateStats {
		content = a.statsView()
	} else if a.state == StateHelp {
		content = a.helpView()
	} else {
		content = lipgloss.JoinHorizontal(lipgloss.Top, a.keyList.View(), a.valueView.View())
	}

	return lipgloss.JoinVertical(lipgloss.Left, content, a.statusView())
}

func (a App) helpView() string {
	helpTitle := styles.HelpTitleStyle.Render("Keybindings")

	helpItems := []string{
		"",
		helpTitle,
		"",
		"  ↑/↓       Navigate keys",
		"  ←/→       Navigate panes",
		"  r         Reload keys",
		"  /         Fuzzy filter keys",
		"  Ctrl+F    Toggle fuzzy/strict mode",
		"  d         Switch database",
		"  t         Set TTL for selected key",
		"  w         Toggle word wrap",
		"  i         View server statistics",
		"  e         Edit selected key in $EDITOR",
		"  n         Create new key in $EDITOR",
		"  x         Delete selected key",
		"  P         Purge database (delete all keys)",
		"  ?         Toggle this help",
		"  Ctrl+C    Quit application",
		"",
		"Press ? or ESC to close",
	}

	content := strings.Join(helpItems, "\n")

	// Calculate the height for the help content area
	statusBarHeight := lipgloss.Height(a.statusView())
	availableHeight := a.height - statusBarHeight

	// Use lipgloss.Place to center the content
	return lipgloss.Place(
		a.width,
		availableHeight,
		lipgloss.Center,
		lipgloss.Center,
		styles.HelpDialogStyle.Render(content),
	)
}

func (a App) statusView() string {
	var status string
	var statusDesc string

	// Pre-render fixed elements to get their widths
	var statusKey, encoding, wrapIndicator, datetime string

	switch a.state {
	case StateFuzzySearch:
		if a.fuzzyStrict {
			status = "Strict"
		} else {
			status = "Fuzzy"
		}
		statusDesc = a.filterDialog.View()
	case StateSwitchDB:
		status = "Switch DB"
		statusDesc = a.switchDBDialog.View()
	case StateSetTTL:
		status = "Set TTL"
		statusDesc = a.ttlInput.View()
	case StateCreateKeyInput:
		status = "Create"
		statusDesc = a.createKeyInput.View()
	case StateEditingKey:
		status = "Editor"
		statusDesc = a.statusMessage
	case StateConfirmDelete:
		status = "Confirm"
		statusKey = styles.StatusStyle.Render(status)
		encoding = styles.EncodingStyle.Render("UTF-8")
		if a.valueView.WordWrap() {
			wrapIndicator = styles.WrapIndicatorStyle.Render("WRAP")
		}
		datetime = styles.DatetimeStyle.Render(a.now)

		// Calculate available width for the confirmation message
		fixedWidth := lipgloss.Width(statusKey) + lipgloss.Width(encoding) + lipgloss.Width(wrapIndicator) + lipgloss.Width(datetime)
		availableWidth := a.width - fixedWidth

		// Account for the message template
		messageOverhead := len("Delete key ''? (y/n)")
		maxKeyLen := availableWidth - messageOverhead - 5

		if maxKeyLen < 10 {
			maxKeyLen = 10
		}

		keyName := a.keyToDelete
		if len(keyName) > maxKeyLen {
			keyName = keyName[:maxKeyLen] + "..."
		}
		statusDesc = fmt.Sprintf("Delete key '%s'? (y/n)", keyName)
	case StateConfirmPurge:
		status = "DANGER"
		statusKey = styles.StatusDangerStyle.Render(status)
		encoding = styles.EncodingStyle.Render("UTF-8")
		if a.valueView.WordWrap() {
			wrapIndicator = styles.WrapIndicatorStyle.Render("WRAP")
		}
		datetime = styles.DatetimeStyle.Render(a.now)
		statusDesc = fmt.Sprintf("PURGE ALL KEYS in database %d? (y/n)", a.db)
	default:
		status = "Ready"
		statusDesc = a.statusMessage
		if !a.ready {
			status = a.spinner.View()
			statusDesc = "Loading..."
		}
		// Show active fuzzy filter in status
		if a.fuzzyFilter != "" {
			var modeLabel string
			if a.fuzzyStrict {
				modeLabel = "Strict Filter"
			} else {
				modeLabel = "Fuzzy Filter"
			}
			if a.statusMessage != "" {
				statusDesc = fmt.Sprintf("[%s: %s] %s", modeLabel, a.fuzzyFilter, a.statusMessage)
			} else {
				statusDesc = fmt.Sprintf("[%s: %s]", modeLabel, a.fuzzyFilter)
			}
		}
	}

	// Render fixed elements if not already done
	if statusKey == "" {
		statusKey = styles.StatusStyle.Render(status)
	}
	if encoding == "" {
		encoding = styles.EncodingStyle.Render("UTF-8")
	}
	if wrapIndicator == "" && a.valueView.WordWrap() {
		wrapIndicator = styles.WrapIndicatorStyle.Render("WRAP")
	}
	if datetime == "" {
		datetime = styles.DatetimeStyle.Render(a.now)
	}

	// Calculate available width for status description
	availableWidth := a.width - lipgloss.Width(statusKey) - lipgloss.Width(encoding) - lipgloss.Width(wrapIndicator) - lipgloss.Width(datetime)
	if availableWidth < 0 {
		availableWidth = 0
	}

	statusVal := styles.StatusText.Copy().
		Width(availableWidth).
		Render(statusDesc)

	bar := lipgloss.JoinHorizontal(lipgloss.Top, statusKey, statusVal, encoding, wrapIndicator, datetime)

	return styles.StatusBarStyle.Width(a.width).Render(bar)
}

func (a App) statsView() string {
	if a.statsData == nil || a.statsData.loading {
		// Show loading state
		loadingMsg := styles.StatsLoadingStyle.Render(a.spinner.View() + " Loading statistics...")

		return lipgloss.Place(
			a.width,
			a.height-lipgloss.Height(a.statusView()),
			lipgloss.Center,
			lipgloss.Center,
			loadingMsg,
		)
	}

	if a.statsData.err != nil {
		// Show error state
		errorMsg := styles.StatsErrorStyle.Render(fmt.Sprintf("Error loading stats: %v", a.statsData.err))

		return lipgloss.Place(
			a.width,
			a.height-lipgloss.Height(a.statusView()),
			lipgloss.Center,
			lipgloss.Center,
			errorMsg,
		)
	}

	var sections []string

	// Title
	sections = append(sections, styles.StatsTitleStyle.Render("Redis Server Statistics"))

	// Server Info Section
	if serverStats, ok := a.statsData.serverStats.(*redis.ServerStats); ok && serverStats != nil {
		sections = append(sections, styles.StatsSectionStyle.Render("Server Information"))

		serverInfo := []string{
			styles.StatsLabelStyle.Render("Redis Version:") + styles.StatsValueStyle.Render(serverStats.Version),
			styles.StatsLabelStyle.Render("Uptime:") + styles.StatsValueStyle.Render(formatUptime(serverStats.UptimeSeconds)),
			styles.StatsLabelStyle.Render("Connected Clients:") + styles.StatsValueStyle.Render(fmt.Sprintf("%d", serverStats.ConnectedClients)),
			styles.StatsLabelStyle.Render("Ops/sec:") + styles.StatsValueStyle.Render(fmt.Sprintf("%d", serverStats.OpsPerSec)),
			styles.StatsLabelStyle.Render("Total Commands:") + styles.StatsValueStyle.Render(formatNumber(serverStats.TotalCommandsProcessed)),
		}
		sections = append(sections, lipgloss.JoinVertical(lipgloss.Left, serverInfo...))

		// Memory Section
		sections = append(sections, styles.StatsSectionStyle.Render("Memory Statistics"))

		usedMem := serverStats.UsedMemory
		if usedMem == "" {
			usedMem = "N/A"
		}
		peakMem := serverStats.UsedMemoryPeak
		if peakMem == "" {
			peakMem = "N/A"
		}

		memoryInfo := []string{
			styles.StatsLabelStyle.Render("Used Memory:") + styles.StatsValueStyle.Render(usedMem),
			styles.StatsLabelStyle.Render("Peak Memory:") + styles.StatsValueStyle.Render(peakMem),
			styles.StatsLabelStyle.Render("Fragmentation Ratio:") + styles.StatsValueStyle.Render(fmt.Sprintf("%.2f", serverStats.MemFragmentationRatio)),
			styles.StatsLabelStyle.Render("Evicted Keys:") + styles.StatsValueStyle.Render(formatNumber(serverStats.EvictedKeys)),
			styles.StatsLabelStyle.Render("Expired Keys:") + styles.StatsValueStyle.Render(formatNumber(serverStats.ExpiredKeys)),
		}
		sections = append(sections, lipgloss.JoinVertical(lipgloss.Left, memoryInfo...))
	}

	// Database Section
	if dbStats, ok := a.statsData.dbStats.([]*redis.DatabaseStats); ok && len(dbStats) > 0 {
		sections = append(sections, styles.StatsSectionStyle.Render("Database Statistics"))

		// Table header
		tableHeader := lipgloss.JoinHorizontal(
			lipgloss.Left,
			styles.StatsHeaderStyle.Copy().Width(10).Render("Database"),
			styles.StatsHeaderStyle.Copy().Width(15).Render("Keys"),
			styles.StatsHeaderStyle.Copy().Width(20).Render("Avg TTL"),
		)
		sections = append(sections, tableHeader)

		// Table rows
		for _, db := range dbStats {
			avgTTL := db.AvgTTL
			if avgTTL == "" {
				avgTTL = "No TTL"
			}

			row := lipgloss.JoinHorizontal(
				lipgloss.Left,
				styles.StatsRowStyle.Copy().Width(10).Render(fmt.Sprintf("DB %d", db.DB)),
				styles.StatsRowStyle.Copy().Width(15).Render(formatNumber(db.Keys)),
				styles.StatsRowStyle.Copy().Width(20).Render(avgTTL),
			)
			sections = append(sections, row)
		}
	}

	// Footer
	sections = append(sections, "")
	sections = append(sections, styles.StatsFooterStyle.Render("Press 'i', 'q', or ESC to close | Press 'r' to reload"))

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Center the content
	return lipgloss.Place(
		a.width,
		a.height-lipgloss.Height(a.statusView()),
		lipgloss.Center,
		lipgloss.Center,
		styles.StatsContentStyle.Render(content),
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
