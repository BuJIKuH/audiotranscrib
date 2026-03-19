package telegram

import (
	"audiotranscrib/internal/speech"
	"audiotranscrib/internal/storage"
	"context"
	"io"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func registerHandlers(
	bot *tele.Bot,
	storage *storage.DBStorage,
	speechClient *speech.Client,
	logger *zap.Logger,
) {

	bot.Handle("/start", func(c tele.Context) error {

		ctx := context.Background()

		err := storage.CreateUser(
			ctx,
			c.Sender().ID,
			c.Sender().Username,
		)

		if err != nil {

			logger.Error(
				"cannot create user",
				zap.Error(err),
			)

			return c.Send("Ошибка регистрации")
		}

		logger.Info(
			"user registered",
			zap.Int64("user_id", c.Sender().ID),
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

	bot.Handle("/get", func(c tele.Context) error {
		logger.Info("command /get", zap.Int64("user_id", c.Sender().ID))
		return c.Send("Получение встречи")
	})

	bot.Handle("/find", func(c tele.Context) error {
		logger.Info("command /find", zap.Int64("user_id", c.Sender().ID))
		return c.Send("Поиск встречи")
	})

	bot.Handle("/chat", func(c tele.Context) error {
		logger.Info("command /chat", zap.Int64("user_id", c.Sender().ID))
		return c.Send("Чат с ИИ")
	})

	bot.Handle(tele.OnVoice, func(c tele.Context) error {

		file := c.Message().Voice.File

		reader, err := bot.File(&file)
		if err != nil {
			logger.Error("failed to get file", zap.Error(err))
			return c.Send("Ошибка загрузки файла")
		}
		defer reader.Close()

		data, err := io.ReadAll(reader)
		if err != nil {
			logger.Error("failed to read file", zap.Error(err))
			return c.Send("Ошибка чтения файла")
		}

		text, err := speechClient.Recognize(context.Background(), data)
		if err != nil {
			logger.Error("speech recognition failed", zap.Error(err))
			return c.Send("Ошибка распознавания")
		}

		return c.Send(text)
	})

	bot.Handle(tele.OnAudio, func(c tele.Context) error {
		logger.Info("audio message", zap.Int64("user_id", c.Sender().ID))
		return c.Send("Получен аудио файл")
	})
}
