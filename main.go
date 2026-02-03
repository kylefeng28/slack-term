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
	ctx             *context.AppContext
	channels        *components.Channels
	chat            *components.Chat
	threads         *components.Threads
	debug           *components.Debug
	input           *components.Input
	mode            *components.Mode
	ready           bool
	width           int
	height          int
	err             error
	pendingChannels []components.ChannelItem
	pendingMessages string
	showThreads     bool
	showDebug       bool
}

func debugPrintf(format string, args ...any) {
	if flgDebug {
		log.Printf(format, args...)
	}
}

func (m *model) debugLog(format string, args ...any) {
	if m.debug != nil {
		m.debug.Println(fmt.Sprintf(format, args...))
	}
	debugPrintf(format, args...)
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
		ctx:       ctx,
		input:     components.NewInput(),
		mode:      components.NewMode(),
		showDebug: flgDebug,
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
			case "d":
				// Toggle debug pane and trigger resize
				m.showDebug = !m.showDebug
				return m, func() tea.Msg {
					return tea.WindowSizeMsg{Width: m.width, Height: m.height}
				}
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
		contentHeight := m.height - 2
		
		// Calculate widths based on what's shown
		chatWidth := m.width - sidebarWidth
		threadsWidth := 0
		debugWidth := 0
		
		if m.showThreads {
			threadsWidth = m.width / 4
			chatWidth -= threadsWidth
		}
		if m.showDebug {
			debugWidth = 20
			chatWidth -= debugWidth
		}

		if !m.ready {
			m.channels = components.NewChannels(sidebarWidth, contentHeight)
			m.chat = components.NewChat(chatWidth, contentHeight)
			if m.showThreads {
				m.threads = components.NewThreads(threadsWidth, contentHeight)
			}
			if m.showDebug {
				m.debug = components.NewDebug(debugWidth, contentHeight)
			}
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
			if m.threads != nil {
				m.threads.SetSize(threadsWidth, contentHeight)
			}
			if m.debug != nil {
				m.debug.SetSize(debugWidth, contentHeight)
			}
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

	// Build main view with channels, chat, and optional threads/debug
	views := []string{}
	
	// Channels (left sidebar)
	sidebarWidth := m.width / 3
	channelsView := lipgloss.NewStyle().
		Width(sidebarWidth).
		Height(m.height - 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Render(m.channels.View())
	views = append(views, channelsView)

	// Chat (main area)
	chatWidth := m.width - sidebarWidth
	if m.showThreads {
		chatWidth -= m.width / 4
	}
	if m.showDebug {
		chatWidth -= 20
	}
	
	chatView := lipgloss.NewStyle().
		Width(chatWidth).
		Height(m.height - 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Render(m.chat.View())
	views = append(views, chatView)

	// Threads (optional)
	if m.showThreads && m.threads != nil {
		threadsView := lipgloss.NewStyle().
			Width(m.width / 4).
			Height(m.height - 2).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Render(m.threads.View())
		views = append(views, threadsView)
	}

	// Debug (optional)
	if m.showDebug && m.debug != nil {
		debugView := lipgloss.NewStyle().
			Width(20).
			Height(m.height - 2).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Render(m.debug.View())
		views = append(views, debugView)
	}

	mainView := lipgloss.JoinHorizontal(lipgloss.Top, views...)

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
		for i := 0; i < len(messages); i++ {
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
