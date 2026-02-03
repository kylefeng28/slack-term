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
	ctx            *context.AppContext
	channels       *components.Channels
	chat           *components.Chat
	input          *components.Input
	mode           *components.Mode
	ready          bool
	width          int
	height         int
	err            error
	pendingChannels []components.ChannelItem
	pendingMessages string
}

func debugPrintf(format string, args ...any) {
	if flgDebug {
		fmt.Printf(format, args...)
	}
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

	// Debug logging
	if m.ctx.Debug {
		switch msg.(type) {
		case tea.KeyMsg, tea.WindowSizeMsg:
			// Skip noisy messages
		default:
			debugPrintf("Update received: %T %+v", msg, msg)
		}
	}

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
			
			// Apply pending data
			if len(m.pendingChannels) > 0 {
				m.channels.SetChannels(m.pendingChannels)
				m.pendingChannels = nil
			}
			if m.pendingMessages != "" {
				m.chat.SetMessages(m.pendingMessages)
				m.pendingMessages = ""
			}
		} else {
			m.channels.SetSize(sidebarWidth, contentHeight)
			m.chat.SetSize(chatWidth, contentHeight)
		}

	case channelsLoadedMsg:
		debugPrintf("channelsLoadedMsg: Received %d channels, ready=%v", len(msg.channels), m.ready)
		if m.ready {
			m.channels.SetChannels(msg.channels)
			debugPrintf("channelsLoadedMsg: Set channels on component")
		} else {
			m.pendingChannels = msg.channels
			debugPrintf("channelsLoadedMsg: Buffered channels (not ready yet)")
		}

	case messagesLoadedMsg:
		debugPrintf("messagesLoadedMsg: Received messages, ready=%v, content length=%d", m.ready, len(msg.content))
		if m.ready {
			m.chat.SetMessages(msg.content)
			debugPrintf("messagesLoadedMsg: Set messages on component")
		} else {
			m.pendingMessages = msg.content
			debugPrintf("messagesLoadedMsg: Buffered messages (not ready yet)")
		}

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
	channels []components.ChannelItem
}

type messagesLoadedMsg struct {
	content string
}

type errMsg struct {
	err error
}

// Commands
func loadChannelsCmd(ctx *context.AppContext) tea.Cmd {
	return func() tea.Msg {
		debugPrintf("loadChannelsCmd: Starting to load channels")
		channels, err := ctx.Service.GetChannels()
		if err != nil {
			debugPrintf("loadChannelsCmd: Error: %v", err)
			return errMsg{err: err}
		}
		debugPrintf("loadChannelsCmd: Loaded %d channels", len(channels))
		return channelsLoadedMsg{channels: channels}
	}
}

func loadMessagesCmd(ctx *context.AppContext, channelID string) tea.Cmd {
	return func() tea.Msg {
		debugPrintf("loadMessagesCmd: Loading messages for channel %s", channelID)
		
		messages, _, err := ctx.Service.GetMessages(channelID, 100)
		if err != nil {
			debugPrintf("loadMessagesCmd: Error loading messages: %v", err)
			return errMsg{err: err}
		}

		debugPrintf("loadMessagesCmd: Loaded %d messages", len(messages))
		
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
	// Setup debug logging
	if flgDebug {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer f.Close()
	}

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
