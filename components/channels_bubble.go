package components

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
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

func NewChannels(width, height int) *Channels {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), width, height)
	l.Title = "Channels"
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

