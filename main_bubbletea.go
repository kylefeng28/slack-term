package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/slack-go/slack"

	components "github.com/erroneousboat/slack-term/components_bubbletea"
	"github.com/erroneousboat/slack-term/config"
	"github.com/erroneousboat/slack-term/context"
	"github.com/erroneousboat/slack-term/service"
)

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
		listenRTMCmd(m.ctx),
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
			case "ctrl+f", "pgdown":
				m.channels.List.Paginator.NextPage()
			case "ctrl+b", "pgup":
				m.channels.List.Paginator.PrevPage()
			case "g":
				m.channels.List.Select(0)
			case "G":
				m.channels.List.Select(len(m.channels.List.Items()) - 1)
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

	case rtmEventMsg:
		// Handle RTM events and continue listening
		cmd := m.handleRTMEvent(msg.event)
		return m, tea.Batch(cmd, listenRTMCmd(m.ctx))

	case errMsg:
		m.err = msg.err
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7aa2f7")).
			Bold(true).
			Render("⏳ Loading Slack...")
	}

	// Color scheme
	borderColor := lipgloss.Color("#414868")
	
	// Build main view with channels, chat, and optional threads/debug
	views := []string{}
	
	// Channels (left sidebar)
	sidebarWidth := m.width / 3
	channelsView := lipgloss.NewStyle().
		Width(sidebarWidth).
		Height(m.height - 3).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Render(m.channels.View())
	views = append(views, channelsView)

	// Chat (main area)
	chatWidth := m.width - sidebarWidth - 4 // Account for borders and padding
	if m.showThreads {
		chatWidth -= m.width / 4
	}
	if m.showDebug {
		chatWidth -= 20
	}
	
	chatView := lipgloss.NewStyle().
		Width(chatWidth).
		Height(m.height - 3).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Render(m.chat.View())
	views = append(views, chatView)

	// Threads (optional)
	if m.showThreads && m.threads != nil {
		threadsView := lipgloss.NewStyle().
			Width(m.width / 4).
			Height(m.height - 3).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1).
			Render(m.threads.View())
		views = append(views, threadsView)
	}

	// Debug (optional)
	if m.showDebug && m.debug != nil {
		debugView := lipgloss.NewStyle().
			Width(20).
			Height(m.height - 3).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1).
			Render(m.debug.View())
		views = append(views, debugView)
	}

	mainView := lipgloss.JoinHorizontal(lipgloss.Top, views...)

	// Status bar with better styling
	statusBar := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#c0caf5")).
		Background(lipgloss.Color("#1a1b26")).
		Width(m.width).
		Padding(0, 1).
		Render(lipgloss.JoinHorizontal(
			lipgloss.Left,
			m.mode.View(),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89")).Render(" │ "),
			m.input.View(),
		))

	return lipgloss.JoinVertical(lipgloss.Left, mainView, statusBar)
}

// Messages
type channelsLoadedMsg struct {
	channels []components.ChannelItem
}

type messagesLoadedMsg struct {
	content string
}

type rtmEventMsg struct {
	event interface{}
}

type errMsg struct {
	err error
}

// Commands
func loadChannelsCmd(ctx *context.AppContext) tea.Cmd {
	return func() tea.Msg {
		debugPrintf("loadChannelsCmd: Starting to load channels")

		var channels []components.ChannelItem
		var err error
		if ctx.Config.IsEnterprise {
			channels, err = ctx.Service.GetConversationsForUser()
		} else {
			channels, err = ctx.Service.GetChannels(true)
		}

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
		
		messages, _, err := ctx.Service.GetMessages(channelID, 100, 3)
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

func listenRTMCmd(ctx *context.AppContext) tea.Cmd {
	return func() tea.Msg {
		debugPrintf("RTM: Waiting for event...")
		// Wait for next RTM event
		rtmEvent := <-ctx.Service.RTM.IncomingEvents
		debugPrintf("RTM: Received event type: %T", rtmEvent.Data)
		return rtmEventMsg{event: rtmEvent.Data}
	}
}

func (m *model) handleRTMEvent(event interface{}) tea.Cmd {
	switch ev := event.(type) {
	case *slack.MessageEvent:
		debugPrintf("RTM: Message event in channel %s from user %s", ev.Channel, ev.User)
		if m.debug != nil {
			m.debug.Println(fmt.Sprintf("New message in %s", ev.Channel))
		}
		
		// If it's for the current channel, reload messages
		if m.channels.SelectedChannel() != nil && ev.Channel == m.channels.SelectedChannel().ID {
			debugPrintf("RTM: Reloading messages for current channel")
			return loadMessagesCmd(m.ctx, ev.Channel)
		}
		
		// TODO: Mark channel as having unread messages
		
	case *slack.PresenceChangeEvent:
		debugPrintf("RTM: Presence change for user %s: %s", ev.User, ev.Presence)
		// TODO: Update user presence in channels list
		
	case *slack.RTMError:
		debugPrintf("RTM: Error: %v", ev.Error())
		if m.debug != nil {
			m.debug.Println(fmt.Sprintf("RTM Error: %v", ev.Error()))
		}
		
	case *slack.ConnectedEvent:
		debugPrintf("RTM: Connected to Slack RTM")
		if m.debug != nil {
			m.debug.Println("RTM: Connected")
		}
		
	case *slack.HelloEvent:
		debugPrintf("RTM: Received Hello")
		if m.debug != nil {
			m.debug.Println("RTM: Hello received")
		}
		
	default:
		debugPrintf("RTM: Unhandled event type: %T", event)
	}
	
	return nil
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
