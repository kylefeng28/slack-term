package components

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Chat struct {
	Viewport viewport.Model
}

func NewChat(width, height int) *Chat {
	vp := viewport.New(width, height)
	return &Chat{Viewport: vp}
}

func (c *Chat) SetMessages(content string) {
	c.Viewport.SetContent(content)
	c.Viewport.GotoBottom()
}

func (c *Chat) Update(msg tea.Msg) (*Chat, tea.Cmd) {
	var cmd tea.Cmd
	c.Viewport, cmd = c.Viewport.Update(msg)
	return c, cmd
}

func (c *Chat) View() string {
	return c.Viewport.View()
}

func (c *Chat) SetSize(width, height int) {
	c.Viewport.Width = width
	c.Viewport.Height = height
}
