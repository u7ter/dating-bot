package main

import (
	"context"
	"encoding/json"
	"fmt"
	tele "gopkg.in/telebot.v3"
	"log"
	"runtime"
	"time"

	"github.com/go-redis/redis/v8"
)

type Metrics struct {
	redis *redis.Client
	ctx   context.Context
}

type SystemMetrics struct {
	Timestamp      time.Time `json:"timestamp"`
	CPUUsage       float64   `json:"cpu_usage"`
	MemoryUsage    float64   `json:"memory_usage"`
	GoroutineCount int       `json:"goroutine_count"`
	RequestsPerSec int64     `json:"requests_per_sec"`
	ActiveUsers    int64     `json:"active_users"`
	QueueLength    int64     `json:"queue_length"`
	CacheHitRatio  float64   `json:"cache_hit_ratio"`
}

func NewMetrics(redisClient *redis.Client) *Metrics {
	return &Metrics{
		redis: redisClient,
		ctx:   context.Background(),
	}
}

func (b *Bot) startMetricsCollection() {
	metrics := NewMetrics(b.redis)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
			metrics.collectAndStore()
		}
	}
}

func (m *Metrics) collectAndStore() {
	metrics := m.collectSystemMetrics()

	// Store current metrics
	data, err := json.Marshal(metrics)
	if err != nil {
		log.Printf("Failed to marshal metrics: %v", err)
		return
	}

	// Store in time series
	timestamp := time.Now().Unix()
	key := fmt.Sprintf("metrics:%d", timestamp)

	pipe := m.redis.Pipeline()
	pipe.Set(m.ctx, key, data, time.Hour*24)
	pipe.ZAdd(m.ctx, "metrics_timeline", &redis.Z{
		Score:  float64(timestamp),
		Member: key,
	})

	// Update real-time metrics
	pipe.Set(m.ctx, "system:cpu_usage", metrics.CPUUsage, time.Minute*5)
	pipe.Set(m.ctx, "system:memory_usage", metrics.MemoryUsage, time.Minute*5)
	pipe.Set(m.ctx, "system:goroutines", metrics.GoroutineCount, time.Minute*5)

	_, err = pipe.Exec(m.ctx)
	if err != nil {
		log.Printf("Failed to store metrics: %v", err)
	}
}

func (m *Metrics) collectSystemMetrics() SystemMetrics {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Calculate memory usage percentage
	memoryUsage := float64(memStats.Alloc) / (1024 * 1024 * 1024) * 100

	return SystemMetrics{
		Timestamp:      time.Now(),
		CPUUsage:       m.getCPUUsage(),
		MemoryUsage:    memoryUsage,
		GoroutineCount: runtime.NumGoroutine(),
		RequestsPerSec: 0, // Simplified for now
		ActiveUsers:    0, // Simplified for now
		QueueLength:    0, // Simplified for now
		CacheHitRatio:  0, // Simplified for now
	}
}

func (m *Metrics) getCPUUsage() float64 {
	// Simple CPU usage estimation
	goroutines := runtime.NumGoroutine()
	if goroutines > 1000 {
		return 90.0
	} else if goroutines > 500 {
		return 70.0
	} else if goroutines > 100 {
		return 50.0
	}
	return 20.0
}

// Performance monitoring middleware
func (b *Bot) PerformanceMiddleware() tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			start := time.Now()

			// Execute handler
			err := next(c)

			// Track performance
			duration := time.Since(start)

			// Log slow requests
			if duration > time.Second {
				log.Printf("Slow request: %s took %v", c.Message().Text, duration)
			}

			return err
		}
	}
}
