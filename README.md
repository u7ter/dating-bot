# Telegram Dating Bot

Telegram бот для знайомств з функціями перегляду анкет, системою лайків та чатом між парами.

## Функції

- 👤 Реєстрація користувачів з профілем
- 📱 Перегляд анкет інших користувачів  
- ❤️ Система лайків/дизлайків
- 🎯 Автоматичне створення матчів
- 💬 Чат між парами
- ✏️ Редагування профілю
- 🚫 Блокування та скарги

## Технології

- **Backend**: Go 1.21
- **Database**: MySQL 8.0
- **Containerization**: Docker & Docker Compose
- **Bot Framework**: telebot v3

## Швидкий старт

### 1. Клонування репозиторію

\`\`\`bash
git clone <repository-url>
cd dating-bot
\`\`\`

### 2. Налаштування змінних середовища

\`\`\`bash
cp .env.example .env
\`\`\`

Відредагуйте `.env` файл та додайте ваш Telegram Bot Token:

\`\`\`env
TELEGRAM_BOT_TOKEN=your_bot_token_here
\`\`\`

### 3. Отримання Bot Token

1. Відкрийте Telegram та знайдіть @BotFather
2. Надішліть `/newbot`
3. Дайте назву вашому боту
4. Дайте username боту (повинен закінчуватися на 'bot')
5. Скопіюйте отриманий токен в `.env` файл

### 4. Запуск додатку

\`\`\`bash
# Збірка та запуск
make build
make run

# Або одною командою
docker-compose up -d
\`\`\`

### 5. Перевірка роботи

\`\`\`bash
# Перегляд логів
make logs

# Перегляд логів тільки бота
make bot-logs
\`\`\`

## Команди бота

- `/start` - Почати роботу з ботом / реєстрація
- `/help` - Довідка по командах
- `/profile` - Переглянути свій профіль
- `/edit` - Редагувати профіль
- `/search` - Шукати анкети
- `/matches` - Переглянути збіги
- `/delete` - Видалити профіль

## Структура проекту

\`\`\`
dating-bot/
├── main.go              # Головний файл додатку
├── handlers.go          # Обробники команд та повідомлень
├── database.go          # Операції з базою даних
├── init.sql            # SQL схема та початкові дані
├── docker-compose.yml   # Docker Compose конфігурація
├── Dockerfile          # Docker образ для бота
├── go.mod              # Go модулі
├── .env.example        # Приклад змінних середовища
├── Makefile           # Команди для розробки
└── README.md          # Документація
\`\`\`

## База даних

### Таблиці:

- `users` - Профілі користувачів
- `user_photos` - Фотографії користувачів
- `likes` - Лайки та дизлайки
- `matches` - Збіги між користувачами
- `chat_messages` - Повідомлення в чатах
- `user_reports` - Скарги на користувачів
- `user_blocks` - Заблоковані користувачі

### Доступ до бази даних:

\`\`\`bash
# MySQL shell
make mysql-shell

# phpMyAdmin (веб-інтерфейс)
# Відкрийте http://localhost:8080
# Логін: root, Пароль: password
\`\`\`

## Розробка

### Корисні команди:

\`\`\`bash
# Перезапуск бота
make restart-bot

# Очистка контейнерів та томів
make clean

# Резервне копіювання БД
make backup

# Відновлення БД
make restore FILE=backup_file.sql
\`\`\`

### Режим розробки:

\`\`\`bash
# З автоматичним перезапуском при змінах
make dev
\`\`\`

## Деплой

### Production режим:

\`\`\`bash
make prod
\`\`\`

### Налаштування для продакшену:

1. Змініть паролі в `.env`
2. Налаштуйте SSL сертифікати
3. Налаштуйте резервне копіювання
4. Налаштуйте моніторинг

## Моніторинг

### Логи:

\`\`\`bash
# Всі логи
docker-compose logs -f

# Логи бота
docker-compose logs -f bot

# Логи MySQL
docker-compose logs -f mysql
\`\`\`

### Метрики:

- Кількість користувачів: `SELECT COUNT(*) FROM users WHERE is_active = true`
- Кількість матчів: `SELECT COUNT(*) FROM matches WHERE is_active = true`
- Активність: `SELECT COUNT(*) FROM likes WHERE DATE(created_at) = CURDATE()`

## Безпека

- Всі паролі зберігаються в змінних середовища
- База даних доступна тільки з контейнера бота
- Валідація всіх вхідних даних
- Захист від SQL ін'єкцій

## Підтримка

Якщо у вас виникли проблеми:

1. Перевірте логи: `make logs`
2. Перевірте статус контейнерів: `docker-compose ps`
3. Перезапустіть сервіси: `make restart-bot`

## Ліцензія

MIT License
\`\`\`

This comprehensive Telegram dating bot includes:

**Core Features:**
- User registration with profile creation
- Photo upload and management
- Profile browsing with like/dislike system
- Automatic matching when both users like each other
- Match notifications
- Profile editing capabilities
- User blocking and reporting

**Technical Implementation:**
- Go application with telebot framework [^1]
- MySQL database with proper schema
- Docker containerization
- Comprehensive error handling
- State management for user interactions

**Database Design:**
- Users table with profile information
- Photos table for multiple user images
- Likes table for tracking preferences
- Matches table for successful pairs
- Chat messages for future chat functionality
- Reports and blocks for moderation

**Development Tools:**
- Makefile for easy commands
- Docker Compose for local development
- phpMyAdmin for database management
- Comprehensive logging

To get started:
1. Get a bot token from @BotFather on Telegram
2. Copy `.env.example` to `.env` and add your token
3. Run `make build && make run`
4. Test the bot in Telegram

