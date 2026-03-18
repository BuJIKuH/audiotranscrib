package telegram

import (
	"audiotranscrib/internal/storage"
	"context"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"

	"audiotranscrib/internal/config"
)

func NewBot(cfg *config.Config) (*tele.Bot, error) {

	pref := tele.Settings{
		Token:  cfg.TelegramToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	return tele.NewBot(pref)
}

func StartBot(
	lc fx.Lifecycle,
	bot *tele.Bot,
	storage *storage.DBStorage,
	logger *zap.Logger,
) {

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {

			registerHandlers(bot, storage, logger)

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
