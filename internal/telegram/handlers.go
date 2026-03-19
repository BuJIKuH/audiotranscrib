package telegram

import (
	"audiotranscrib/internal/ai"
	"audiotranscrib/internal/speech"
	"audiotranscrib/internal/storage"
	"context"
	"database/sql"
	"io"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func registerHandlers(
	bot *tele.Bot,
	storage *storage.DBStorage,
	userRepo *storage.UserRepo,
	speechClient *speech.Client,
	gptClient *ai.GigaChatClient,
	logger *zap.Logger,
) {

	bot.Handle("/start", func(c tele.Context) error {
		ctx := context.Background()
		err := storage.CreateUser(ctx, c.Sender().ID, c.Sender().Username)
		if err != nil {
			logger.Error("cannot create user", zap.Error(err))
			return c.Send("Ошибка регистрации")
		}

		logger.Info("user registered", zap.Int64("user_id", c.Sender().ID))
		return c.Send("Добро пожаловать")
	})

	bot.Handle("/list", func(c tele.Context) error {
		logger.Info("command list", zap.Int64("user_id", c.Sender().ID))
		return c.Send("Список встреч")
	})

	bot.Handle("/get", func(c tele.Context) error {
		logger.Info("command get", zap.Int64("user_id", c.Sender().ID))
		return c.Send("Получение встречи")
	})

	bot.Handle("/find", func(c tele.Context) error {
		logger.Info("command find", zap.Int64("user_id", c.Sender().ID))
		return c.Send("Поиск встречи")
	})

	bot.Handle("/chat", func(c tele.Context) error {
		logger.Info("command chat", zap.Int64("user_id", c.Sender().ID))
		return c.Send("Чат с ИИ")
	})

	bot.Handle(tele.OnVoice, func(c tele.Context) error {
		ctx := context.Background()

		sender := c.Sender()
		user, err := userRepo.GetUserByTelegramID(ctx, sender.ID)
		if err != nil {
			if err == sql.ErrNoRows {
				user, err = userRepo.CreateUser(ctx, sender.ID, sender.Username)
				if err != nil {
					logger.Error("failed to create user", zap.Error(err))
					return c.Send("Ошибка создания пользователя")
				}
			} else {
				logger.Error("failed to get user", zap.Error(err))
				return c.Send("Ошибка доступа к пользователю")
			}
		}

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

		transcription, err := speechClient.Recognize(ctx, data)
		if err != nil {
			logger.Error("speech recognition failed", zap.Error(err))
			return c.Send("Ошибка распознавания")
		}

		var summary string
		if transcription != "" {
			logger.Info("sending transcription to GigaChat", zap.Int64("user_id", sender.ID))
			summary, err = gptClient.GetSummary(ctx, transcription)
			if err != nil {
				logger.Warn("failed to get summary", zap.Error(err))
				summary = ""
			}
		}

		_, err = storage.SaveMeeting(ctx, user.ID, file.FileID, transcription, summary)
		if err != nil {
			logger.Error("failed to save meeting", zap.Error(err))
			return c.Send("Ошибка сохранения встречи")
		}

		if summary != "" {
			return c.Send(summary)
		}
		return c.Send(transcription)
	})

	bot.Handle(tele.OnAudio, func(c tele.Context) error {
		logger.Info("audio message received", zap.Int64("user_id", c.Sender().ID))
		return c.Send("Получен аудио файл")
	})
}
