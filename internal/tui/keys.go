package tui

import keybind "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up       keybind.Binding
	Down     keybind.Binding
	Top      keybind.Binding
	Bottom   keybind.Binding
	Filter   keybind.Binding
	Spawn    keybind.Binding
	Refresh  keybind.Binding
	Mgmt     keybind.Binding
	Settings keybind.Binding
	Help     keybind.Binding
	Quit     keybind.Binding
}

var keys = keyMap{
	Up:       keybind.NewBinding(keybind.WithKeys("up", "k"), keybind.WithHelp("↑/k", "up")),
	Down:     keybind.NewBinding(keybind.WithKeys("down", "j"), keybind.WithHelp("↓/j", "down")),
	Top:      keybind.NewBinding(keybind.WithKeys("g"), keybind.WithHelp("g", "top")),
	Bottom:   keybind.NewBinding(keybind.WithKeys("G"), keybind.WithHelp("G", "bottom")),
	Filter:   keybind.NewBinding(keybind.WithKeys("/"), keybind.WithHelp("/", "filter")),
	Spawn:    keybind.NewBinding(keybind.WithKeys("enter"), keybind.WithHelp("⏎", "shell")),
	Refresh:  keybind.NewBinding(keybind.WithKeys("r"), keybind.WithHelp("r", "refresh")),
	Mgmt:     keybind.NewBinding(keybind.WithKeys("m"), keybind.WithHelp("m", "mgmt")),
	Settings: keybind.NewBinding(keybind.WithKeys("s"), keybind.WithHelp("s", "settings")),
	Help:     keybind.NewBinding(keybind.WithKeys("?"), keybind.WithHelp("?", "help")),
	Quit:     keybind.NewBinding(keybind.WithKeys("q", "ctrl+c"), keybind.WithHelp("q", "quit")),
}
