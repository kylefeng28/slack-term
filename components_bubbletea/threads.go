package components

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type Threads struct {
	List list.Model
}

func NewThreads(width, height int) *Threads {
	l := list.New([]list.Item{}, channelDelegate{}, width, height)
	l.Title = "Threads"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	return &Threads{List: l}
}

func (t *Threads) SetThreads(threads []ChannelItem) {
	items := make([]list.Item, len(threads))
	for i, th := range threads {
		items[i] = th
	}
	t.List.SetItems(items)
}

func (t *Threads) Update(msg tea.Msg) (*Threads, tea.Cmd) {
	var cmd tea.Cmd
	t.List, cmd = t.List.Update(msg)
	return t, cmd
}

func (t *Threads) View() string {
	return t.List.View()
}

func (t *Threads) SetSize(width, height int) {
	t.List.SetSize(width, height)
}

func (t *Threads) SelectedThread() *ChannelItem {
	if item, ok := t.List.SelectedItem().(ChannelItem); ok {
		return &item
	}
	return nil
}
