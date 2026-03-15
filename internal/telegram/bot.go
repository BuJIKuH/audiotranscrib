package telegram

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"

	"audiotranscrib/internal/config"
)

func NewBot(cfg *config.Config) (*tele.Bot, error) {

	pref := tele.Settings{
		Token: cfg.TelegramToken,
	}

	return tele.NewBot(pref)
}

func StartBot(
	lc fx.Lifecycle,
	bot *tele.Bot,
	logger *zap.Logger,
) {

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {

			registerHandlers(bot, logger)

			go bot.Start()

			logger.Info("telegram bot started")

			return nil
		},

		OnStop: func(ctx context.Context) error {

			logger.Info("telegram bot stopped")

			bot.Stop()

			return nil
		},
	})
}
