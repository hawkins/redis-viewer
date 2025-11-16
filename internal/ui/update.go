package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	redisv8 "github.com/go-redis/redis/v8"
	"github.com/hawkins/redis-viewer/internal/constant"
	"github.com/hawkins/redis-viewer/internal/styles"
	"github.com/hawkins/redis-viewer/internal/ui/components/keylist"
	"github.com/hawkins/redis-viewer/internal/ui/dialogs"
	"github.com/spf13/cast"
)

// Update handles all messages and updates the model
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// Global message handling
	switch msg := msg.(type) {
	case StatsMsg:
		if msg.Err != nil {
			a.statusMessage = fmt.Sprintf("Failed to load stats: %v", msg.Err)
			a.statsData = &StatsData{loading: false, err: msg.Err}
		} else {
			a.statsData = &StatsData{
				serverStats: msg.ServerStats,
				dbStats:     msg.DBStats,
				loading:     false,
				err:         nil,
			}
		}
	case EditKeyMsg:
		if msg.Err != nil {
			a.state = StateDefault
			a.statusMessage = fmt.Sprintf("Failed to prepare edit: %v", msg.Err)
		} else {
			a.editingKey = msg.Key
			a.editingTmpFile = msg.TmpFile
			a.editingIsCreate = false
			cmds = append(cmds, a.openEditorCmd(msg.TmpFile))
		}
	case CreateKeyMsg:
		if msg.Err != nil {
			a.state = StateDefault
			a.statusMessage = fmt.Sprintf("Failed to prepare create: %v", msg.Err)
		} else {
			a.editingKey = msg.Key
			a.editingTmpFile = msg.TmpFile
			a.editingIsCreate = true
			cmds = append(cmds, a.openEditorCmd(msg.TmpFile))
		}
	case EditorFinishedMsg:
		if msg.Err != nil {
			a.state = StateDefault
			a.statusMessage = fmt.Sprintf("Editor failed: %v", msg.Err)
			_ = os.Remove(msg.TmpFile)
			a.editingKey = ""
			a.editingTmpFile = ""
		} else {
			if a.editingIsCreate {
				cmds = append(cmds, a.processCreatedKeyCmd(a.editingKey, a.editingTmpFile))
			} else {
				cmds = append(cmds, a.processEditedKeyCmd(a.editingKey, a.editingTmpFile))
			}
			a.editingKey = ""
			a.editingTmpFile = ""
		}
	case EditKeyResultMsg:
		a.state = StateDefault
		if msg.Err != nil {
			a.statusMessage = fmt.Sprintf("Failed to update key: %v", msg.Err)
		} else {
			a.statusMessage = fmt.Sprintf("Key '%s' updated successfully", msg.Key)
			a.ready = false
			a.scanInProgress = true
			a.scannedKeyCount = 0
			cmds = append(cmds, a.scanCmd(), a.countCmd())
		}
	case CreateKeyResultMsg:
		a.state = StateDefault
		if msg.Err != nil {
			a.statusMessage = fmt.Sprintf("Failed to create key: %v", msg.Err)
		} else {
			a.statusMessage = fmt.Sprintf("Key '%s' created successfully", msg.Key)
			a.ready = false
			a.scanInProgress = true
			a.scannedKeyCount = 0
			cmds = append(cmds, a.scanCmd(), a.countCmd())
		}
	case DeleteMsg:
		if msg.Err != nil {
			a.statusMessage = fmt.Sprintf("Failed to delete key: %v", msg.Err)
		} else {
			a.statusMessage = fmt.Sprintf("Key '%s' deleted successfully", msg.Key)
			a.ready = false
			a.scanInProgress = true
			a.scannedKeyCount = 0
			cmds = append(cmds, a.scanCmd(), a.countCmd())
		}
	case SetTTLMsg:
		if msg.Err != nil {
			a.statusMessage = fmt.Sprintf("Failed to set TTL: %v", msg.Err)
		} else {
			if msg.TTL <= 0 {
				a.statusMessage = fmt.Sprintf("TTL removed from key '%s' (now persistent)", msg.Key)
			} else {
				a.statusMessage = fmt.Sprintf("TTL set to %d seconds for key '%s'", msg.TTL, msg.Key)
			}
			a.ready = false
			a.scanInProgress = true
			a.scannedKeyCount = 0
			cmds = append(cmds, a.scanCmd(), a.countCmd())
		}
	case PurgeMsg:
		if msg.Err != nil {
			a.statusMessage = fmt.Sprintf("Failed to purge database: %v", msg.Err)
		} else {
			a.statusMessage = fmt.Sprintf("Database %d purged successfully", msg.DB)
			a.ready = false
			a.scanInProgress = true
			a.scannedKeyCount = 0
			cmds = append(cmds, a.scanCmd(), a.countCmd())
		}
	case SwitchDBMsg:
		a.state = StateDefault
		a.switchDBDialog.Blur()
		a.switchDBDialog.Reset()
		if msg.Err != nil {
			a.statusMessage = fmt.Sprintf("Failed to switch database: %v", msg.Err)
		} else {
			if a.rdb != nil {
				_ = a.rdb.Close()
			}
			a.rdb = msg.NewRdb.(redisv8.UniversalClient)
			a.db = msg.DB
			a.statusMessage = fmt.Sprintf("Switched to database %d", msg.DB)
			a.ready = false
			a.scanInProgress = true
			a.scannedKeyCount = 0
			cmds = append(cmds, a.scanCmd(), a.countCmd())
		}
	case ErrMsg:
		a.statusMessage = msg.Err.Error()
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		statusBarHeight := lipgloss.Height(a.statusView())
		height := a.height - statusBarHeight

		listViewWidth := cast.ToInt(constant.ListProportion * float64(a.width))
		listWidth := listViewWidth - styles.ListViewStyle.GetHorizontalFrameSize()
		a.keyList.SetSize(listWidth, height)

		detailViewWidth := a.width - listViewWidth
		a.valueView.SetSize(detailViewWidth, height)
		content := a.valueView.FormatContent(a.getCurrentItem())
		a.valueView.SetContent(content)
	case TickMsg:
		a.now = msg.T
		cmds = append(cmds, a.tickCmd())
	case LoadValueMsg:
		items := a.keyList.Items()
		for i, listItem := range items {
			if it, ok := listItem.(keylist.Item); ok && it.Key == msg.Key {
				items[i] = keylist.Item{
					KeyType:    it.KeyType,
					Key:        it.Key,
					Val:        msg.Val,
					Err:        msg.Err != nil,
					TTLSeconds: it.TTLSeconds,
					Loaded:     true,
				}
				a.keyList.SetItems(items)
				content := a.valueView.FormatContent(items[i].(keylist.Item))
				a.valueView.SetContent(content)
				break
			}
		}
	case ScanMsg:
		a.scannedKeyCount = msg.TotalScanned
		a.scanInProgress = !msg.IsComplete

		a.pendingScanItems = make([]keylist.Item, len(msg.Items))
		for i, item := range msg.Items {
			a.pendingScanItems[i] = item.(keylist.Item)
		}
		a.pendingScanIndex = 0
		a.keyList.SetItems(nil)
		a.valueView.GotoTop()
		a.valueView.SetContent("")
		cmds = append(cmds, a.displayBatchCmd())

		if msg.IsComplete {
			a.scanInProgress = false
		} else {
			a.statusMessage = fmt.Sprintf("Scanning... %d keys processed", msg.TotalScanned)
		}
	case ScanBatchMsg:
		currentItems := a.keyList.Items()
		newItems := make([]list.Item, len(currentItems)+len(msg.Batch))
		copy(newItems, currentItems)
		for i, item := range msg.Batch {
			newItems[len(currentItems)+i] = item
		}
		a.keyList.SetItems(newItems)
		a.pendingScanIndex += len(msg.Batch)

		if !msg.IsComplete {
			a.statusMessage = fmt.Sprintf("Displaying... %d/%d keys", a.pendingScanIndex, len(a.pendingScanItems))
		}

		if len(currentItems) == 0 && len(msg.Batch) > 0 {
			a.valueView.GotoTop()
			content := a.valueView.FormatContent(a.getCurrentItem())
			a.valueView.SetContent(content)

			if selectedItem := a.keyList.SelectedItem(); selectedItem != nil {
				if it, ok := selectedItem.(keylist.Item); ok && !it.Loaded {
					cmds = append(cmds, a.loadValueCmd(it.Key, it.KeyType, it.TTLSeconds))
				}
			}
		}

		if !msg.IsComplete {
			cmds = append(cmds, a.displayBatchCmd())
		} else {
			a.pendingScanItems = nil
			a.pendingScanIndex = 0
			if a.totalKeysToScan > 0 {
				a.statusMessage = fmt.Sprintf("DB %d: %d keys found", a.db, a.totalKeysToScan)
			}
		}
	case CountMsg:
		a.totalKeysToScan = msg.Count
		a.statusMessage = fmt.Sprintf("DB %d: %d keys found", a.db, msg.Count)
		a.ready = true
	}

	// State-specific handling
	switch a.state {
	case StateDefault:
		cmd = a.handleDefaultState(msg)
		cmds = append(cmds, cmd)
	case StateSearch:
		a.searchDialog, cmd = a.searchDialog.Update(msg)
		cmds = append(cmds, cmd)
	case StateFuzzySearch:
		a.filterDialog, cmd = a.filterDialog.Update(msg)
		cmds = append(cmds, cmd)
	case StateSwitchDB:
		// Handle escape to cancel
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.Type == tea.KeyEscape {
				a.state = StateDefault
				a.switchDBDialog.Blur()
				a.switchDBDialog.Reset()
				return a, nil
			}
		}
		a.switchDBDialog, cmd = a.switchDBDialog.Update(msg)
		cmds = append(cmds, cmd)
	case StateSetTTL:
		cmd = a.handleSetTTLState(msg)
		cmds = append(cmds, cmd)
	case StateCreateKeyInput:
		cmd = a.handleCreateKeyInputState(msg)
		cmds = append(cmds, cmd)
	case StateEditingKey:
		// Non-interactive state
	case StateConfirmDelete, StateConfirmPurge:
		a.confirmDialog, cmd = a.confirmDialog.Update(msg)
		cmds = append(cmds, cmd)
	case StateHelp:
		cmd = a.handleHelpState(msg)
		cmds = append(cmds, cmd)
	case StateStats:
		cmd = a.handleStatsState(msg)
		cmds = append(cmds, cmd)
	}

	a.spinner, cmd = a.spinner.Update(msg)
	cmds = append(cmds, cmd)

	return a, tea.Batch(cmds...)
}

func (a *App) handleDefaultState(msg tea.Msg) tea.Cmd {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.MouseMsg:
		a.valueView, cmd = a.valueView.Update(msg)
		cmds = append(cmds, cmd)
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyRunes:
			switch {
			case key.Matches(msg, a.keyMap.Search):
				a.state = StateSearch
				a.searchDialog.SetCallbacks(
					func(pattern string) tea.Cmd {
						a.searchValue = pattern
						a.state = StateDefault
						a.ready = false
						a.scanInProgress = true
						a.scannedKeyCount = 0
						return tea.Batch(a.scanCmd(), a.countCmd())
					},
					func() tea.Cmd {
						a.state = StateDefault
						if a.searchValue != "" {
							a.searchValue = ""
							a.ready = false
							a.scanInProgress = true
							a.scannedKeyCount = 0
							return tea.Batch(a.scanCmd(), a.countCmd())
						}
						return nil
					},
				)
				return a.searchDialog.Focus()
			case key.Matches(msg, a.keyMap.FuzzySearch):
				a.state = StateFuzzySearch
				a.filterDialog.SetValue(a.fuzzyFilter)
				a.filterDialog.SetCallbacks(
					func(pattern string) tea.Cmd {
						a.fuzzyFilter = pattern
						a.state = StateDefault
						a.ready = false
						a.scanInProgress = true
						a.scannedKeyCount = 0
						return tea.Batch(a.scanCmd(), a.countCmd())
					},
					func() tea.Cmd {
						a.state = StateDefault
						if a.fuzzyFilter != "" {
							a.fuzzyFilter = ""
							a.ready = false
							a.scanInProgress = true
							a.scannedKeyCount = 0
							return tea.Batch(a.scanCmd(), a.countCmd())
						}
						return nil
					},
					func() tea.Cmd {
						a.fuzzyStrict = !a.fuzzyStrict
						if a.fuzzyStrict {
							a.statusMessage = "Switched to strict mode"
						} else {
							a.statusMessage = "Switched to fuzzy mode"
						}
						if a.fuzzyFilter != "" {
							a.ready = false
							a.scanInProgress = true
							a.scannedKeyCount = 0
							return tea.Batch(a.scanCmd(), a.countCmd())
						}
						return nil
					},
				)
				return a.filterDialog.Focus()
			case key.Matches(msg, a.keyMap.SwitchDB):
				a.state = StateSwitchDB
				a.switchDBDialog.Reset()
				a.switchDBDialog.SetCallbacks(
					func(dbNum string) tea.Cmd {
						db := cast.ToInt(dbNum)
						if dbNum == "" || db < 0 {
							a.statusMessage = "Invalid database number"
							return nil
						}
						a.ready = false
						return a.switchDBCmd(db)
					},
					func() tea.Cmd {
						return nil
					},
				)
				return a.switchDBDialog.Focus()
			case key.Matches(msg, a.keyMap.SetTTL):
				if selectedItem := a.keyList.SelectedItem(); selectedItem != nil {
					if i, ok := selectedItem.(keylist.Item); ok {
						a.keyToSetTTL = i.Key
						a.state = StateSetTTL
						return a.ttlInput.Focus()
					}
				}
			case key.Matches(msg, a.keyMap.Reload):
				a.ready = false
				a.scanInProgress = true
				a.scannedKeyCount = 0
				return tea.Batch(a.scanCmd(), a.countCmd())
			case key.Matches(msg, a.keyMap.Delete):
				if selectedItem := a.keyList.SelectedItem(); selectedItem != nil {
					if i, ok := selectedItem.(keylist.Item); ok {
						a.keyToDelete = i.Key
						a.state = StateConfirmDelete
						a.confirmDialog = dialogs.NewConfirmDialog(dialogs.ConfirmDelete, i.Key)
						a.confirmDialog.SetCallbacks(
							func() tea.Cmd {
								a.state = StateDefault
								return a.deleteCmd(a.keyToDelete)
							},
							func() tea.Cmd {
								a.state = StateDefault
								a.keyToDelete = ""
								return nil
							},
						)
					}
				}
			case key.Matches(msg, a.keyMap.Purge):
				a.state = StateConfirmPurge
				a.confirmDialog = dialogs.NewConfirmDialog(dialogs.ConfirmPurge, a.db)
				a.confirmDialog.SetCallbacks(
					func() tea.Cmd {
						a.state = StateDefault
						return a.purgeCmd()
					},
					func() tea.Cmd {
						a.state = StateDefault
						return nil
					},
				)
			case key.Matches(msg, a.keyMap.ToggleWrap):
				a.valueView.ToggleWordWrap()
				if a.valueView.WordWrap() {
					a.statusMessage = "Word wrap enabled"
				} else {
					a.statusMessage = "Word wrap disabled"
				}
				content := a.valueView.FormatContent(a.getCurrentItem())
				a.valueView.SetContent(content)
			case key.Matches(msg, a.keyMap.Help):
				a.state = StateHelp
			case key.Matches(msg, a.keyMap.Stats):
				a.state = StateStats
				a.statsData = &StatsData{loading: true}
				return a.statsCmd()
			case key.Matches(msg, a.keyMap.Edit):
				if selectedItem := a.keyList.SelectedItem(); selectedItem != nil {
					if i, ok := selectedItem.(keylist.Item); ok {
						a.state = StateEditingKey
						a.statusMessage = fmt.Sprintf("Opening editor for key '%s'...", i.Key)
						return a.editKeyCmd(i.Key, i.Val)
					}
				}
			case key.Matches(msg, a.keyMap.Create):
				a.state = StateCreateKeyInput
				return a.createKeyInput.Focus()
			}
		case tea.KeyCtrlC:
			return tea.Quit
		case tea.KeyCtrlF:
			a.fuzzyStrict = !a.fuzzyStrict
			if a.fuzzyStrict {
				a.statusMessage = "Switched to strict mode"
			} else {
				a.statusMessage = "Switched to fuzzy mode"
			}
			if a.fuzzyFilter != "" {
				a.ready = false
				a.scanInProgress = true
				a.scannedKeyCount = 0
				return tea.Batch(a.scanCmd(), a.countCmd())
			}
		case tea.KeyLeft:
			a.focused = PaneList
			a.keyList.SetFocus(true)
			a.valueView.SetFocus(false)
		case tea.KeyRight:
			a.focused = PaneViewport
			a.keyList.SetFocus(false)
			a.valueView.SetFocus(true)
		case tea.KeyUp, tea.KeyDown:
			if a.focused == PaneList {
				a.keyList, cmd = a.keyList.Update(msg)
				cmds = append(cmds, cmd)

				if selectedItem := a.keyList.SelectedItem(); selectedItem != nil {
					if it, ok := selectedItem.(keylist.Item); ok && !it.Loaded {
						cmds = append(cmds, a.loadValueCmd(it.Key, it.KeyType, it.TTLSeconds))
					}
				}

				a.valueView.GotoTop()
				content := a.valueView.FormatContent(a.getCurrentItem())
				a.valueView.SetContent(content)
			} else {
				a.valueView, cmd = a.valueView.Update(msg)
				cmds = append(cmds, cmd)
			}
		}
	default:
		a.keyList, cmd = a.keyList.Update(msg)
		cmds = append(cmds, cmd)

		a.valueView, cmd = a.valueView.Update(msg)
		cmds = append(cmds, cmd)
	}

	return tea.Batch(cmds...)
}

func (a *App) handleSetTTLState(msg tea.Msg) tea.Cmd {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			a.ttlInput.Blur()
			a.ttlInput.Reset()
			a.state = StateDefault
			a.keyToSetTTL = ""
			return tea.Batch(cmds...)
		case tea.KeyEnter:
			ttlStr := a.ttlInput.Value()

			a.ttlInput.Blur()
			a.ttlInput.Reset()
			a.state = StateDefault

			ttl := cast.ToInt64(ttlStr)
			if ttlStr == "" {
				a.statusMessage = "TTL value cannot be empty. Use 0 to remove TTL."
				a.keyToSetTTL = ""
				return tea.Batch(cmds...)
			}

			if ttl < 0 {
				a.statusMessage = "TTL value must be 0 or positive"
				a.keyToSetTTL = ""
				return tea.Batch(cmds...)
			}

			a.ready = false
			cmds = append(cmds, a.setTTLCmd(a.keyToSetTTL, ttl))
			a.keyToSetTTL = ""
			return tea.Batch(cmds...)
		}
	}

	a.ttlInput, cmd = a.ttlInput.Update(msg)
	cmds = append(cmds, cmd)

	return tea.Batch(cmds...)
}

func (a *App) handleCreateKeyInputState(msg tea.Msg) tea.Cmd {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			a.createKeyInput.Blur()
			a.createKeyInput.Reset()
			a.state = StateDefault
			return tea.Batch(cmds...)
		case tea.KeyEnter:
			keyName := a.createKeyInput.Value()

			a.createKeyInput.Blur()
			a.createKeyInput.Reset()

			if keyName == "" {
				a.statusMessage = "Key name cannot be empty"
				a.state = StateDefault
				return tea.Batch(cmds...)
			}

			a.state = StateEditingKey
			a.statusMessage = fmt.Sprintf("Opening editor to create key '%s'...", keyName)
			cmds = append(cmds, a.createKeyCmd(keyName))
			return tea.Batch(cmds...)
		}
	}

	a.createKeyInput, cmd = a.createKeyInput.Update(msg)
	cmds = append(cmds, cmd)

	return tea.Batch(cmds...)
}

func (a *App) handleHelpState(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "?", "esc":
			a.state = StateDefault
		}
	}

	return tea.Batch(cmds...)
}

func (a *App) handleStatsState(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "i", "esc", "q":
			a.state = StateDefault
		case "r":
			a.statsData = &StatsData{loading: true}
			cmds = append(cmds, a.statsCmd())
		}
	}

	return tea.Batch(cmds...)
}

func (a App) getCurrentItem() keylist.Item {
	if selectedItem := a.keyList.SelectedItem(); selectedItem != nil {
		if it, ok := selectedItem.(keylist.Item); ok {
			return it
		}
	}
	return keylist.Item{}
}
