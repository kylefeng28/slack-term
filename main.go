package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/OpenPeeDeeP/xdg"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/erroneousboat/slack-term/components"
	"github.com/erroneousboat/slack-term/config"
	"github.com/erroneousboat/slack-term/context"
	"github.com/erroneousboat/slack-term/service"
)

const VERSION = "bubbletea"

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
	ctx      *context.AppContext
	channels *components.Channels
	chat     *components.Chat
	input    *components.Input
	mode     *components.Mode
	ready    bool
	width    int
	height   int
	err      error
}

func initialModel() (model, error) {
	cfg, err := config.NewConfig(flgConfig)
	if err != nil {
		return model{}, err
	}

	if flgToken != "" {
		cfg.SlackToken = flgToken
	}
	if flgCookie != "" {
		cfg.SlackCookie = flgCookie
	}

	svc, err := service.NewSlackService(cfg)
	if err != nil {
		return model{}, err
	}

	ctx := &context.AppContext{
		Service: svc,
		Config:  cfg,
		Debug:   flgDebug,
	}

	return model{
		ctx:   ctx,
		input: components.NewInput(),
		mode:  components.NewMode(),
	}, nil
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		loadChannelsCmd(m.ctx),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.mode.Get() {
		case components.InsertMode:
			switch msg.String() {
			case "esc":
				m.mode.Set(components.CommandMode)
				m.input.Blur()
			case "enter":
				if m.input.Value() != "" {
					// TODO: Send message
					m.input.SetValue("")
				}
			default:
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				cmds = append(cmds, cmd)
			}

		case components.CommandMode:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "i":
				m.mode.Set(components.InsertMode)
				m.input.Focus()
			case "enter":
				if ch := m.channels.SelectedChannel(); ch != nil {
					return m, loadMessagesCmd(m.ctx, ch.ID)
				}
			case "j", "down":
				var cmd tea.Cmd
				m.channels, cmd = m.channels.Update(msg)
				cmds = append(cmds, cmd)
			case "k", "up":
				var cmd tea.Cmd
				m.channels, cmd = m.channels.Update(msg)
				cmds = append(cmds, cmd)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		
		sidebarWidth := m.width / 3
		chatWidth := m.width - sidebarWidth
		contentHeight := m.height - 2 // Reserve space for input and mode

		if !m.ready {
			m.channels = components.NewChannels(sidebarWidth, contentHeight)
			m.chat = components.NewChat(chatWidth, contentHeight)
			m.ready = true
		} else {
			m.channels.SetSize(sidebarWidth, contentHeight)
			m.chat.SetSize(chatWidth, contentHeight)
		}

	case channelsLoadedMsg:
		items := make([]components.ChannelItem, len(msg.channels))
		for i, ch := range msg.channels {
			items[i] = components.ChannelItem{
				ID:   ch.ID,
				Name: ch.Name,
			}
		}
		m.channels.SetChannels(items)

	case messagesLoadedMsg:
		m.chat.SetMessages(msg.content)

	case errMsg:
		m.err = msg.err
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Layout
	channelsView := lipgloss.NewStyle().
		Width(m.width / 3).
		Height(m.height - 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Render(m.channels.View())

	chatView := lipgloss.NewStyle().
		Width(m.width - m.width/3).
		Height(m.height - 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Render(m.chat.View())

	mainView := lipgloss.JoinHorizontal(lipgloss.Top, channelsView, chatView)

	statusBar := lipgloss.JoinHorizontal(
		lipgloss.Left,
		m.mode.View(),
		" | ",
		m.input.View(),
	)

	return lipgloss.JoinVertical(lipgloss.Left, mainView, statusBar)
}

// Messages
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

// Commands
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
