package ui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/go-redis/redis/v8"
	"github.com/hawkins/redis-viewer/internal/config"
	"github.com/hawkins/redis-viewer/internal/constant"
	"github.com/hawkins/redis-viewer/internal/ui/components/keylist"
	"github.com/hawkins/redis-viewer/internal/ui/components/valueview"
	"github.com/hawkins/redis-viewer/internal/ui/dialogs"
	"github.com/muesli/termenv"
)

// AppState represents the current state of the application
type AppState int

const (
	StateDefault AppState = iota
	StateFuzzySearch
	StateSwitchDB
	StateSetTTL
	StateCreateKeyInput
	StateEditingKey
	StateConfirmDelete
	StateConfirmPurge
	StateHelp
	StateStats
)

// FocusedPane represents which pane has focus
type FocusedPane int

const (
	PaneList FocusedPane = iota
	PaneViewport
)

// App is the main application model
type App struct {
	width, height int

	// Components
	keyList   keylist.Model
	valueView valueview.Model
	spinner   spinner.Model

	// Dialogs
	filterDialog   dialogs.FilterDialog
	switchDBDialog dialogs.SwitchDBDialog
	ttlInput       textinput.Model
	createKeyInput textinput.Model
	confirmDialog  dialogs.ConfirmDialog

	// Redis connection
	rdb       redis.UniversalClient
	redisOpts *redis.UniversalOptions
	db        int

	// Application state
	state          AppState
	focused        FocusedPane
	fuzzyFilter    string
	fuzzyStrict    bool
	wordWrap       bool
	statusMessage  string
	ready          bool
	now            string
	keyToDelete    string
	keyToSetTTL    string
	editingKey     string
	editingTmpFile string
	editingIsCreate bool

	// Stats
	statsData *StatsData

	// Scan settings
	offset int64
	limit  int64

	// Scan state
	pendingScanItems []keylist.Item
	pendingScanIndex int
	scanInProgress   bool
	scannedKeyCount  int
	totalKeysToScan  int

	// Keybindings
	keyMap KeyMap
}

// StatsData holds statistics information
type StatsData struct {
	serverStats interface{}
	dbStats     interface{}
	loading     bool
	err         error
}

// New creates a new App instance
func New(cfg config.Config) (*App, error) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	opts := &redis.UniversalOptions{
		Addrs:        cfg.Addrs,
		DB:           cfg.DB,
		Username:     cfg.Username,
		Password:     cfg.Password,
		MaxRetries:   constant.MaxRetries,
		MaxRedirects: constant.MaxRedirects,
		MasterName:   cfg.MasterName,
	}
	rdb := redis.NewUniversalClient(opts)
	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("connect to redis failed: %w", err)
	}

	// Initialize components
	keyListModel := keylist.New(0, 0)
	valueViewModel := valueview.New(0, 0)

	s := spinner.New()
	s.Spinner = spinner.Dot

	// Initialize TTL input
	ttlInput := textinput.New()
	ttlInput.Prompt = "> "
	ttlInput.Placeholder = "TTL in seconds (or 0 to remove)"
	ttlInput.PlaceholderStyle = lipgloss.NewStyle()
	ttlInput.CharLimit = 10

	// Initialize create key input
	createKeyInput := textinput.New()
	createKeyInput.Prompt = "> "
	createKeyInput.Placeholder = "Key Name"
	createKeyInput.PlaceholderStyle = lipgloss.NewStyle()

	app := &App{
		keyList:         keyListModel,
		valueView:       valueViewModel,
		spinner:         s,
		filterDialog:    dialogs.NewFilterDialog(),
		switchDBDialog:  dialogs.NewSwitchDBDialog(),
		ttlInput:        ttlInput,
		createKeyInput:  createKeyInput,
		rdb:             rdb,
		redisOpts:       opts,
		db:              cfg.DB,
		limit:           cfg.Limit,
		keyMap:          DefaultKeyMap(),
		state:           StateDefault,
		focused:         PaneList,
	}

	// Set initial focus on the app's keyList component
	app.keyList.SetFocus(true)

	return app, nil
}

// Init initializes the Bubble Tea program
func (a App) Init() tea.Cmd {
	return tea.Batch(
		a.tickCmd(),
		a.spinner.Tick,
		a.scanCmd(),
		a.countCmd(),
	)
}
