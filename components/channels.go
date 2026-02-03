package components

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	IconOnline       = "●"
	IconOffline      = "○"
	IconChannel      = "#"
	IconGroup        = "☰"
	IconIM           = "●"
	IconMpIM         = "☰"
	IconNotification = "*"
)

type Channels struct {
	List list.Model
}

const (
	ChannelTypeChannel = "channel"
	ChannelTypeGroup   = "group"
	ChannelTypeIM      = "im"
	ChannelTypeMpIM    = "mpim"
)

type ChannelItem struct {
	ID           string
	Name         string
	Topic        string
	UserID       string
	Presence     string
	Notification bool
	Type         string
	Unread       int
	StylePrefix  string
	StyleIcon    string
	StyleText    string
}

func (c ChannelItem) Title() string       { return c.Name }
func (c ChannelItem) Description() string { return c.ID }
func (c ChannelItem) FilterValue() string { return c.Name }

// Custom compact delegate
type channelDelegate struct{}

func (d channelDelegate) Height() int                               { return 1 }
func (d channelDelegate) Spacing() int                              { return 0 }
func (d channelDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d channelDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ch, ok := item.(ChannelItem)
	if !ok {
		return
	}

	// Get icon based on type
	icon := IconChannel
	switch ch.Type {
	case ChannelTypeGroup:
		icon = IconGroup
	case ChannelTypeIM:
		if ch.Presence == "active" {
			icon = IconOnline
		} else {
			icon = IconOffline
		}
	case ChannelTypeMpIM:
		icon = IconMpIM
	}

	// Notification indicator
	notification := " "
	if ch.Notification {
		notification = IconNotification
	}

	// Style based on selection
	var style lipgloss.Style
	if index == m.Index() {
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true)
	} else {
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	}

	// Format: [*] # channel-name
	line := fmt.Sprintf("[%s] %s %s", notification, icon, ch.Name)
	fmt.Fprint(w, style.Render(line))
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


