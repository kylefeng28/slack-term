package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Debug struct {
	Viewport viewport.Model
	lines    []string
}

func NewDebug(width, height int) *Debug {
	vp := viewport.New(width, height)
	return &Debug{
		Viewport: vp,
		lines:    []string{},
	}
}

func (d *Debug) Println(text string) {
	d.lines = append(d.lines, text)
	
	// Keep only last N lines to prevent memory growth
	maxLines := d.Viewport.Height * 2
	if len(d.lines) > maxLines {
		d.lines = d.lines[len(d.lines)-maxLines:]
	}
	
	d.Viewport.SetContent(strings.Join(d.lines, "\n"))
	d.Viewport.GotoBottom()
}

func (d *Debug) Update(msg tea.Msg) (*Debug, tea.Cmd) {
	var cmd tea.Cmd
	d.Viewport, cmd = d.Viewport.Update(msg)
	return d, cmd
}

func (d *Debug) View() string {
	return d.Viewport.View()
}

func (d *Debug) SetSize(width, height int) {
	d.Viewport.Width = width
	d.Viewport.Height = height
}
