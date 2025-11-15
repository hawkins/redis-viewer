// @description: 键盘映射
// @file: keymap.go
// @date: 2022/02/07

package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap defines the keybindings for the app.
type keyMap struct {
	reload      key.Binding
	search      key.Binding
	fuzzySearch key.Binding
	delete      key.Binding
	switchDB    key.Binding
	toggleWrap  key.Binding
	help        key.Binding
}

// defaultKeyMap returns a set of default keybindings.
func defaultKeyMap() keyMap {
	return keyMap{
		reload: key.NewBinding(
			key.WithKeys("r"),
		),
		search: key.NewBinding(
			key.WithKeys("s"),
		),
		fuzzySearch: key.NewBinding(
			key.WithKeys("/"),
		),
		delete: key.NewBinding(
			key.WithKeys("x"),
		),
		switchDB: key.NewBinding(
			key.WithKeys("d"),
		),
		toggleWrap: key.NewBinding(
			key.WithKeys("w"),
		),
		help: key.NewBinding(
			key.WithKeys("?"),
		),
	}
}
