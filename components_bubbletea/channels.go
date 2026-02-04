package components

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/lipgloss"

	components "github.com/erroneousboat/slack-term/components"
)

type Channels struct {
	List list.Model
}

// Type alias from original components
type ChannelItem = components.ChannelItem

// Custom compact delegate
type channelDelegate struct{}

func (d channelDelegate) Height() int                               { return 1 }
func (d channelDelegate) Spacing() int                              { return 0 }
func (d channelDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d channelDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	c, ok := item.(ChannelItem)
	if !ok {
		return
	}

	channelName := c.ToString()

	// Style based on selection
	var style lipgloss.Style
	if index == m.Index() {
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true)
	} else {
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	}

	fmt.Fprint(w, style.Render(channelName))
}

func NewChannels(width, height int) *Channels {
	l := list.New([]list.Item{}, channelDelegate{}, width, height)
	l.Title = "Channels"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	return &Channels{List: l}
}

func (c *Channels) SetChannels(channels []ChannelItem) {
	items := make([]list.Item, len(channels))
	for i, ch := range channels {
		items[i] = ch
	}
	c.List.SetItems(items)
}

func (c *Channels) Update(msg tea.Msg) (*Channels, tea.Cmd) {
	var cmd tea.Cmd
	c.List, cmd = c.List.Update(msg)
	return c, cmd
}

func (c *Channels) View() string {
	return c.List.View()
}

func (c *Channels) SetSize(width, height int) {
	c.List.SetSize(width, height)
}

func (c *Channels) SelectedChannel() *ChannelItem {
	if item, ok := c.List.SelectedItem().(ChannelItem); ok {
		return &item
	}
	return nil
}


