package context

import (
	"github.com/erroneousboat/slack-term/config"
	"github.com/erroneousboat/slack-term/service"
)

type AppContext struct {
	Service *service.SlackService
	Config  *config.Config
	Debug   bool
}
