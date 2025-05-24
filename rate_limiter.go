package main

import (
	"context"
	"fmt"
	"log"
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
