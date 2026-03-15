package telegram

import (
	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func registerHandlers(bot *tele.Bot, logger *zap.Logger) {

	bot.Handle("/start", func(c tele.Context) error {

		logger.Info(
			"command start",
			zap.Int64("user_id", c.Sender().ID),
			zap.String("username", c.Sender().Username),
		)

		return c.Send("Добро пожаловать")
	})

	bot.Handle("/list", func(c tele.Context) error {

		logger.Info(
			"command list",
			zap.Int64("user_id", c.Sender().ID),
		)

		return c.Send("Список встреч")
	})

	bot.Handle(tele.OnVoice, func(c tele.Context) error {

		logger.Info(
			"voice received",
			zap.Int64("user_id", c.Sender().ID),
		)

		return c.Send("Получено голосовое сообщение")
	})
}
