package ui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the keybindings for the app
type KeyMap struct {
	Reload      key.Binding
	Search      key.Binding
	FuzzySearch key.Binding
	Delete      key.Binding
	Purge       key.Binding
	SwitchDB    key.Binding
	SetTTL      key.Binding
	ToggleWrap  key.Binding
	Help        key.Binding
	Stats       key.Binding
	Edit        key.Binding
	Create      key.Binding
}

// DefaultKeyMap returns a set of default keybindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Reload: key.NewBinding(
			key.WithKeys("r"),
		),
		Search: key.NewBinding(
			key.WithKeys("s"),
		),
		FuzzySearch: key.NewBinding(
			key.WithKeys("/"),
		),
		Delete: key.NewBinding(
			key.WithKeys("x"),
		),
		Purge: key.NewBinding(
			key.WithKeys("P"),
		),
		SwitchDB: key.NewBinding(
			key.WithKeys("d"),
		),
		SetTTL: key.NewBinding(
			key.WithKeys("t"),
		),
		ToggleWrap: key.NewBinding(
			key.WithKeys("w"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
		),
		Stats: key.NewBinding(
			key.WithKeys("i"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
		),
		Create: key.NewBinding(
			key.WithKeys("n"),
		),
	}
}
