package telegram

import (
	"audiotranscrib/internal/ai"
	"audiotranscrib/internal/speech"
	"audiotranscrib/internal/storage"
	"context"
	"database/sql"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func registerHandlers(
	bot *tele.Bot,
	repository *storage.DBStorage,
	userRepo *storage.UserRepo,
	speechClient *speech.Client,
	gptClient *ai.GigaChatClient,
	logger *zap.Logger,
) {

	// --- helpers ---

	getOrCreateUser := func(ctx context.Context, sender *tele.User) (*storage.User, error) {
		user, err := userRepo.GetUserByTelegramID(ctx, sender.ID)
		if err != nil {
			if err == sql.ErrNoRows {
				return userRepo.CreateUser(ctx, sender.ID, sender.Username)
			}
			return nil, err
		}
		return user, nil
	}

	processAudio := func(c tele.Context, file tele.File, mime string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		sender := c.Sender()

		user, err := getOrCreateUser(ctx, sender)
		if err != nil {
			logger.Error("user error", zap.Error(err))
			return c.Send("Ошибка пользователя")
		}

		reader, err := bot.File(&file)
		if err != nil {
			logger.Error("failed to get file", zap.Error(err))
			return c.Send("Ошибка загрузки файла")
		}
		defer reader.Close()

		data, err := io.ReadAll(reader)
		if err != nil {
			logger.Error("read file failed", zap.Error(err))
			return c.Send("Ошибка чтения файла")
		}

		// 🔒 ограничение размера (например 20MB)
		if len(data) > 20*1024*1024 {
			return c.Send("Файл слишком большой (макс 20MB)")
		}

		strategy := detectStrategy(mime)

		var finalData []byte
		finalMime := mime

		switch strategy {

		case StrategyDirect:
			logger.Info("direct audio processing", zap.String("mime", mime))
			finalData = data

		case StrategyConvert:
			logger.Info("converting audio", zap.String("mime", mime))

			finalData, err = ConvertToPCM16k(data, logger)
			if err != nil {
				logger.Error("audio conversion failed", zap.Error(err))
				return c.Send("Ошибка обработки аудио")
			}

			finalMime = "audio/wav"
		}

		// отправка в SaluteSpeech
		transcription, err := speechClient.Recognize(ctx, finalData, finalMime)
		if err != nil {
			logger.Error("speech recognition failed", zap.Error(err))
			return c.Send("Ошибка распознавания")
		}

		var summary string
		if transcription != "" {
			logger.Info("sending transcription to GigaChat",
				zap.Int64("user_id", sender.ID),
			)

			summary, err = gptClient.GetSummary(ctx, transcription)
			if err != nil {
				logger.Warn("failed to get summary", zap.Error(err))
				summary = ""
			}
		}

		_, err = repository.SaveMeeting(ctx, user.ID, file.FileID, transcription, summary)
		if err != nil {
			logger.Error("failed to save meeting", zap.Error(err))
			return c.Send("Ошибка сохранения встречи")
		}

		if summary != "" {
			return c.Send(summary)
		}
		return c.Send(transcription)
	}

	// --- commands ---

	bot.Handle(tele.OnDocument, func(c tele.Context) error {
		msg := c.Message()
		if msg == nil || msg.Document == nil {
			logger.Warn("document is nil")
			return c.Send("Не удалось получить файл")
		}

		doc := msg.Document

		logger.Info("document received",
			zap.String("name", doc.FileName),
			zap.String("mime", doc.MIME),
			zap.Int64("size", doc.FileSize),
		)

		return processAudio(c, doc.File, doc.MIME)
	})

	bot.Handle("/start", func(c tele.Context) error {
		ctx := context.Background()

		err := repository.CreateUser(ctx, c.Sender().ID, c.Sender().Username)
		if err != nil {
			logger.Error("cannot create user", zap.Error(err))
			return c.Send("Ошибка регистрации")
		}

		logger.Info("user registered", zap.Int64("user_id", c.Sender().ID))
		return c.Send("Добро пожаловать")
	})

	bot.Handle("/list", func(c tele.Context) error {
		ctx := context.Background()

		user, err := userRepo.GetUserByTelegramID(ctx, c.Sender().ID)
		if err != nil {
			return c.Send("Пользователь не найден")
		}

		meetings, err := repository.ListMeetings(ctx, user.ID)
		if err != nil {
			return c.Send("Ошибка получения списка")
		}

		if len(meetings) == 0 {
			return c.Send("У вас пока нет встреч")
		}

		var resp string
		resp += "📚 Ваши встречи:\n\n"

		for _, m := range meetings {
			resp += fmt.Sprintf(
				"ID: %d\n📅 %s\n📎 %s\n📝 %s\n\n",
				m.ID,
				m.CreatedAt.Format("02.01.2006 15:04"),
				m.FileName,
				truncate(m.Transcription, 200),
			)
		}

		return sendLongMessage(c, resp, logger)
	})

	bot.Handle("/get", func(c tele.Context) error {
		ctx := context.Background()

		args := c.Args()
		if len(args) == 0 {
			return c.Send("Использование: /get <id>")
		}

		id, err := strconv.Atoi(args[0])
		if err != nil {
			return c.Send("Некорректный ID")
		}

		user, err := userRepo.GetUserByTelegramID(ctx, c.Sender().ID)
		if err != nil {
			return c.Send("Пользователь не найден")
		}

		meeting, err := repository.GetMeeting(ctx, id, user.ID)
		if err != nil {
			return c.Send("Встреча не найдена")
		}

		resp := fmt.Sprintf(
			"📝 Транскрипция:\n\n%s\n\n📌 Summary:\n%s",
			meeting.Transcription,
			meeting.Summary,
		)

		if len(resp) > 4000 {
			resp = resp[:4000] + "..."
		}

		return c.Send(resp)
	})

	bot.Handle("/find", func(c tele.Context) error {
		ctx := context.Background()

		args := c.Args()
		if len(args) == 0 {
			return c.Send("Использование: /find <слово>")
		}

		query := strings.Join(args, " ")

		user, err := userRepo.GetUserByTelegramID(ctx, c.Sender().ID)
		if err != nil {
			return c.Send("Пользователь не найден")
		}

		results, err := repository.FindMeetings(ctx, user.ID, query)
		if err != nil {
			return c.Send("Ошибка поиска")
		}

		if len(results) == 0 {
			return c.Send("Ничего не найдено")
		}

		var resp string
		resp += "🔎 Найдено:\n\n"

		for _, r := range results {
			resp += fmt.Sprintf(
				"ID: %d\n📌 %s\n📝 %s\n\n",
				r.ID,
				r.Summary,
				truncate(r.Transcription, 200),
			)
		}

		return sendLongMessage(c, resp, logger)
	})

	bot.Handle("/chat", func(c tele.Context) error {
		ctx := context.Background()

		args := c.Args()
		if len(args) == 0 {
			return c.Send(
				"🤖 Напиши вопрос после команды\n\n" +
					"Пример:\n" +
					"/chat что обсуждали на встрече?",
			)
		}

		question := strings.Join(args, " ")

		answer, err := gptClient.Ask(ctx, question)
		if err != nil {
			return c.Send("Ошибка ИИ")
		}

		return c.Send(answer)
	})

	// --- voice ---

	bot.Handle(tele.OnVoice, func(c tele.Context) error {
		msg := c.Message()
		if msg == nil || msg.Voice == nil {
			logger.Warn("voice message is nil")
			return c.Send("Не удалось получить голосовое сообщение")
		}

		file := msg.Voice.File

		mime := "audio/ogg"

		return processAudio(c, file, mime)
	})

	// --- audio ---

	bot.Handle(tele.OnAudio, func(c tele.Context) error {
		msg := c.Message()

		if msg == nil || msg.Audio == nil {
			logger.Warn("audio message is nil")
			return c.Send("Не удалось получить аудио")
		}

		audio := msg.Audio
		if audio.FileSize > 20*1024*1024 {
			return c.Send(
				"Файл слишком большой 😢\n\n" +
					"Попробуйте:\n" +
					"• сжать аудио\n" +
					"• отправить как voice\n" +
					"• разбить на части",
			)
		}

		logger.Info("audio meta",
			zap.String("mime", audio.MIME),
			zap.Int64("size", audio.FileSize),
		)

		return processAudio(c, audio.File, audio.MIME)
	})
}

func truncate(text string, max int) string {
	if len(text) <= max {
		return text
	}
	return text[:max] + "..."
}

func sendLongMessage(c tele.Context, text string, logger *zap.Logger) error {
	const chunkSize = 4000

	logger.Info("sending message", zap.Int("length", len(text)))

	for len(text) > chunkSize {
		chunk := text[:chunkSize]

		logger.Info("sending chunk", zap.Int("size", len(chunk)))

		err := c.Send(chunk)
		if err != nil {
			logger.Error("failed to send chunk", zap.Error(err))
			return err
		}

		text = text[chunkSize:]
	}

	if len(text) > 0 {
		logger.Info("sending last chunk", zap.Int("size", len(text)))

		return c.Send(text)
	}

	return nil
}
