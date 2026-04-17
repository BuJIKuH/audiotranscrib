package app

import (
	"audiotranscrib/internal/ai"
	"audiotranscrib/internal/speech"
	"audiotranscrib/internal/storage"

	"go.uber.org/fx"

	"audiotranscrib/internal/config"
	"audiotranscrib/internal/logger"
	"audiotranscrib/internal/telegram"
)

var Module = fx.Options(

	fx.Provide(
		config.InitConfig,
		logger.InitLogger,

		storage.NewDBStorage,
		storage.NewUserRepo,
		storage.NewMeetingRepo,

		telegram.NewBot,
		speech.NewClient,
		ai.NewGigaChatClient,
	),

	fx.Invoke(
		telegram.StartBot,
	),
)
