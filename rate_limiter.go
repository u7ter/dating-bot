package main

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	tele "gopkg.in/telebot.v3"
)

type RateLimiter struct {
	redis      *redis.Client
	rateLimit  int
	burstLimit int
	ctx        context.Context
}

func NewRateLimiter(redisClient *redis.Client, rateLimit, burstLimit int) *RateLimiter {
	return &RateLimiter{
		redis:      redisClient,
		rateLimit:  rateLimit,
		burstLimit: burstLimit,
		ctx:        context.Background(),
	}
}

// Check if user is rate limited
func (rl *RateLimiter) IsAllowed(userID int64) (bool, error) {
	key := fmt.Sprintf("rate_limit:%d", userID)

	// Use sliding window rate limiting
	now := time.Now()
	window := time.Minute

	pipe := rl.redis.Pipeline()

	// Remove old entries
	pipe.ZRemRangeByScore(rl.ctx, key, "0", fmt.Sprintf("%d", now.Add(-window).UnixNano()))

	// Count current requests
	pipe.ZCard(rl.ctx, key)

	// Add current request
	pipe.ZAdd(rl.ctx, key, &redis.Z{
		Score:  float64(now.UnixNano()),
		Member: now.UnixNano(),
	})

	// Set expiration
	pipe.Expire(rl.ctx, key, window)

	results, err := pipe.Exec(rl.ctx)
	if err != nil {
		return false, err
	}

	count := results[1].(*redis.IntCmd).Val()
	return count < int64(rl.rateLimit), nil
}

// Rate limiting middleware
func (b *Bot) RateLimitMiddleware() tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			userID := c.Sender().ID

			allowed, err := b.rateLimiter.IsAllowed(userID)
			if err != nil {
				log.Printf("Rate limiter error: %v", err)
				return next(c) // Allow on error
			}

			if !allowed {
				return c.Send("⚠️ Забагато запитів. Спробуйте через хвилину.")
			}

			return next(c)
		}
	}
}

// Advanced rate limiting with different limits for different actions
func (rl *RateLimiter) IsActionAllowed(userID int64, action string) (bool, error) {
	limits := map[string]int{
		"like":    50,  // 50 likes per hour
		"message": 100, // 100 messages per hour
		"search":  200, // 200 searches per hour
		"profile": 20,  // 20 profile views per hour
	}

	limit, exists := limits[action]
	if !exists {
		limit = rl.rateLimit
	}

	key := fmt.Sprintf("rate_limit:%s:%d", action, userID)

	now := time.Now()
	window := time.Hour

	pipe := rl.redis.Pipeline()
	pipe.ZRemRangeByScore(rl.ctx, key, "0", fmt.Sprintf("%d", now.Add(-window).UnixNano()))
	pipe.ZCard(rl.ctx, key)
	pipe.ZAdd(rl.ctx, key, &redis.Z{
		Score:  float64(now.UnixNano()),
		Member: now.UnixNano(),
	})
	pipe.Expire(rl.ctx, key, window)

	results, err := pipe.Exec(rl.ctx)
	if err != nil {
		return false, err
	}

	count := results[1].(*redis.IntCmd).Val()
	return count < int64(limit), nil
}

// IP-based rate limiting for additional protection
func (rl *RateLimiter) IsIPAllowed(ip string) (bool, error) {
	key := fmt.Sprintf("ip_rate_limit:%s", ip)

	count, err := rl.redis.Incr(rl.ctx, key).Result()
	if err != nil {
		return false, err
	}

	if count == 1 {
		rl.redis.Expire(rl.ctx, key, time.Minute)
	}

	return count <= int64(rl.burstLimit), nil
}

// Global rate limiting
func (rl *RateLimiter) IsGlobalAllowed() (bool, error) {
	key := "global_rate_limit"

	count, err := rl.redis.Incr(rl.ctx, key).Result()
	if err != nil {
		return false, err
	}

	if count == 1 {
		rl.redis.Expire(rl.ctx, key, time.Second)
	}

	// Allow 1000 requests per second globally
	return count <= 1000, nil
}

// Adaptive rate limiting based on system load
func (rl *RateLimiter) GetAdaptiveLimit(baseLimit int) int {
	// Get system metrics from Redis
	cpuUsage, _ := rl.redis.Get(rl.ctx, "system:cpu_usage").Float64()
	memUsage, _ := rl.redis.Get(rl.ctx, "system:memory_usage").Float64()

	// Reduce limits if system is under high load
	if cpuUsage > 80 || memUsage > 80 {
		return baseLimit / 2
	} else if cpuUsage > 60 || memUsage > 60 {
		return int(float64(baseLimit) * 0.75)
	}

	return baseLimit
}

// Whitelist certain users (premium users, admins)
func (rl *RateLimiter) IsWhitelisted(userID int64) (bool, error) {
	key := "whitelist_users"
	return rl.redis.SIsMember(rl.ctx, key, userID).Result()
}

// Add user to whitelist
func (rl *RateLimiter) AddToWhitelist(userID int64) error {
	key := "whitelist_users"
	return rl.redis.SAdd(rl.ctx, key, userID).Err()
}

// Temporary ban for abusive users
func (rl *RateLimiter) BanUser(userID int64, duration time.Duration) error {
	key := fmt.Sprintf("banned_user:%d", userID)
	return rl.redis.Set(rl.ctx, key, "banned", duration).Err()
}

// Check if user is banned
func (rl *RateLimiter) IsBanned(userID int64) (bool, error) {
	key := fmt.Sprintf("banned_user:%d", userID)
	_, err := rl.redis.Get(rl.ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	return err == nil, err
}
