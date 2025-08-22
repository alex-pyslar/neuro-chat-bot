# Neuro Chat Bot

**Neuro Chat Bot** — это Telegram-бот, написанный на Go, который использует модель LLaMA для обработки текстовых запросов пользователей и MongoDB для хранения данных. Проект предоставляет функциональность интерактивного чат-бота с поддержкой истории сообщений и логированием.

## Технологии

- **Язык программирования**: Go
- **База данных**: MongoDB (хранение данных пользователей и чатов)
- **LLM**: LLaMA (через LLaMA C++ сервер)
- **Мессенджер**: Telegram Bot API
- **Библиотеки**:
  - [godotenv](https://github.com/joho/godotenv) для загрузки переменных окружения
  - [mongo-go-driver](https://github.com/mongodb/mongo-go-driver) для работы с MongoDB
- **Логирование**: Собственный консольный логгер

## Основные возможности

- Обработка текстовых запросов пользователей через Telegram
- Интеграция с LLaMA для генерации ответов
- Хранение данных пользователей и истории чатов в MongoDB
- Поддержка до 100 сообщений в истории чата
- Логирование всех уровней (от debug до fatal)
- Асинхронная обработка сообщений через Telegram Bot Polling

## Установка

### Предварительные требования

- Go (версия 1.16 или выше)
- MongoDB
- Telegram Bot Token (получите через [BotFather](https://t.me/BotFather))
- LLaMA C++ сервер (например, [llama-cpp-python](https://github.com/abetlen/llama-cpp-python))
- Git

### Шаги установки

1. Клонируйте репозиторий:
```bash
git clone https://github.com/alex-pyslar/neuro-chat-bot.git
cd neuro-chat-bot
```

2. Установите зависимости:
```bash
go mod download
```

3. Настройте переменные окружения:
Создайте файл `.env` в корне проекта со следующими переменными:
```bash
MONGO_URI=mongodb://localhost:27017
MONGO_DB_NAME=neuro_chat_db
TELEGRAM_BOT_TOKEN=your_telegram_bot_token
LLAMA_BASE_URL=http://localhost:8080
```

4. Настройте MongoDB:
- Убедитесь, что MongoDB запущен и доступен по указанному `MONGO_URI`.
- Создайте базу данных `neuro_chat_db` (коллекции будут созданы автоматически).

5. Настройте LLaMA C++ сервер:
- Установите и запустите LLaMA C++ сервер (по умолчанию ожидается на `http://localhost:8080`).
- Убедитесь, что модель LLaMA доступна и сервер работает.

## Запуск проекта

1. Запустите сервер LLaMA C++ (если требуется):
```bash
# Пример для llama-cpp-python
python -m llama_cpp.server --model path/to/your/model
```

2. Скомпилируйте и запустите бот:
```bash
go run main.go
```

3. Бот начнет polling Telegram API и будет готов к обработке сообщений.

## Использование

- Найдите бота в Telegram, используя его `@BotName` (заданный через BotFather).
- Отправляйте текстовые сообщения боту, и он будет отвечать, используя LLaMA для генерации ответов.
- История чата сохраняется в MongoDB.

## Логирование

- Логи выводятся в консоль с уровнями `DEBUG`, `INFO`, `WARN`, `ERROR`, `FATAL`.
- Для настройки других методов логирования (например, в файл) обновите `logger` пакет.

## Разработка

Для добавления новой функциональности:
1. Обновите схему данных в MongoDB, если требуется.
2. Расширьте `usecases.UserInteractor` для новой бизнес-логики.
3. Настройте дополнительные команды в `telegram.BotController`.

## Лицензия

[MIT License](LICENSE)