# Telegram Dating Bot

Telegram бот для знайомств з функціями перегляду анкет, системою лайків та чатом між парами.

## Функції

- 👤 Реєстрація користувачів з профілем
- 📱 Перегляд анкет інших користувачів
- ❤️ Система лайків/дизлайків
- 🎯 Автоматичне створення матчів
- 💬 Чат між парами (в розробці)
- ✏️ Редагування профілю
- 🚫 Блокування та скарги

## Технології

- **Backend**: Go 1.21
- **Database**: MySQL 8.0
- **Cache**: Redis 7
- **Containerization**: Docker & Docker Compose
- **Bot Framework**: telebot v3
- **Load Balancer**: Nginx
- **CI/CD**: GitHub Actions

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

# Перевірка здоров'я
curl http://localhost:8080/health
\`\`\`

## Команди бота

- `/start` - Почати роботу з ботом / реєстрація
- `/help` - Довідка по командах
- `/profile` - Переглянути свій профіль
- `/edit` - Редагувати профіль
- `/search` - Шукати анкети
- `/matches` - Переглянути збіги
- `/delete` - Видалити профіль

## Архітектура

### Компоненти:
- **Bot Service**: Основний сервіс бота
- **MySQL**: База даних для зберігання профілів
- **Redis**: Кеш та черги повідомлень
- **Nginx**: Load balancer та reverse proxy

### Високі навантаження:
- Connection pooling для бази даних
- Redis кешування користувачів
- Rate limiting для запобігання спаму
- Асинхронна обробка матчів
- Горизонтальне масштабування

## Розробка

### Корисні команди:

\`\`\`bash
# Перезапуск бота
make restart-bot

# Масштабування
make scale

# Очистка
make clean

# Доступ до MySQL
make mysql-shell

# Доступ до Redis
make redis-cli

# Перегляд ресурсів
make resources
\`\`\`

### Тестування:

\`\`\`bash
# Запуск тестів
make test

# Навантажувальне тестування
make load-test
\`\`\`

## Деплой

### GitHub Actions:

Проект налаштований для автоматичного деплою через GitHub Actions:

1. **CI Pipeline**: Тестування, лінтинг, сканування безпеки
2. **Staging Deploy**: Автоматичний деплой на staging при push в `develop`
3. **Production Deploy**: Деплой на production при push в `main`

### Налаштування секретів:

\`\`\`
TELEGRAM_BOT_TOKEN - токен бота
AWS_ACCESS_KEY_ID - AWS ключ
AWS_SECRET_ACCESS_KEY - AWS секрет
SLACK_WEBHOOK_URL - webhook для сповіщень
\`\`\`

### Локальний продакшн:

\`\`\`bash
make prod
\`\`\`

## Моніторинг

### Метрики:
- Кількість користувачів
- Кількість матчів
- Швидкість відповіді
- Використання ресурсів

### Health Check:
\`\`\`bash
curl http://localhost:8080/health
\`\`\`

### Логи:
\`\`\`bash
# Всі логи
make logs

# Логи бота
make bot-logs

# Логи бази даних
make db-logs
\`\`\`

## Безпека

- Rate limiting для запобігання спаму
- Валідація всіх вхідних даних
- Захист від SQL ін'єкцій
- Безпечне зберігання паролів
- HTTPS для продакшену

## Структура проекту

\`\`\`
dating-bot/
├── main.go              # Головний файл
├── handlers.go          # Обробники команд
├── database.go          # Операції з БД
├── cache.go            # Redis кеш
├── rate_limiter.go     # Rate limiting
├── queue.go            # Черги повідомлень
├── metrics.go          # Метрики
├── health.go           # Health checks
├── init.sql            # SQL схема
├── docker-compose.yml  # Docker конфігурація
├── Dockerfile          # Docker образ
├── .github/workflows/  # GitHub Actions
├── k8s/               # Kubernetes manifests
└── README.md          # Документація
\`\`\`

## Підтримка

Якщо у вас виникли проблеми:

1. Перевірте логи: `make logs`
2. Перевірте статус: `curl http://localhost:8080/health`
3. Перезапустіть сервіси: `make restart-bot`
4. Створіть issue в GitHub

## Ліцензія

MIT License
\`\`\`

Це повний набір файлів проекту з усіма актуальними змінами:

**Ключові особливості:**
- ✅ Виправлені кнопки вибору статі та інших параметрів
- ✅ Повна система кешування з Redis
- ✅ Rate limiting та захист від перевантажень
- ✅ Асинхронна обробка матчів через черги
- ✅ Горизонтальне масштабування
- ✅ GitHub Actions для CI/CD
- ✅ Health checks та моніторинг
- ✅ Безпека та валідація даних
- ✅ Повна документація

Проект готовий до розгортання в продакшені та може витримувати високі навантаження з тисячами користувачів одночасно.