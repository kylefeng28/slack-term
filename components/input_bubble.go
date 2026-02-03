package components

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type Input struct {
	TextInput textinput.Model
}

func NewInput() *Input {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Focus()
	return &Input{TextInput: ti}
}

func (i *Input) SetValue(value string) {
	i.TextInput.SetValue(value)
}

func (i *Input) Value() string {
	return i.TextInput.Value()
}

func (i *Input) Focus() {
	i.TextInput.Focus()
}

func (i *Input) Blur() {
	i.TextInput.Blur()
}

func (i *Input) Focused() bool {
	return i.TextInput.Focused()
}

func (i *Input) Update(msg tea.Msg) (*Input, tea.Cmd) {
	var cmd tea.Cmd
	i.TextInput, cmd = i.TextInput.Update(msg)
	return i, cmd
}

func (i *Input) View() string {
	return i.TextInput.View()
}
