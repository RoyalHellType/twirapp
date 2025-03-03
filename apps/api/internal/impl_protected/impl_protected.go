package impl_protected

import (
	"github.com/alexedwards/scs/v2"
	"github.com/redis/go-redis/v9"
	"github.com/satont/twir/apps/api/internal/impl_deps"
	"github.com/satont/twir/apps/api/internal/impl_protected/bot"
	"github.com/satont/twir/apps/api/internal/impl_protected/build_in_variables"
	"github.com/satont/twir/apps/api/internal/impl_protected/community"
	"github.com/satont/twir/apps/api/internal/impl_protected/dashboard"
	"github.com/satont/twir/apps/api/internal/impl_protected/events"
	"github.com/satont/twir/apps/api/internal/impl_protected/feedback"
	"github.com/satont/twir/apps/api/internal/impl_protected/files"
	"github.com/satont/twir/apps/api/internal/impl_protected/integrations"
	"github.com/satont/twir/apps/api/internal/impl_protected/moderation"
	"github.com/satont/twir/apps/api/internal/impl_protected/modules"
	"github.com/satont/twir/apps/api/internal/impl_protected/overlays"
	"github.com/satont/twir/apps/api/internal/impl_protected/rewards"
	"github.com/satont/twir/apps/api/internal/impl_protected/twitch"
	"github.com/satont/twir/apps/api/internal/impl_protected/users"
	config "github.com/satont/twir/libs/config"
	"github.com/satont/twir/libs/logger"
	apimodules "github.com/satont/twir/libs/types/types/api/modules"
	buscore "github.com/twirapp/twir/libs/bus-core"
	generic_cacher "github.com/twirapp/twir/libs/cache/generic-cacher"
	"github.com/twirapp/twir/libs/grpc/discord"
	integrationsGrpc "github.com/twirapp/twir/libs/grpc/integrations"
	"github.com/twirapp/twir/libs/grpc/parser"
	"github.com/twirapp/twir/libs/grpc/tokens"
	"github.com/twirapp/twir/libs/grpc/websockets"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type Protected struct {
	*integrations.Integrations
	*modules.Modules
	*bot.Bot
	*community.Community
	*events.Events
	*rewards.Rewards
	*build_in_variables.BuildInVariables
	*dashboard.Dashboard
	*twitch.Twitch
	*files.Files
	*overlays.Overlays
	*moderation.Moderation
	*users.Users
	*feedback.Feedback
}

type Opts struct {
	fx.In

	Redis          *redis.Client
	DB             *gorm.DB
	Config         config.Config
	SessionManager *scs.SessionManager

	TokensGrpc        tokens.TokensClient
	IntegrationsGrpc  integrationsGrpc.IntegrationsClient
	ParserGrpc        parser.ParserClient
	WebsocketsGrpc    websockets.WebsocketClient
	DiscordGrpc       discord.DiscordClient
	Logger            logger.Logger
	Bus               *buscore.Bus
	TTSSettingsCacher *generic_cacher.GenericCacher[apimodules.TTSSettings]
}

func New(opts Opts) *Protected {
	d := &impl_deps.Deps{
		Redis:          opts.Redis,
		Db:             opts.DB,
		Config:         opts.Config,
		SessionManager: opts.SessionManager,
		Grpc: &impl_deps.Grpc{
			Tokens:       opts.TokensGrpc,
			Integrations: opts.IntegrationsGrpc,
			Parser:       opts.ParserGrpc,
			Websockets:   opts.WebsocketsGrpc,
			Discord:      opts.DiscordGrpc,
		},
		Logger:            opts.Logger,
		Bus:               opts.Bus,
		TTSSettingsCacher: opts.TTSSettingsCacher,
	}

	return &Protected{
		Integrations:     &integrations.Integrations{Deps: d},
		Modules:          &modules.Modules{Deps: d},
		Bot:              &bot.Bot{Deps: d},
		Community:        &community.Community{Deps: d},
		Events:           &events.Events{Deps: d},
		Rewards:          &rewards.Rewards{Deps: d},
		BuildInVariables: &build_in_variables.BuildInVariables{Deps: d},
		Dashboard:        &dashboard.Dashboard{Deps: d},
		Twitch:           &twitch.Twitch{Deps: d},
		Files:            files.New(d),
		Overlays:         &overlays.Overlays{Deps: d},
		Moderation:       &moderation.Moderation{Deps: d},
		Users:            &users.Users{Deps: d},
		Feedback:         &feedback.Feedback{Deps: d},
	}
}
