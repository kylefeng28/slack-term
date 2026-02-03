package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/OpenPeeDeeP/xdg"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"

	"github.com/erroneousboat/slack-term/context"
)

const VERSION = "master-bubbletea"

var (
	flgConfig string
	flgToken  string
	flgCookie string
	flgDebug  bool
)

func init() {
	configFile := xdg.New("slack-term", "").QueryConfig("config")
	flag.StringVar(&flgConfig, "config", configFile, "location of config file")
	flag.StringVar(&flgToken, "token", "", "the slack token")
	flag.StringVar(&flgCookie, "cookie", "", "the slack cookie")
	flag.BoolVar(&flgDebug, "debug", false, "turn on debugging")
	flag.Parse()
}

type model struct {
	ctx            *context.AppContext
	channels       list.Model
	messages       viewport.Model
	input          textinput.Model
	currentChannel string
	ready          bool
	width          int
	height         int
	err            error
}

func initialModel() (model, error) {
	usage := "slack-term with bubbletea"
	ctx, err := context.CreateAppContext(flgConfig, flgToken, flgCookie, flgDebug, VERSION, usage)
	if err != nil {
		return model{}, err
	}

	/*
	// Load config
	config, err := config.NewConfig(flgConfig)
	if err != nil {
		return model{}, err
	}

	// Override with command line flags if provided
	if flgToken != "" {
		config.SlackToken = flgToken
	}
	if flgCookie != "" {
		config.SlackCookie = flgCookie
	}

	// Create Service only (skip view creation)
	svc, err := service.NewSlackService(config)
	if err != nil {
		return model{}, err
	}

	// Create minimal context without termui
	ctx := &context.AppContext{
		Service: svc,
		Config:  config,
		Debug:   flgDebug,
	}
	*/

	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Focus()

	return model{
		ctx:   ctx,
		input: ti,
	}, nil
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		loadChannelsCmd(m.ctx),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			// Switch focus between channels and input
			if m.input.Focused() {
				m.input.Blur()
			} else {
				m.input.Focus()
			}
		case "enter":
			if m.input.Focused() && m.input.Value() != "" {
				// TODO: Send message
				m.input.SetValue("")
			} else if !m.input.Focused() {
				// Load messages for selected channel
				if item, ok := m.channels.SelectedItem().(channelItem); ok {
					m.currentChannel = item.id
					return m, loadMessagesCmd(m.ctx, item.id)
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.channels = list.New([]list.Item{}, list.NewDefaultDelegate(), msg.Width/3, msg.Height-3)
			m.channels.Title = "Channels"
			m.messages = viewport.New(msg.Width*2/3, msg.Height-3)
			m.ready = true
		} else {
			m.channels.SetSize(msg.Width/3, msg.Height-3)
			m.messages.Width = msg.Width*2/3
			m.messages.Height = msg.Height - 3
		}

	case channelsLoadedMsg:
		items := make([]list.Item, len(msg.channels))
		for i, ch := range msg.channels {
			items[i] = channelItem{id: ch.ID, name: ch.Name}
		}
		m.channels.SetItems(items)

	case messagesLoadedMsg:
		m.messages.SetContent(msg.content)

	case errMsg:
		m.err = msg.err
	}

	var cmd tea.Cmd
	if m.input.Focused() {
		m.input, cmd = m.input.Update(msg)
	} else {
		m.channels, cmd = m.channels.Update(msg)
	}
	return m, cmd
}

func (m model) View() string {
	if !m.ready {
		return "Loading..."
	}

	channelsView := lipgloss.NewStyle().
		Width(m.width / 3).
		Height(m.height - 3).
		Render(m.channels.View())

	messagesView := lipgloss.NewStyle().
		Width(m.width*2/3).
		Height(m.height - 3).
		Render(m.messages.View())

	inputView := m.input.View()

	main := lipgloss.JoinHorizontal(lipgloss.Top, channelsView, messagesView)
	return lipgloss.JoinVertical(lipgloss.Left, main, inputView)
}

type channelItem struct {
	id   string
	name string
}

func (c channelItem) Title() string       { return c.name }
func (c channelItem) Description() string { return c.id }
func (c channelItem) FilterValue() string { return c.name }

type channelsLoadedMsg struct {
	channels []channelData
}

type messagesLoadedMsg struct {
	content string
}

type errMsg struct {
	err error
}

type channelData struct {
	ID   string
	Name string
}

func loadChannelsCmd(ctx *context.AppContext) tea.Cmd {
	return func() tea.Msg {
		channels := []channelData{}
		for _, ch := range ctx.Service.Conversations {
			channels = append(channels, channelData{
				ID:   ch.ID,
				Name: ch.Name,
			})
		}
		return channelsLoadedMsg{channels: channels}
	}
}

func loadMessagesCmd(ctx *context.AppContext, channelID string) tea.Cmd {
	return func() tea.Msg {
		messages, _, err := ctx.Service.GetMessages(channelID, 100)
		if err != nil {
			return errMsg{err: err}
		}

		content := ""
		for i := len(messages) - 1; i >= 0; i-- {
			msg := messages[i]
			content += fmt.Sprintf("%s %s: %s\n", 
				msg.Time.Format("15:04"), 
				msg.Name, 
				msg.Content)
		}

		return messagesLoadedMsg{content: content}
	}
}

func main() {
	m, err := initialModel()
	if err != nil {
		log.Fatal(err)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
