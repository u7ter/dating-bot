package main

import (
	"fmt"
	tele "gopkg.in/telebot.v3"
	"log"
)

// Database operations for users
func (b *Bot) getNextCandidate(userID int64) (*User, error) {
	currentUser, err := b.getUser(userID)
	if err != nil {
		return nil, err
	}

	// Build query based on user preferences
	var genderFilter string
	switch currentUser.LookingFor {
	case "чоловік":
		genderFilter = "AND u.gender = 'чоловік'"
	case "жінка":
		genderFilter = "AND u.gender = 'жінка'"
	default:
		genderFilter = ""
	}

	query := fmt.Sprintf(`
		SELECT u.id, u.telegram_id, u.name, u.age, u.gender, u.description, u.city, u.looking_for, u.created_at, u.is_active
		FROM users u
		WHERE u.telegram_id != ? 
		AND u.is_active = true
		%s
		AND u.telegram_id NOT IN (
			SELECT liked_id FROM likes WHERE liker_id = ?
		)
		AND u.telegram_id NOT IN (
			SELECT blocked_id FROM user_blocks WHERE blocker_id = ?
		)
		ORDER BY RAND()
		LIMIT 1
	`, genderFilter)

	user := &User{}
	err = b.db.QueryRow(query, userID, userID, userID).Scan(
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

func (b *Bot) saveLike(likerID, likedID int64, isLike bool) error {
	query := `INSERT INTO likes (liker_id, liked_id, is_like) VALUES (?, ?, ?) 
			  ON DUPLICATE KEY UPDATE is_like = VALUES(is_like)`

	_, err := b.db.Exec(query, likerID, likedID, isLike)
	return err
}

func (b *Bot) checkMatch(user1ID, user2ID int64) (bool, error) {
	query := `SELECT COUNT(*) FROM likes 
			  WHERE liker_id = ? AND liked_id = ? AND is_like = true`

	var count int
	err := b.db.QueryRow(query, user2ID, user1ID).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (b *Bot) createMatch(user1ID, user2ID int64) error {
	// Ensure user1ID is always smaller for consistency
	if user1ID > user2ID {
		user1ID, user2ID = user2ID, user1ID
	}

	query := `INSERT INTO matches (user1_id, user2_id, is_active) VALUES (?, ?, true)
			  ON DUPLICATE KEY UPDATE is_active = true`

	_, err := b.db.Exec(query, user1ID, user2ID)
	return err
}

func (b *Bot) getUserMatches(userID int64) ([]*User, error) {
	query := `
		SELECT u.id, u.telegram_id, u.name, u.age, u.gender, u.description, u.city, u.looking_for, u.created_at, u.is_active
		FROM users u
		INNER JOIN matches m ON (
			(m.user1_id = ? AND m.user2_id = u.telegram_id) OR 
			(m.user2_id = ? AND m.user1_id = u.telegram_id)
		)
		WHERE m.is_active = true
		ORDER BY m.created_at DESC
	`

	rows, err := b.db.Query(query, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(
			&user.ID, &user.TelegramID, &user.Name, &user.Age, &user.Gender,
			&user.Description, &user.City, &user.LookingFor, &user.CreatedAt, &user.IsActive,
		)
		if err != nil {
			log.Printf("Error scanning match: %v", err)
			continue
		}

		// Get photos
		photos, err := b.getUserPhotos(user.ID)
		if err == nil {
			user.Photos = photos
		}

		matches = append(matches, user)
	}

	return matches, nil
}

func (b *Bot) updateUserField(telegramID int64, field string, value interface{}) error {
	query := fmt.Sprintf("UPDATE users SET %s = ?, updated_at = CURRENT_TIMESTAMP WHERE telegram_id = ?", field)
	_, err := b.db.Exec(query, value, telegramID)
	return err
}

func (b *Bot) deleteUser(telegramID int64) error {
	// Start transaction
	tx, err := b.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete user photos
	_, err = tx.Exec("DELETE FROM user_photos WHERE user_id = (SELECT id FROM users WHERE telegram_id = ?)", telegramID)
	if err != nil {
		return err
	}

	// Delete likes
	_, err = tx.Exec("DELETE FROM likes WHERE liker_id = ? OR liked_id = ?", telegramID, telegramID)
	if err != nil {
		return err
	}

	// Delete matches
	_, err = tx.Exec("DELETE FROM matches WHERE user1_id = ? OR user2_id = ?", telegramID, telegramID)
	if err != nil {
		return err
	}

	// Delete chat messages
	_, err = tx.Exec(`DELETE FROM chat_messages WHERE sender_id = ? OR match_id IN (
		SELECT id FROM matches WHERE user1_id = ? OR user2_id = ?
	)`, telegramID, telegramID, telegramID)
	if err != nil {
		return err
	}

	// Delete user
	_, err = tx.Exec("DELETE FROM users WHERE telegram_id = ?", telegramID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (b *Bot) notifyMatch(user1ID, user2ID int64) {
	// Get user names
	user1, err := b.getUser(user1ID)
	if err != nil {
		return
	}

	user2, err := b.getUser(user2ID)
	if err != nil {
		return
	}

	// Create match notification markup
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(btnStartChat))

	// Notify first user
	message1 := fmt.Sprintf("🎉 У вас новий збіг!\n\n👤 %s, %d років з %s\n\nВи сподобались одне одному!",
		user2.Name, user2.Age, user2.City)

	if len(user2.Photos) > 0 {
		photo := &tele.Photo{File: tele.FromURL(user2.Photos[0])}
		b.Send(&tele.User{ID: user1ID}, photo, message1, markup)
	} else {
		b.Send(&tele.User{ID: user1ID}, message1, markup)
	}

	// Notify second user
	message2 := fmt.Sprintf("🎉 У вас новий збіг!\n\n👤 %s, %d років з %s\n\nВи сподобались одне одному!",
		user1.Name, user1.Age, user1.City)

	if len(user1.Photos) > 0 {
		photo := &tele.Photo{File: tele.FromURL(user1.Photos[0])}
		b.Send(&tele.User{ID: user2ID}, photo, message2, markup)
	} else {
		b.Send(&tele.User{ID: user2ID}, message2, markup)
	}
}

func (b *Bot) blockUser(blockerID, blockedID int64) error {
	query := `INSERT INTO user_blocks (blocker_id, blocked_id) VALUES (?, ?)
			  ON DUPLICATE KEY UPDATE created_at = CURRENT_TIMESTAMP`

	_, err := b.db.Exec(query, blockerID, blockedID)
	return err
}

func (b *Bot) reportUser(reporterID, reportedID int64, reason string) error {
	query := `INSERT INTO user_reports (reporter_id, reported_id, reason) VALUES (?, ?, ?)`
	_, err := b.db.Exec(query, reporterID, reportedID, reason)
	return err
}
