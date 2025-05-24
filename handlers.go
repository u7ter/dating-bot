package main

import (
	"fmt"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v3"
)

// Registration handlers
func (b *Bot) handleRegistrationName(c tele.Context, state *UserState) error {
	name := strings.TrimSpace(c.Text())
	if len(name) < 2 || len(name) > 50 {
		return c.Send("Ім'я повинно містити від 2 до 50 символів. Спробуйте ще раз:")
	}

	state.Data["name"] = name
	state.State = "registration_age"
	return c.Send("Чудово! Скільки вам років? (введіть число від 18 до 99)")
}

func (b *Bot) handleRegistrationAge(c tele.Context, state *UserState) error {
	age, err := strconv.Atoi(strings.TrimSpace(c.Text()))
	if err != nil || age < 18 || age > 99 {
		return c.Send("Будь ласка, введіть коректний вік (число від 18 до 99):")
	}

	state.Data["age"] = age
	state.State = "registration_gender"

	markup := &tele.ReplyMarkup{}
	btnMale := markup.Data("👨 Чоловік", "gender", "чоловік")
	btnFemale := markup.Data("👩 Жінка", "gender", "жінка")
	markup.Inline(markup.Row(btnMale, btnFemale))

	return c.Send("Оберіть вашу стать:", markup)
}

func (b *Bot) handleRegistrationGender(c tele.Context, state *UserState) error {
	gender := strings.TrimSpace(c.Text())
	if gender != "чоловік" && gender != "жінка" {
		return c.Send("Будь ласка, оберіть стать за допомогою кнопок вище")
	}

	state.Data["gender"] = gender
	state.State = "registration_description"
	return c.Send("Розкажіть трохи про себе (до 300 символів):")
}

func (b *Bot) handleRegistrationDescription(c tele.Context, state *UserState) error {
	description := strings.TrimSpace(c.Text())
	if len(description) > 300 {
		return c.Send("Опис занадто довгий. Максимум 300 символів. Спробуйте ще раз:")
	}

	state.Data["description"] = description
	state.State = "registration_city"
	return c.Send("В якому місті ви живете?")
}

func (b *Bot) handleRegistrationCity(c tele.Context, state *UserState) error {
	city := strings.TrimSpace(c.Text())
	if len(city) < 2 || len(city) > 50 {
		return c.Send("Назва міста повинна містити від 2 до 50 символів. Спробуйте ще раз:")
	}

	state.Data["city"] = city
	state.State = "registration_looking_for"

	markup := &tele.ReplyMarkup{}
	btnMale := markup.Data("👨 Чоловіків", "looking_for", "чоловік")
	btnFemale := markup.Data("👩 Жінок", "looking_for", "жінка")
	btnAll := markup.Data("👥 Усіх", "looking_for", "усі")
	markup.Inline(markup.Row(btnMale, btnFemale), markup.Row(btnAll))

	return c.Send("Кого ви шукаете?", markup)
}

func (b *Bot) handleRegistrationLookingFor(c tele.Context, state *UserState) error {
	lookingFor := strings.TrimSpace(c.Text())
	if lookingFor != "чоловік" && lookingFor != "жінка" && lookingFor != "усі" {
		return c.Send("Будь ласка, оберіть варіант за допомогою кнопок вище")
	}

	state.Data["looking_for"] = lookingFor
	state.State = "registration_photos"
	return c.Send("Тепер надішліть від 1 до 5 фотографій. Коли закінчите, напишіть 'готово'")
}

func (b *Bot) handlePhotoUpload(c tele.Context, state *UserState) error {
	if state.Data["photos"] == nil {
		state.Data["photos"] = []string{}
	}

	photos := state.Data["photos"].([]string)
	if len(photos) >= 5 {
		return c.Send("Максимум 5 фотографій. Напишіть 'готово' для завершення")
	}

	// In a real implementation, you would save the photo to a file server
	// For now, we'll use the file ID as a placeholder
	photoID := c.Message().Photo.FileID
	photos = append(photos, photoID)
	state.Data["photos"] = photos

	return c.Send(fmt.Sprintf("Фото збережено (%d/5). Надішліть ще фото або напишіть 'готово'", len(photos)))
}

// Like/Dislike handlers
func (b *Bot) handleLike(c tele.Context) error {
	return b.handleLikeDislike(c, true)
}

func (b *Bot) handleDislike(c tele.Context) error {
	return b.handleLikeDislike(c, false)
}

func (b *Bot) handleLikeDislike(c tele.Context, isLike bool) error {
	// Extract candidate ID from callback data
	data := c.Callback().Data
	candidateID, err := strconv.ParseInt(data, 10, 64)
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
			// Create match
			err = b.createMatch(c.Sender().ID, candidateID)
			if err == nil {
				// Notify both users
				b.notifyMatch(c.Sender().ID, candidateID)
			}
		}
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
	// Implementation for going back to previous candidate
	return c.Respond(&tele.CallbackResponse{Text: "Функція повернення в розробці"})
}

func (b *Bot) handleStartChat(c tele.Context) error {
	return c.Respond(&tele.CallbackResponse{Text: "Чат функція в розробці"})
}

// Edit handlers
func (b *Bot) handleEditName(c tele.Context) error {
	userStates[c.Sender().ID] = &UserState{
		TelegramID: c.Sender().ID,
		State:      "edit_name",
		Data:       make(map[string]interface{}),
	}
	return c.Edit("Введіть нове ім'я:")
}

func (b *Bot) handleEditAge(c tele.Context) error {
	userStates[c.Sender().ID] = &UserState{
		TelegramID: c.Sender().ID,
		State:      "edit_age",
		Data:       make(map[string]interface{}),
	}
	return c.Edit("Введіть новий вік:")
}

func (b *Bot) handleEditGender(c tele.Context) error {
	markup := &tele.ReplyMarkup{}
	btnMale := markup.Data("👨 Чоловік", "update_gender", "чоловік")
	btnFemale := markup.Data("👩 Жінка", "update_gender", "жінка")
	markup.Inline(markup.Row(btnMale, btnFemale))

	return c.Edit("Оберіть стать:", markup)
}

func (b *Bot) handleEditDescription(c tele.Context) error {
	userStates[c.Sender().ID] = &UserState{
		TelegramID: c.Sender().ID,
		State:      "edit_description",
		Data:       make(map[string]interface{}),
	}
	return c.Edit("Введіть новий опис:")
}

func (b *Bot) handleEditCity(c tele.Context) error {
	userStates[c.Sender().ID] = &UserState{
		TelegramID: c.Sender().ID,
		State:      "edit_city",
		Data:       make(map[string]interface{}),
	}
	return c.Edit("Введіть нове місто:")
}

func (b *Bot) handleEditLookingFor(c tele.Context) error {
	markup := &tele.ReplyMarkup{}
	btnMale := markup.Data("👨 Чоловіків", "update_looking_for", "чоловік")
	btnFemale := markup.Data("👩 Жінок", "update_looking_for", "жінка")
	btnAll := markup.Data("👥 Усіх", "update_looking_for", "усі")
	markup.Inline(markup.Row(btnMale, btnFemale), markup.Row(btnAll))

	return c.Edit("Кого ви шукаете?", markup)
}

func (b *Bot) handleEditPhotos(c tele.Context) error {
	userStates[c.Sender().ID] = &UserState{
		TelegramID: c.Sender().ID,
		State:      "edit_photos",
		Data:       make(map[string]interface{}),
	}
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

	delete(userStates, c.Sender().ID)
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

	delete(userStates, c.Sender().ID)
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

	delete(userStates, c.Sender().ID)
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

	delete(userStates, c.Sender().ID)
	return c.Send("Місто оновлено! ✅")
}

// Helper methods for candidate display
func (b *Bot) showCandidate(c tele.Context, candidate *User) error {
	text := b.formatCandidateProfile(candidate)

	markup := &tele.ReplyMarkup{}
	btnLike.Data = strconv.FormatInt(candidate.TelegramID, 10)
	btnDislike.Data = strconv.FormatInt(candidate.TelegramID, 10)
	markup.Inline(markup.Row(btnDislike, btnLike))

	if len(candidate.Photos) > 0 {
		photo := &tele.Photo{File: tele.FromURL(candidate.Photos[0])}
		return c.Send(photo, text, markup)
	}

	return c.Send(text, markup)
}

func (b *Bot) editCandidate(c tele.Context, candidate *User) error {
	text := b.formatCandidateProfile(candidate)

	markup := &tele.ReplyMarkup{}
	btnLike.Data = strconv.FormatInt(candidate.TelegramID, 10)
	btnDislike.Data = strconv.FormatInt(candidate.TelegramID, 10)
	markup.Inline(markup.Row(btnDislike, btnLike))

	if len(candidate.Photos) > 0 {
		photo := &tele.Photo{File: tele.FromURL(candidate.Photos[0])}
		return c.EditMedia(photo, text, markup)
	}

	return c.Edit(text, markup)
}

func (b *Bot) formatCandidateProfile(user *User) string {
	return fmt.Sprintf("👤 %s, %d років\n📍 %s\n\n%s",
		user.Name, user.Age, user.City, user.Description)
}
