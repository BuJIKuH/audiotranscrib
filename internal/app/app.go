package app

import (
	"go.uber.org/fx"

	"audiotranscrib/internal/config"
	"audiotranscrib/internal/logger"
	"audiotranscrib/internal/telegram"
)

var Module = fx.Options(

	fx.Provide(
		config.InitConfig,
		logger.InitLogger,
		telegram.NewBot,
	),

	fx.Invoke(
		telegram.StartBot,
	),
)
