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

  PresenceAway   = "away"
  PresenceActive = "active"
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
	Type         string
	UserID       string
	Presence     string
	Notification bool
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

// ToString will set the label of the channel, how it will be
// displayed on screen. Based on the type, different icons are
// shown, as well as an optional notification icon.
func (c ChannelItem) ToString() string {
	var prefix string
	if c.Notification {
		prefix = IconNotification
	} else {
		prefix = " "
	}

	var icon string
	switch c.Type {
	case ChannelTypeChannel:
		icon = IconChannel
	case ChannelTypeGroup:
		icon = IconGroup
	case ChannelTypeMpIM:
		icon = IconMpIM
	case ChannelTypeIM:
		switch c.Presence {
		case PresenceActive:
			icon = IconOnline
		case PresenceAway:
			icon = IconOffline
		default:
			icon = IconIM
		}
	}

	label := fmt.Sprintf(
		"[%s](%s) [%s](%s) [%s](%s)",
		prefix, c.StylePrefix,
		icon, c.StyleIcon,
		c.Name, c.StyleText,
	)

	return label
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


