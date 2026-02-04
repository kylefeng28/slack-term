package views

import (
	"fmt"
	"github.com/erroneousboat/termui"

	"github.com/erroneousboat/slack-term/components"
	"github.com/erroneousboat/slack-term/config"
	"github.com/erroneousboat/slack-term/service"
)

type View struct {
	Config   *config.Config
	Input    *components.Input
	Chat     *components.Chat
	Channels *components.Channels
	Threads  *components.Threads
	Mode     *components.Mode
	Debug    *components.Debug
}

func CreateView(config *config.Config, svc *service.SlackService) (*View, error) {
	// Create Input component
	input := components.CreateInputComponent()

	// Channels: create the component
	sideBarHeight := termui.TermHeight() - input.Par.Height
	channels := components.CreateChannelsComponent(sideBarHeight)

	// Channels: fill the component
	var slackChans []components.ChannelItem
	var err error
	if config.IsEnterprise {
		slackChans, err = svc.GetConversationsForUser()
	} else {
		slackChans, err = svc.GetChannels(true)
	}

	if err != nil {
		return nil, err
	}

	// Channels: set channels in component
	channels.SetChannels(slackChans)

	if len(channels.ChannelItems) == 0 {
		return nil, fmt.Errorf("no channels available")
	}

	selectedChannel := channels.GetSelectedChannel()

	// Threads: create component
	threads := components.CreateThreadsComponent(sideBarHeight)

	// Chat: create the component
	chat := components.CreateChatComponent(input.Par.Height)

	// Chat: fill the component
	msgs, thr, err := svc.GetMessages(
		selectedChannel.ID,
		chat.GetMaxItems(),
		1,
	)
	if err != nil {
		return nil, err
	}

	// Chat: set messages in component
	chat.SetMessages(msgs)

	chat.SetBorderLabel(
		selectedChannel.GetChannelName(),
	)

	// Threads: set threads in component
	if len(thr) > 0 {

		// Make the first thread the current Channel
		threads.SetChannels(
			append(
				[]components.ChannelItem{channels.GetSelectedChannel()},
				thr...,
			),
		)
	}

	// Debug: create the component
	debug := components.CreateDebugComponent(input.Par.Height)

	// Mode: create the component
	mode := components.CreateModeComponent()

	view := &View{
		Config:   config,
		Input:    input,
		Channels: channels,
		Threads:  threads,
		Chat:     chat,
		Mode:     mode,
		Debug:    debug,
	}

	return view, nil
}

func (v *View) Refresh() {
	termui.Render(
		v.Input,
		v.Chat,
		v.Channels,
		v.Threads,
		v.Mode,
	)
}
