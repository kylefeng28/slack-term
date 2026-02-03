package components

import (
	"github.com/charmbracelet/lipgloss"
)

type Mode struct {
	current string
}

const (
	CommandMode = "COMMAND"
	InsertMode  = "INSERT"
	SearchMode  = "SEARCH"
)

func NewMode() *Mode {
	return &Mode{current: CommandMode}
}

func (m *Mode) Set(mode string) {
	m.current = mode
}

func (m *Mode) Get() string {
	return m.current
}

func (m *Mode) View() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("cyan")).
		Bold(true)
	return style.Render(m.current)
}
