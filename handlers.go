package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v3"
)

// Fixed button definitions with proper callback handling
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

	// Gender selection buttons
	btnGenderMale = tele.InlineButton{
		Unique: "gender_male",
		Text:   "👨 Чоловік",
	}
	btnGenderFemale = tele.InlineButton{
		Unique: "gender_female",
		Text:   "👩 Жінка",
	}

	// Looking for buttons
	btnLookingMale = tele.InlineButton{
		Unique: "looking_male",
		Text:   "👨 Чоловіків",
	}
	btnLookingFemale = tele.InlineButton{
		Unique: "looking_female",
		Text:   "👩 Жінок",
	}
	btnLookingAll = tele.InlineButton{
		Unique: "looking_all",
		Text:   "👥 Усіх",
	}

	// Edit profile buttons
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

func (b *Bot) registerHandlers() {
	// Commands
	b.Handle("/start", b.handleStart)
	b.Handle("/help", b.handleHelp)
	b.Handle("/profile", b.handleProfile)
	b.Handle("/edit", b.handleEdit)
	b.Handle("/search", b.handleSearch)
	b.Handle("/matches", b.handleMatches)
	b.Handle("/delete", b.handleDelete)

	// Callback buttons for candidate browsing
	b.Handle(&btnLike, b.handleLike)
	b.Handle(&btnDislike, b.handleDislike)
	b.Handle(&btnBack, b.handleBack)
	b.Handle(&btnStartChat, b.handleStartChat)

	// Gender selection callbacks
	b.Handle(&btnGenderMale, b.handleGenderSelection)
	b.Handle(&btnGenderFemale, b.handleGenderSelection)

	// Looking for selection callbacks
	b.Handle(&btnLookingMale, b.handleLookingForSelection)
	b.Handle(&btnLookingFemale, b.handleLookingForSelection)
	b.Handle(&btnLookingAll, b.handleLookingForSelection)

	// Edit profile callbacks
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

	// Add rate limiting middleware
	b.Use(b.RateLimitMiddleware())
	b.Use(b.PerformanceMiddleware())
}

func (b *Bot) handleStart(c tele.Context) error {
	// Check if user already exists
	user, err := b.getUser(c.Sender().ID)
	if err != nil && err != sql.ErrNoRows {
		return c.Send("Помилка при перевірці профілю")
	}

	if user != nil {
		return c.Send("Ласкаво просимо назад! Ваш профіль вже створено.\n\nВикористовуйте /search для пошуку анкет або /profile для перегляду профілю.")
	}

	// Start registration process
	state := &UserState{
		TelegramID: c.Sender().ID,
		State:      "registration_name",
		Data:       make(map[string]interface{}),
	}

	// Store in both cache and memory
	b.userCache.SetUserState(c.Sender().ID, state)
	userStatesMutex.Lock()
	userStates[c.Sender().ID] = state
	userStatesMutex.Unlock()

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
	// Check rate limiting
	allowed, err := b.rateLimiter.IsActionAllowed(c.Sender().ID, "search")
	if err != nil {
		log.Printf("Rate limiter error: %v", err)
	} else if !allowed {
		return c.Send("⚠️ Забагато пошуків. Спробуйте через хвилину.")
	}

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

	// Track search activity
	b.userCache.IncrementDailyStats("searches_performed")

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
		text += fmt.Sprintf("%d. %s, %d років\n", i+1, match.Name, match.Age)
	}

	return c.Send(text)
}

func (b *Bot) handleDelete(c tele.Context) error {
	err := b.deleteUser(c.Sender().ID)
	if err != nil {
		return c.Send("Помилка при видаленні профілю")
	}

	// Clear cache and state
	b.userCache.DeleteUser(c.Sender().ID)
	b.userCache.DeleteUserState(c.Sender().ID)
	userStatesMutex.Lock()
	delete(userStates, c.Sender().ID)
	userStatesMutex.Unlock()

	return c.Send("Ваш профіль видалено. Дякуємо за використання бота! 👋")
}

func (b *Bot) handleText(c tele.Context) error {
	// Check rate limiting first
	allowed, err := b.rateLimiter.IsActionAllowed(c.Sender().ID, "message")
	if err != nil {
		log.Printf("Rate limiter error: %v", err)
	} else if !allowed {
		return c.Send("⚠️ Забагато повідомлень. Спробуйте через хвилину.")
	}

	// Track user activity
	b.userCache.SetUserActivity(c.Sender().ID)

	// Get user state from cache first, then fallback to memory
	state, err := b.userCache.GetUserState(c.Sender().ID)
	if err != nil {
		// Fallback to memory state
		userStatesMutex.RLock()
		state, exists := userStates[c.Sender().ID]
		userStatesMutex.RUnlock()
		if !exists {
			return c.Send("Використовуйте /start для початку роботи з ботом")
		}
	}

	text := strings.TrimSpace(c.Text())

	// Handle "готово" for photo upload
	if (state.State == "registration_photos" || state.State == "edit_photos") &&
		(strings.ToLower(text) == "готово" || strings.ToLower(text) == "готов") {
		return b.handlePhotosComplete(c, state)
	}

	switch state.State {
	case "registration_name":
		return b.handleRegistrationName(c, state)
	case "registration_age":
		return b.handleRegistrationAge(c, state)
	case "registration_description":
		return b.handleRegistrationDescription(c, state)
	case "registration_city":
		return b.handleRegistrationCity(c, state)
	case "edit_name":
		return b.handleEditNameText(c, state)
	case "edit_age":
		return b.handleEditAgeText(c, state)
	case "edit_description":
		return b.handleEditDescriptionText(c, state)
	case "edit_city":
		return b.handleEditCityText(c, state)
	default:
		return c.Send("Використовуйте команди меню або /help для довідки")
	}
}

func (b *Bot) handlePhoto(c tele.Context) error {
	// Get user state
	state, err := b.userCache.GetUserState(c.Sender().ID)
	if err != nil {
		userStatesMutex.RLock()
		state, exists := userStates[c.Sender().ID]
		userStatesMutex.RUnlock()
		if !exists {
			return c.Send("Використовуйте /start для початку роботи з ботом")
		}
	}

	if state.State == "registration_photos" || state.State == "edit_photos" {
		return b.handlePhotoUpload(c, state)
	}

	return c.Send("Надішліть фото тільки під час реєстрації або редагування профілю")
}

// Registration handlers
func (b *Bot) handleRegistrationName(c tele.Context, state *UserState) error {
	name := strings.TrimSpace(c.Text())
	if len(name) < 2 || len(name) > 50 {
		return c.Send("Ім'я повинно містити від 2 до 50 символів. Спробуйте ще раз:")
	}

	state.Data["name"] = name
	state.State = "registration_age"

	// Update state in cache
	b.userCache.SetUserState(c.Sender().ID, state)

	return c.Send("Чудово! Скільки вам років? (введіть число від 18 до 99)")
}

func (b *Bot) handleRegistrationAge(c tele.Context, state *UserState) error {
	age, err := strconv.Atoi(strings.TrimSpace(c.Text()))
	if err != nil || age < 18 || age > 99 {
		return c.Send("Будь ласка, введіть коректний вік (число від 18 до 99):")
	}

	state.Data["age"] = age
	state.State = "registration_gender"

	// Update state in cache
	b.userCache.SetUserState(c.Sender().ID, state)

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(btnGenderMale, btnGenderFemale))

	return c.Send("Оберіть вашу стать:", markup)
}

// Fixed gender selection handler
func (b *Bot) handleGenderSelection(c tele.Context) error {
	// Get user state
	state, err := b.userCache.GetUserState(c.Sender().ID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{
			Text: "Помилка: стан користувача не знайдено. Почніть спочатку з /start",
		})
	}

	var gender string
	switch c.Callback().Unique {
	case "gender_male":
		gender = "чоловік"
	case "gender_female":
		gender = "жінка"
	default:
		return c.Respond(&tele.CallbackResponse{Text: "Невідома стать"})
	}

	state.Data["gender"] = gender

	if state.State == "registration_gender" {
		state.State = "registration_description"
		b.userCache.SetUserState(c.Sender().ID, state)

		return c.Edit("Розкажіть трохи про себе (до 300 символів):")
	} else if state.State == "edit_gender" {
		// Update user in database
		err := b.updateUserField(c.Sender().ID, "gender", gender)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{Text: "Помилка оновлення"})
		}

		// Clear cache and state
		b.userCache.DeleteUser(c.Sender().ID)
		b.userCache.DeleteUserState(c.Sender().ID)

		return c.Edit(fmt.Sprintf("Стать оновлено на: %s ✅", gender))
	}

	return c.Respond(&tele.CallbackResponse{Text: "Стать обрано: " + gender})
}

func (b *Bot) handleRegistrationDescription(c tele.Context, state *UserState) error {
	description := strings.TrimSpace(c.Text())
	if len(description) > 300 {
		return c.Send("Опис занадто довгий. Максимум 300 символів. Спробуйте ще раз:")
	}

	state.Data["description"] = description
	state.State = "registration_city"

	b.userCache.SetUserState(c.Sender().ID, state)

	return c.Send("В якому місті ви живете?")
}

func (b *Bot) handleRegistrationCity(c tele.Context, state *UserState) error {
	city := strings.TrimSpace(c.Text())
	if len(city) < 2 || len(city) > 50 {
		return c.Send("Назва міста повинна містити від 2 до 50 символів. Спробуйте ще раз:")
	}

	state.Data["city"] = city
	state.State = "registration_looking_for"

	b.userCache.SetUserState(c.Sender().ID, state)

	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(btnLookingMale, btnLookingFemale),
		markup.Row(btnLookingAll),
	)

	return c.Send("Кого ви шукаете?", markup)
}

// Fixed looking for selection handler
func (b *Bot) handleLookingForSelection(c tele.Context) error {
	state, err := b.userCache.GetUserState(c.Sender().ID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{
			Text: "Помилка: стан користувача не знайдено. Почніть спочатку з /start",
		})
	}

	var lookingFor string
	switch c.Callback().Unique {
	case "looking_male":
		lookingFor = "чоловік"
	case "looking_female":
		lookingFor = "жінка"
	case "looking_all":
		lookingFor = "усі"
	default:
		return c.Respond(&tele.CallbackResponse{Text: "Невідомий вибір"})
	}

	state.Data["looking_for"] = lookingFor

	if state.State == "registration_looking_for" {
		state.State = "registration_photos"
		b.userCache.SetUserState(c.Sender().ID, state)

		return c.Edit("Тепер надішліть від 1 до 5 фотографій. Коли закінчите, напишіть 'готово'")
	} else if state.State == "edit_looking_for" {
		// Update user in database
		err := b.updateUserField(c.Sender().ID, "looking_for", lookingFor)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{Text: "Помилка оновлення"})
		}

		// Clear cache and state
		b.userCache.DeleteUser(c.Sender().ID)
		b.userCache.DeleteUserState(c.Sender().ID)

		return c.Edit(fmt.Sprintf("Пошук оновлено на: %s ✅", lookingFor))
	}

	return c.Respond(&tele.CallbackResponse{Text: "Вибір збережено: " + lookingFor})
}

func (b *Bot) handlePhotoUpload(c tele.Context, state *UserState) error {
	if state.Data["photos"] == nil {
		state.Data["photos"] = []string{}
	}

	photos := state.Data["photos"].([]string)
	if len(photos) >= 5 {
		return c.Send("Максимум 5 фотографій. Напишіть 'готово' для завершення")
	}

	// Save photo (in real implementation, upload to file storage)
	photoID := c.Message().Photo.FileID
	photos = append(photos, photoID)
	state.Data["photos"] = photos

	b.userCache.SetUserState(c.Sender().ID, state)

	return c.Send(fmt.Sprintf("Фото збережено (%d/5). Надішліть ще фото або напишіть 'готово'", len(photos)))
}

func (b *Bot) handlePhotosComplete(c tele.Context, state *UserState) error {
	photos, ok := state.Data["photos"].([]string)
	if !ok || len(photos) == 0 {
		return c.Send("Будь ласка, надішліть хоча б одне фото перед завершенням")
	}

	if state.State == "registration_photos" {
		// Complete registration
		return b.completeRegistration(c, state)
	} else if state.State == "edit_photos" {
		// Update photos in database
		return b.updateUserPhotos(c, state)
	}

	return nil
}

func (b *Bot) completeRegistration(c tele.Context, state *UserState) error {
	// Create user object
	user := &User{
		TelegramID:  c.Sender().ID,
		Name:        state.Data["name"].(string),
		Age:         state.Data["age"].(int),
		Gender:      state.Data["gender"].(string),
		Description: state.Data["description"].(string),
		City:        state.Data["city"].(string),
		LookingFor:  state.Data["looking_for"].(string),
		Photos:      state.Data["photos"].([]string),
		IsActive:    true,
	}

	// Save to database
	err := b.createUser(user)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		return c.Send("Помилка при створенні профілю. Спробуйте пізніше.")
	}

	// Cache the user
	b.userCache.SetUser(user)

	// Clear state
	b.userCache.DeleteUserState(c.Sender().ID)
	userStatesMutex.Lock()
	delete(userStates, c.Sender().ID)
	userStatesMutex.Unlock()

	// Update daily stats
	b.userCache.IncrementDailyStats("new_registrations")

	successMsg := fmt.Sprintf(`🎉 Вітаємо! Ваш профіль створено!

👤 %s, %d років
📍 %s
💭 %s
📷 %d фото

Тепер ви можете:
/search - Шукати анкети
/profile - Переглянути профіль
/matches - Переглянути збіги`,
		user.Name, user.Age, user.City, user.Description, len(user.Photos))

	return c.Send(successMsg)
}

// Edit handlers
func (b *Bot) handleEditName(c tele.Context) error {
	state := &UserState{
		TelegramID: c.Sender().ID,
		State:      "edit_name",
		Data:       make(map[string]interface{}),
	}
	b.userCache.SetUserState(c.Sender().ID, state)
	return c.Edit("Введіть нове ім'я:")
}

func (b *Bot) handleEditAge(c tele.Context) error {
	state := &UserState{
		TelegramID: c.Sender().ID,
		State:      "edit_age",
		Data:       make(map[string]interface{}),
	}
	b.userCache.SetUserState(c.Sender().ID, state)
	return c.Edit("Введіть новий вік:")
}

func (b *Bot) handleEditGender(c tele.Context) error {
	state := &UserState{
		TelegramID: c.Sender().ID,
		State:      "edit_gender",
		Data:       make(map[string]interface{}),
	}
	b.userCache.SetUserState(c.Sender().ID, state)

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(btnGenderMale, btnGenderFemale))

	return c.Edit("Оберіть стать:", markup)
}

func (b *Bot) handleEditDescription(c tele.Context) error {
	state := &UserState{
		TelegramID: c.Sender().ID,
		State:      "edit_description",
		Data:       make(map[string]interface{}),
	}
	b.userCache.SetUserState(c.Sender().ID, state)
	return c.Edit("Введіть новий опис:")
}

func (b *Bot) handleEditCity(c tele.Context) error {
	state := &UserState{
		TelegramID: c.Sender().ID,
		State:      "edit_city",
		Data:       make(map[string]interface{}),
	}
	b.userCache.SetUserState(c.Sender().ID, state)
	return c.Edit("Введіть нове місто:")
}

func (b *Bot) handleEditLookingFor(c tele.Context) error {
	state := &UserState{
		TelegramID: c.Sender().ID,
		State:      "edit_looking_for",
		Data:       make(map[string]interface{}),
	}
	b.userCache.SetUserState(c.Sender().ID, state)

	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(btnLookingMale, btnLookingFemale),
		markup.Row(btnLookingAll),
	)

	return c.Edit("Кого ви шукаете?", markup)
}

func (b *Bot) handleEditPhotos(c tele.Context) error {
	state := &UserState{
		TelegramID: c.Sender().ID,
		State:      "edit_photos",
		Data:       make(map[string]interface{}),
	}
	b.userCache.SetUserState(c.Sender().ID, state)
	return c.Edit("Надішліть нові фотографії (до 5 штук). Напишіть 'готово' коли закінчите:")
}

// Text edit handlers
func (b *Bot) handleEditNameText(c tele.Context, state *UserState) error {
	name := strings.TrimSpace(c.Text())
	if len(name) < 2 || len(name) > 50 {
		return c.Send("Ім'я повинно містити від 2 до 50 символів. Спробуйте ще раз:")
	}

	err := b.updateUserField(c.Sender().ID, "name", name)
	if err != nil {
		return c.Send("Помилка оновлення. Спробуйте пізніше.")
	}

	b.userCache.DeleteUser(c.Sender().ID)
	b.userCache.DeleteUserState(c.Sender().ID)
	return c.Send("Ім'я оновлено! ✅")
}

func (b *Bot) handleEditAgeText(c tele.Context, state *UserState) error {
	age, err := strconv.Atoi(strings.TrimSpace(c.Text()))
	if err != nil || age < 18 || age > 99 {
		return c.Send("Будь ласка, введіть коректний вік (число від 18 до 99):")
	}

	err = b.updateUserField(c.Sender().ID, "age", age)
	if err != nil {
		return c.Send("Помилка оновлення. Спробуйте пізніше.")
	}

	b.userCache.DeleteUser(c.Sender().ID)
	b.userCache.DeleteUserState(c.Sender().ID)
	return c.Send("Вік оновлено! ✅")
}

func (b *Bot) handleEditDescriptionText(c tele.Context, state *UserState) error {
	description := strings.TrimSpace(c.Text())
	if len(description) > 300 {
		return c.Send("Опис занадто довгий. Максимум 300 символів. Спробуйте ще раз:")
	}

	err := b.updateUserField(c.Sender().ID, "description", description)
	if err != nil {
		return c.Send("Помилка оновлення. Спробуйте пізніше.")
	}

	b.userCache.DeleteUser(c.Sender().ID)
	b.userCache.DeleteUserState(c.Sender().ID)
	return c.Send("Опис оновлено! ✅")
}

func (b *Bot) handleEditCityText(c tele.Context, state *UserState) error {
	city := strings.TrimSpace(c.Text())
	if len(city) < 2 || len(city) > 50 {
		return c.Send("Назва міста повинна містити від 2 до 50 символів. Спробуйте ще раз:")
	}

	err := b.updateUserField(c.Sender().ID, "city", city)
	if err != nil {
		return c.Send("Помилка оновлення. Спробуйте пізніше.")
	}

	b.userCache.DeleteUser(c.Sender().ID)
	b.userCache.DeleteUserState(c.Sender().ID)
	return c.Send("Місто оновлено! ✅")
}

// Like/Dislike handlers
func (b *Bot) handleLike(c tele.Context) error {
	return b.handleLikeDislike(c, true)
}

func (b *Bot) handleDislike(c tele.Context) error {
	return b.handleLikeDislike(c, false)
}

func (b *Bot) handleLikeDislike(c tele.Context, isLike bool) error {
	// Check rate limiting
	action := "dislike"
	if isLike {
		action = "like"
	}

	allowed, err := b.rateLimiter.IsActionAllowed(c.Sender().ID, action)
	if err != nil {
		log.Printf("Rate limiter error: %v", err)
	} else if !allowed {
		return c.Respond(&tele.CallbackResponse{
			Text: "⚠️ Забагато дій. Спробуйте через хвилину.",
		})
	}

	// Extract candidate ID from callback data
	candidateID, err := strconv.ParseInt(c.Callback().Data, 10, 64)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Помилка обробки"})
	}

	// Save like/dislike
	err = b.saveLike(c.Sender().ID, candidateID, isLike)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Помилка збереження"})
	}

	// Check for match if it's a like
	if isLike {
		isMatch, err := b.checkMatch(c.Sender().ID, candidateID)
		if err == nil && isMatch {
			// Add match job to queue for async processing
			b.matchQueue.AddMatchJob(c.Sender().ID, candidateID)
		}
	}

	// Update daily stats
	if isLike {
		b.userCache.IncrementDailyStats("likes_sent")
	} else {
		b.userCache.IncrementDailyStats("dislikes_sent")
	}

	// Show next candidate
	user, err := b.getUser(c.Sender().ID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Помилка"})
	}

	nextCandidate, err := b.getNextCandidate(user.TelegramID)
	if err != nil {
		return c.Edit("Більше немає анкет для перегляду. Спробуйте пізніше!")
	}

	return b.editCandidate(c, nextCandidate)
}

func (b *Bot) handleBack(c tele.Context) error {
	return c.Respond(&tele.CallbackResponse{Text: "Функція повернення в розробці"})
}

func (b *Bot) handleStartChat(c tele.Context) error {
	return c.Respond(&tele.CallbackResponse{Text: "Чат функція в розробці"})
}

// Helper methods for candidate display
func (b *Bot) showCandidate(c tele.Context, candidate *User) error {
	text := b.formatCandidateProfile(candidate)

	markup := &tele.ReplyMarkup{}

	// Create buttons with candidate ID as data
	likeBtn := btnLike
	likeBtn.Data = strconv.FormatInt(candidate.TelegramID, 10)

	dislikeBtn := btnDislike
	dislikeBtn.Data = strconv.FormatInt(candidate.TelegramID, 10)

	markup.Inline(markup.Row(dislikeBtn, likeBtn))

	if len(candidate.Photos) > 0 {
		photo := &tele.Photo{File: tele.FromURL(candidate.Photos[0])}
		return c.Send(photo, text, markup)
	}

	return c.Send(text, markup)
}

func (b *Bot) editCandidate(c tele.Context, candidate *User) error {
	text := b.formatCandidateProfile(candidate)

	markup := &tele.ReplyMarkup{}

	// Create buttons with candidate ID as data
	likeBtn := btnLike
	likeBtn.Data = strconv.FormatInt(candidate.TelegramID, 10)

	dislikeBtn := btnDislike
	dislikeBtn.Data = strconv.FormatInt(candidate.TelegramID, 10)

	markup.Inline(markup.Row(dislikeBtn, likeBtn))

	if len(candidate.Photos) > 0 {
		photo := &tele.Photo{File: tele.FromURL(candidate.Photos[0])}
		return c.EditMedia(photo, text, markup)
	}

	return c.Edit(text, markup)
}

func (b *Bot) formatProfile(user *User) string {
	return fmt.Sprintf(`👤 Ваш профіль:

Ім'я: %s
Вік: %d років
Стать: %s
Місто: %s
Шукаю: %s
Опис: %s
Фото: %d шт.

Натисніть кнопку для редагування:`,
		user.Name, user.Age, user.Gender, user.City, user.LookingFor, user.Description, len(user.Photos))
}

func (b *Bot) formatCandidateProfile(user *User) string {
	return fmt.Sprintf("👤 %s, %d років\n📍 %s\n\n%s",
		user.Name, user.Age, user.City, user.Description)
}

func (b *Bot) updateUserPhotos(c tele.Context, state *UserState) error {
	photos := state.Data["photos"].([]string)

	// Get user ID
	user, err := b.getUser(c.Sender().ID)
	if err != nil {
		return c.Send("Помилка при оновленні фото")
	}

	// Delete old photos
	_, err = b.db.Exec("DELETE FROM user_photos WHERE user_id = ?", user.ID)
	if err != nil {
		return c.Send("Помилка при видаленні старих фото")
	}

	// Add new photos
	for _, photo := range photos {
		_, err := b.db.Exec("INSERT INTO user_photos (user_id, photo_url) VALUES (?, ?)", user.ID, photo)
		if err != nil {
			log.Printf("Error saving photo: %v", err)
		}
	}

	// Clear cache and state
	b.userCache.DeleteUser(c.Sender().ID)
	b.userCache.DeleteUserState(c.Sender().ID)

	return c.Send(fmt.Sprintf("Фото оновлено! ✅ Збережено %d фото.", len(photos)))
}
