package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type UserCache struct {
	redis *redis.Client
	ctx   context.Context
}

func NewUserCache(redisClient *redis.Client) *UserCache {
	return &UserCache{
		redis: redisClient,
		ctx:   context.Background(),
	}
}

// Cache user profile
func (uc *UserCache) SetUser(user *User) error {
	key := fmt.Sprintf("user:%d", user.TelegramID)
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	return uc.redis.Set(uc.ctx, key, data, time.Hour).Err()
}

// Get user from cache
func (uc *UserCache) GetUser(telegramID int64) (*User, error) {
	key := fmt.Sprintf("user:%d", telegramID)
	data, err := uc.redis.Get(uc.ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var user User
	err = json.Unmarshal([]byte(data), &user)
	return &user, err
}

// Delete user from cache
func (uc *UserCache) DeleteUser(telegramID int64) error {
	key := fmt.Sprintf("user:%d", telegramID)
	return uc.redis.Del(uc.ctx, key).Err()
}

// Cache user state
func (uc *UserCache) SetUserState(telegramID int64, state *UserState) error {
	key := fmt.Sprintf("state:%d", telegramID)
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return uc.redis.Set(uc.ctx, key, data, time.Hour*24).Err()
}

// Get user state from cache
func (uc *UserCache) GetUserState(telegramID int64) (*UserState, error) {
	key := fmt.Sprintf("state:%d", telegramID)
	data, err := uc.redis.Get(uc.ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var state UserState
	err = json.Unmarshal([]byte(data), &state)
	return &state, err
}

// Delete user state
func (uc *UserCache) DeleteUserState(telegramID int64) error {
	key := fmt.Sprintf("state:%d", telegramID)
	return uc.redis.Del(uc.ctx, key).Err()
}

// Cache candidate list for user
func (uc *UserCache) SetCandidates(userID int64, candidates []int64) error {
	key := fmt.Sprintf("candidates:%d", userID)
	data, err := json.Marshal(candidates)
	if err != nil {
		return err
	}

	return uc.redis.Set(uc.ctx, key, data, time.Hour).Err()
}

// Get candidates from cache
func (uc *UserCache) GetCandidates(userID int64) ([]int64, error) {
	key := fmt.Sprintf("candidates:%d", userID)
	data, err := uc.redis.Get(uc.ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var candidates []int64
	err = json.Unmarshal([]byte(data), &candidates)
	return candidates, err
}

// Cache match count
func (uc *UserCache) IncrementMatchCount(userID int64) error {
	key := fmt.Sprintf("matches_count:%d", userID)
	return uc.redis.Incr(uc.ctx, key).Err()
}

// Get match count
func (uc *UserCache) GetMatchCount(userID int64) (int64, error) {
	key := fmt.Sprintf("matches_count:%d", userID)
	return uc.redis.Get(uc.ctx, key).Int64()
}

// Cache daily stats
func (uc *UserCache) IncrementDailyStats(metric string) error {
	today := time.Now().Format("2006-01-02")
	key := fmt.Sprintf("stats:%s:%s", metric, today)

	pipe := uc.redis.Pipeline()
	pipe.Incr(uc.ctx, key)
	pipe.Expire(uc.ctx, key, time.Hour*48) // Keep for 2 days

	_, err := pipe.Exec(uc.ctx)
	return err
}

// Get daily stats
func (uc *UserCache) GetDailyStats(metric string, date time.Time) (int64, error) {
	key := fmt.Sprintf("stats:%s:%s", metric, date.Format("2006-01-02"))
	return uc.redis.Get(uc.ctx, key).Int64()
}

// Cache recent activity
func (uc *UserCache) SetUserActivity(userID int64) error {
	key := fmt.Sprintf("activity:%d", userID)
	return uc.redis.Set(uc.ctx, key, time.Now().Unix(), time.Hour*24).Err()
}

// Check if user is recently active
func (uc *UserCache) IsUserActive(userID int64, threshold time.Duration) (bool, error) {
	key := fmt.Sprintf("activity:%d", userID)
	timestamp, err := uc.redis.Get(uc.ctx, key).Int64()
	if err != nil {
		return false, err
	}

	lastActivity := time.Unix(timestamp, 0)
	return time.Since(lastActivity) < threshold, nil
}

// Track cache hit
func (uc *UserCache) TrackCacheHit() {
	uc.redis.Incr(uc.ctx, "cache_hits")
	uc.redis.Expire(uc.ctx, "cache_hits", time.Hour)
}

// Track cache miss
func (uc *UserCache) TrackCacheMiss() {
	uc.redis.Incr(uc.ctx, "cache_misses")
	uc.redis.Expire(uc.ctx, "cache_misses", time.Hour)
}
