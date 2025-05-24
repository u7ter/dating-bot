package main

import (
	"database/sql"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	tele "gopkg.in/telebot.v3"
)

type Bot struct {
	*tele.Bot
	db *sql.DB
}

type User struct {
	ID          int64     `json:"id"`
	TelegramID  int64     `json:"telegram_id"`
	Name        string    `json:"name"`
	Age         int       `json:"age"`
	Gender      string    `json:"gender"`
	Description string    `json:"description"`
	City        string    `json:"city"`
	LookingFor  string    `json:"looking_for"`
	Photos      []string  `json:"photos"`
	CreatedAt   time.Time `json:"created_at"`
	IsActive    bool      `json:"is_active"`
}

type Match struct {
	ID        int64     `json:"id"`
	User1ID   int64     `json:"user1_id"`
	User2ID   int64     `json:"user2_id"`
	CreatedAt time.Time `json:"created_at"`
	IsActive  bool      `json:"is_active"`
}

type Like struct {
	ID        int64     `json:"id"`
	LikerID   int64     `json:"liker_id"`
	LikedID   int64     `json:"liked_id"`
	IsLike    bool      `json:"is_like"`
	CreatedAt time.Time `json:"created_at"`
}

type UserState struct {
	TelegramID int64
	State      string
	Data       map[string]interface{}
}

var userStates = make(map[int64]*UserState)

func main() {
	// Initialize database
	db, err := initDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Initialize bot
	bot, err := tele.NewBot(tele.Settings{
		Token:  os.Getenv("TELEGRAM_BOT_TOKEN"),
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatal("Failed to create bot:", err)
	}

	app := &Bot{Bot: bot, db: db}

	// Register handlers
	app.registerHandlers()

	log.Println("Bot started...")
	app.Start()
}

func initDB() (*sql.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "root:password@tcp(mysql:3306)/dating_bot?charset=utf8mb4&parseTime=True&loc=Local"
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// Wait for database to be ready
	for i := 0; i < 30; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		log.Println("Waiting for database...")
		time.Sleep(2 * time.Second)
	}

	return db, nil
}

func (b *Bot) registerHandlers() {
	// Commands
	b.Handle("/start", b.handleStart)
	b.Handle("/help", b.handleHelp)
	b.Handle("/profile", b.handleProfile)
	b.Handle("/edit", b.handleEdit)
	b.Handle("/search", b.handleSearch)
	b.Handle("/matches", b.handleMatches)
	b.Handle("/delete", b.handleDelete)

	// Callback buttons
	b.Handle(&btnLike, b.handleLike)
	b.Handle(&btnDislike, b.handleDislike)
	b.Handle(&btnBack, b.handleBack)
	b.Handle(&btnStartChat, b.handleStartChat)
	b.Handle(&btnEditName, b.handleEditName)
	b.Handle(&btnEditAge, b.handleEditAge)
	b.Handle(&btnEditGender, b.handleEditGender)
	b.Handle(&btnEditDescription, b.handleEditDescription)
	b.Handle(&btnEditCity, b.handleEditCity)
	b.Handle(&btnEditLookingFor, b.handleEditLookingFor)
	b.Handle(&btnEditPhotos, b.handleEditPhotos)

	// Text and photo handlers
	b.Handle(tele.OnText, b.handleText)
	b.Handle(tele.OnPhoto, b.handlePhoto)
}

// Button definitions
var (
	btnLike = tele.InlineButton{
		Unique: "like",
		Text:   "❤️",
	}
	btnDislike = tele.InlineButton{
		Unique: "dislike",
		Text:   "❌",
	}
	btnBack = tele.InlineButton{
		Unique: "back",
		Text:   "🔁 Назад",
	}
	btnStartChat = tele.InlineButton{
		Unique: "start_chat",
		Text:   "💬 Почати чат",
	}
	btnEditName = tele.InlineButton{
		Unique: "edit_name",
		Text:   "✏️ Ім'я",
	}
	btnEditAge = tele.InlineButton{
		Unique: "edit_age",
		Text:   "✏️ Вік",
	}
	btnEditGender = tele.InlineButton{
		Unique: "edit_gender",
		Text:   "✏️ Стать",
	}
	btnEditDescription = tele.InlineButton{
		Unique: "edit_description",
		Text:   "✏️ Опис",
	}
	btnEditCity = tele.InlineButton{
		Unique: "edit_city",
		Text:   "✏️ Місто",
	}
	btnEditLookingFor = tele.InlineButton{
		Unique: "edit_looking_for",
		Text:   "✏️ Шукаю",
	}
	btnEditPhotos = tele.InlineButton{
		Unique: "edit_photos",
		Text:   "📷 Фото",
	}
)

func (b *Bot) handleStart(c tele.Context) error {
	user, err := b.getUser(c.Sender().ID)
	if err != nil && err != sql.ErrNoRows {
		return c.Send("Помилка при перевірці профілю")
	}

	if user != nil {
		return c.Send("Ласкаво просимо назад! Ваш профіль вже створено.\n\nВикористовуйте /search для пошуку анкет або /profile для перегляду профілю.")
	}

	// Start registration process
	userStates[c.Sender().ID] = &UserState{
		TelegramID: c.Sender().ID,
		State:      "registration_name",
		Data:       make(map[string]interface{}),
	}

	return c.Send("Привіт! 👋 Ласкаво просимо до бота знайомств!\n\nДавайте створимо ваш профіль. Як вас звати?")
}

func (b *Bot) handleHelp(c tele.Context) error {
	help := `🤖 Довідка по боту знайомств

Команди:
/start - Почати роботу з ботом
/profile - Переглянути свій профіль  
/edit - Редагувати профіль
/search - Шукати анкети
/matches - Переглянути збіги
/delete - Видалити профіль
/help - Ця довідка

Як користуватися:
1. Створіть профіль командою /start
2. Шукайте анкети командою /search
3. Ставте ❤️ або ❌ 
4. При взаємній симпатії з'явиться збіг!
5. Переглядайте збіги командою /matches`

	return c.Send(help)
}

func (b *Bot) handleProfile(c tele.Context) error {
	user, err := b.getUser(c.Sender().ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Send("У вас ще немає профілю. Використовуйте /start для створення.")
		}
		return c.Send("Помилка при отриманні профілю")
	}

	profileText := b.formatProfile(user)

	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(btnEditName, btnEditAge),
		markup.Row(btnEditGender, btnEditCity),
		markup.Row(btnEditDescription),
		markup.Row(btnEditLookingFor, btnEditPhotos),
	)

	if len(user.Photos) > 0 {
		photo := &tele.Photo{File: tele.FromURL(user.Photos[0])}
		return c.Send(photo, profileText, markup)
	}

	return c.Send(profileText, markup)
}

func (b *Bot) handleEdit(c tele.Context) error {
	return b.handleProfile(c)
}

func (b *Bot) handleSearch(c tele.Context) error {
	user, err := b.getUser(c.Sender().ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Send("У вас ще немає профілю. Використовуйте /start для створення.")
		}
		return c.Send("Помилка при отриманні профілю")
	}

	candidate, err := b.getNextCandidate(user.TelegramID)
	if err != nil {
		return c.Send("Наразі немає доступних анкет. Спробуйте пізніше!")
	}

	return b.showCandidate(c, candidate)
}

func (b *Bot) handleMatches(c tele.Context) error {
	matches, err := b.getUserMatches(c.Sender().ID)
	if err != nil {
		return c.Send("Помилка при отриманні збігів")
	}

	if len(matches) == 0 {
		return c.Send("У вас поки що немає збігів. Продовжуйте шукати! 💪")
	}

	text := "💕 Ваші збіги:\n\n"
	for i, match := range matches {
		text += strconv.Itoa(i+1) + ". " + match.Name + ", " + strconv.Itoa(match.Age) + " років\n"
	}

	return c.Send(text)
}

func (b *Bot) handleDelete(c tele.Context) error {
	err := b.deleteUser(c.Sender().ID)
	if err != nil {
		return c.Send("Помилка при видаленні профілю")
	}

	delete(userStates, c.Sender().ID)
	return c.Send("Ваш профіль видалено. Дякуємо за використання бота! 👋")
}

func (b *Bot) handleText(c tele.Context) error {
	state, exists := userStates[c.Sender().ID]
	if !exists {
		return c.Send("Використовуйте /start для початку роботи з ботом")
	}

	switch state.State {
	case "registration_name":
		return b.handleRegistrationName(c, state)
	case "registration_age":
		return b.handleRegistrationAge(c, state)
	case "registration_gender":
		return b.handleRegistrationGender(c, state)
	case "registration_description":
		return b.handleRegistrationDescription(c, state)
	case "registration_city":
		return b.handleRegistrationCity(c, state)
	case "registration_looking_for":
		return b.handleRegistrationLookingFor(c, state)
	case "edit_name":
		return b.handleEditNameText(c, state)
	case "edit_age":
		return b.handleEditAgeText(c, state)
	case "edit_description":
		return b.handleEditDescriptionText(c, state)
	case "edit_city":
		return b.handleEditCityText(c, state)
	}

	return nil
}

func (b *Bot) handlePhoto(c tele.Context) error {
	state, exists := userStates[c.Sender().ID]
	if !exists {
		return c.Send("Використовуйте /start для початку роботи з ботом")
	}

	if state.State == "registration_photos" || state.State == "edit_photos" {
		return b.handlePhotoUpload(c, state)
	}

	return c.Send("Надішліть фото тільки під час реєстрації або редагування профілю")
}

// Database helper methods
func (b *Bot) getUser(telegramID int64) (*User, error) {
	query := `SELECT id, telegram_id, name, age, gender, description, city, looking_for, created_at, is_active FROM users WHERE telegram_id = ?`

	user := &User{}
	err := b.db.QueryRow(query, telegramID).Scan(
		&user.ID, &user.TelegramID, &user.Name, &user.Age, &user.Gender,
		&user.Description, &user.City, &user.LookingFor, &user.CreatedAt, &user.IsActive,
	)

	if err != nil {
		return nil, err
	}

	// Get photos
	photos, err := b.getUserPhotos(user.ID)
	if err == nil {
		user.Photos = photos
	}

	return user, nil
}

func (b *Bot) getUserPhotos(userID int64) ([]string, error) {
	query := `SELECT photo_url FROM user_photos WHERE user_id = ? ORDER BY id`
	rows, err := b.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var photos []string
	for rows.Next() {
		var photo string
		if err := rows.Scan(&photo); err != nil {
			continue
		}
		photos = append(photos, photo)
	}

	return photos, nil
}

func (b *Bot) createUser(user *User) error {
	query := `INSERT INTO users (telegram_id, name, age, gender, description, city, looking_for, is_active) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, true)`

	result, err := b.db.Exec(query, user.TelegramID, user.Name, user.Age, user.Gender,
		user.Description, user.City, user.LookingFor)
	if err != nil {
		return err
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return err
	}

	// Save photos
	for _, photo := range user.Photos {
		_, err := b.db.Exec(`INSERT INTO user_photos (user_id, photo_url) VALUES (?, ?)`, userID, photo)
		if err != nil {
			log.Printf("Error saving photo: %v", err)
		}
	}

	return nil
}

func (b *Bot) formatProfile(user *User) string {
	return "👤 Ваш профіль:\n\n" +
		"Ім'я: " + user.Name + "\n" +
		"Вік: " + strconv.Itoa(user.Age) + " років\n" +
		"Стать: " + user.Gender + "\n" +
		"Місто: " + user.City + "\n" +
		"Шукаю: " + user.LookingFor + "\n" +
		"Опис: " + user.Description + "\n" +
		"Фото: " + strconv.Itoa(len(user.Photos)) + " шт."
}

// Additional handler methods would continue here...
// This includes registration flow, editing, matching logic, etc.
