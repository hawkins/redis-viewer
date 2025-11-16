// @file: tui.go
// @date: 2022/02/07

// Package tui .
package tui

import (
	"context"
	"fmt"
	"github.com/muesli/termenv"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/go-redis/redis/v8"
	"github.com/saltfishpr/redis-viewer/internal/conf"
	"github.com/saltfishpr/redis-viewer/internal/constant"
	"github.com/saltfishpr/redis-viewer/internal/rv"
)

type state int

const (
	defaultState state = iota
	searchState
	fuzzySearchState
	switchDBState
	setTTLState
	confirmDeleteState
	confirmPurgeState
	helpState
	statsState
	createKeyInputState
	editingKeyState
)

type focusedPane int

const (
	listPane focusedPane = iota
	viewportPane
)

type statsData struct {
	serverStats *rv.ServerStats
	dbStats     []*rv.DatabaseStats
	loading     bool
	err         error
}

//nolint:govet
type model struct {
	width, height int

	list           list.Model
	textinput      textinput.Model
	fuzzyInput     textinput.Model
	dbInput        textinput.Model
	ttlInput       textinput.Model
	createKeyInput textinput.Model
	viewport       viewport.Model
	spinner        spinner.Model

	rdb             redis.UniversalClient
	redisOpts       *redis.UniversalOptions
	db              int
	searchValue     string
	fuzzyFilter     string
	fuzzyStrict     bool
	wordWrap        bool
	statusMessage   string
	ready           bool
	now             string
	keyToDelete     string
	keyToSetTTL     string
	statsData       *statsData
	editingKey      string
	editingTmpFile  string
	editingIsCreate bool

	offset int64
	limit  int64 // scan size

	keyMap
	state
	focused focusedPane
}

func New(config conf.Config) (*model, error) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	opts := &redis.UniversalOptions{
		Addrs:        config.Addrs,
		DB:           config.DB,
		Username:     config.Username,
		Password:     config.Password,
		MaxRetries:   constant.MaxRetries,
		MaxRedirects: constant.MaxRedirects,
		MasterName:   config.MasterName,
	}
	rdb := redis.NewUniversalClient(opts)
	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("connect to redis failed: %w", err)
	}

	t := textinput.New()
	t.Prompt = "> "
	t.Placeholder = "Search Key"
	t.PlaceholderStyle = lipgloss.NewStyle()

	f := textinput.New()
	f.Prompt = "> "
	f.Placeholder = "Fuzzy Filter"
	f.PlaceholderStyle = lipgloss.NewStyle()

	d := textinput.New()
	d.Prompt = "> "
	d.Placeholder = "Database Number"
	d.PlaceholderStyle = lipgloss.NewStyle()
	d.CharLimit = 4

	ttl := textinput.New()
	ttl.Prompt = "> "
	ttl.Placeholder = "TTL in seconds (or 0 to remove)"
	ttl.PlaceholderStyle = lipgloss.NewStyle()
	ttl.CharLimit = 10

	c := textinput.New()
	c.Prompt = "> "
	c.Placeholder = "Key Name"
	c.PlaceholderStyle = lipgloss.NewStyle()

	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Redis Viewer"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowFilter(false)
	l.SetFilteringEnabled(false)

	s := spinner.New()
	s.Spinner = spinner.Dot

	return &model{
		list:           l,
		textinput:      t,
		fuzzyInput:     f,
		dbInput:        d,
		ttlInput:       ttl,
		createKeyInput: c,
		spinner:        s,

		rdb:       rdb,
		redisOpts: opts,
		db:        config.DB,

		limit: config.Limit,

		keyMap:  defaultKeyMap(),
		state:   defaultState,
		focused: listPane,
	}, nil
}
