# 🎙️ AudioTranscrib Bot

Telegram-бот для расшифровки аудио, генерации summary и поиска по встречам.

---

## 🚀 Возможности

* 🎧 Распознавание аудио (voice, audio, документы)
* 📝 Транскрипция речи в текст
* 🤖 Генерация summary через AI
* 📚 Хранение встреч в базе данных
* 🔍 Поиск по транскрипциям
* 💬 AI-чат для вопросов

---

## 🏗️ Архитектура

Проект построен по layered architecture:

```
Telegram handlers
        ↓
   Repositories
        ↓
   Base Repository
        ↓
     DBStorage
        ↓
   PostgreSQL
```

### Основные компоненты:

* `telegram/` — обработчики Telegram
* `storage/` — работа с БД (Repository pattern)
* `speech/` — интеграция с распознаванием речи
* `ai/` — интеграция с GigaChat
* `config/` — конфигурация

---

## 🧱 Технологии

* Go
* PostgreSQL
* Telebot (Telegram API)
* Uber FX (DI)
* Zap (логирование)

---

## ⚙️ Установка

### 1. Клонирование

```bash
git clone https://github.com/yourusername/audiotranscrib.git
cd audiotranscrib
```

---

### 2. Настройка окружения

Создай `.env` файл:

```env
TELEGRAM_TOKEN=your_token
DATABASE_DSN=postgres://user:password@localhost:5432/dbname?sslmode=disable

GIGACHAT_API_KEY=your_key
SPEECH_API_KEY=your_key
```

---

### 3. Запуск PostgreSQL

Через Docker:

```bash
docker run -d \
  --name postgres \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=audiotranscrib \
  -p 5432:5432 \
  postgres:15
```

---

### 4. Запуск приложения

```bash
go run cmd/app/main.go
```

---

## 📦 Миграции

Миграции применяются автоматически при запуске:

```go
RunMigrations(...)
```

---

## 🤖 Команды бота

| Команда          | Описание                 |
| ---------------- | ------------------------ |
| `/start`         | Регистрация пользователя |
| `/list`          | Список встреч            |
| `/get <id>`      | Получить встречу         |
| `/find <текст>`  | Поиск по встречам        |
| `/chat <вопрос>` | Задать вопрос AI         |

---

## 🔄 Обработка аудио

Поддерживается:

* Voice (ogg)
* Audio (mp3, wav и др.)
* Документы

Если формат не поддерживается напрямую:
👉 выполняется конвертация в PCM 16kHz

---

## 🗄️ Работа с БД

Используется Repository Pattern:

* `UserRepo`
* `MeetingRepo`
* `Repository` (base layer)

Преимущества:

* нет дублирования SQL-инфраструктуры
* чистая архитектура
* легко тестировать

---

## ⚠️ Ограничения

* Максимальный размер файла: **20MB**
* Длина сообщения Telegram: ~4000 символов (разбивается автоматически)

---

## 📈 Планы развития

* [ ] Поиск через SQL (ILIKE + индексы)
* [ ] Service layer
* [ ] Переход на pgx
* [ ] Использование sqlc
* [ ] Кэширование

---

## 🧪 Тестирование

```bash
go test ./...
```

---

## 📝 Логирование

Используется `zap`:

* structured logs
* уровни: info / warn / error

---

## 👨‍💻 Автор

Разработано в рамках выпускного проекта 🚀

---

## 📄 Лицензия

MIT
