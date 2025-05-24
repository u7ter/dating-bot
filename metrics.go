package main

import (
	"context"
	"encoding/json"
	"fmt"
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

	// Keep only last 24 hours of metrics
	cutoff := time.Now().Add(-24 * time.Hour).Unix()
	pipe.ZRemRangeByScore(m.ctx, "metrics_timeline", "0", fmt.Sprintf("%d", cutoff))

	// Update real-time metrics
	pipe.Set(m.ctx, "system:cpu_usage", metrics.CPUUsage, time.Minute*5)
	pipe.Set(m.ctx, "system:memory_usage", metrics.MemoryUsage, time.Minute*5)
	pipe.Set(m.ctx, "system:goroutines", metrics.GoroutineCount, time.Minute*5)
	pipe.Set(m.ctx, "system:requests_per_sec", metrics.RequestsPerSec, time.Minute*5)
	pipe.Set(m.ctx, "system:active_users", metrics.ActiveUsers, time.Minute*5)

	_, err = pipe.Exec(m.ctx)
	if err != nil {
		log.Printf("Failed to store metrics: %v", err)
	}
}

func (m *Metrics) collectSystemMetrics() SystemMetrics {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Calculate memory usage percentage (assuming 1GB max)
	memoryUsage := float64(memStats.Alloc) / (1024 * 1024 * 1024) * 100

	// Get queue lengths
	matchQueueLen, _ := m.redis.LLen(m.ctx, "match_queue").Result()
	notificationQueueLen, _ := m.redis.LLen(m.ctx, "notification_queue").Result()
	totalQueueLen := matchQueueLen + notificationQueueLen

	// Calculate requests per second
	requestsPerSec := m.getRequestsPerSecond()

	// Get active users count
	activeUsers := m.getActiveUsersCount()

	// Calculate cache hit ratio
	cacheHitRatio := m.getCacheHitRatio()

	return SystemMetrics{
		Timestamp:      time.Now(),
		CPUUsage:       m.getCPUUsage(),
		MemoryUsage:    memoryUsage,
		GoroutineCount: runtime.NumGoroutine(),
		RequestsPerSec: requestsPerSec,
		ActiveUsers:    activeUsers,
		QueueLength:    totalQueueLen,
		CacheHitRatio:  cacheHitRatio,
	}
}

func (m *Metrics) getCPUUsage() float64 {
	// Simple CPU usage estimation based on goroutine count
	goroutines := runtime.NumGoroutine()

	// Normalize to percentage (rough estimation)
	if goroutines > 1000 {
		return 90.0
	} else if goroutines > 500 {
		return 70.0
	} else if goroutines > 100 {
		return 50.0
	}
	return 20.0
}

func (m *Metrics) getRequestsPerSecond() int64 {
	// Get request count from last minute
	now := time.Now()
	oneMinuteAgo := now.Add(-time.Minute)

	count, err := m.redis.ZCount(m.ctx, "request_timeline",
		fmt.Sprintf("%d", oneMinuteAgo.UnixNano()),
		fmt.Sprintf("%d", now.UnixNano())).Result()

	if err != nil {
		return 0
	}

	return count / 60 // requests per second
}

func (m *Metrics) getActiveUsersCount() int64 {
	// Count users active in last 5 minutes
	cutoff := time.Now().Add(-5 * time.Minute).Unix()

	pattern := "activity:*"
	keys, err := m.redis.Keys(m.ctx, pattern).Result()
	if err != nil {
		return 0
	}

	var activeCount int64
	for _, key := range keys {
		timestamp, err := m.redis.Get(m.ctx, key).Int64()
		if err == nil && timestamp > cutoff {
			activeCount++
		}
	}

	return activeCount
}

func (m *Metrics) getCacheHitRatio() float64 {
	hits, _ := m.redis.Get(m.ctx, "cache_hits").Int64()
	misses, _ := m.redis.Get(m.ctx, "cache_misses").Int64()

	total := hits + misses
	if total == 0 {
		return 0
	}

	return float64(hits) / float64(total) * 100
}

// Track request
func (m *Metrics) TrackRequest() {
	now := time.Now()
	m.redis.ZAdd(m.ctx, "request_timeline", &redis.Z{
		Score:  float64(now.UnixNano()),
		Member: now.UnixNano(),
	})

	// Clean old entries (keep last hour)
	cutoff := now.Add(-time.Hour).UnixNano()
	m.redis.ZRemRangeByScore(m.ctx, "request_timeline", "0", fmt.Sprintf("%d", cutoff))
}

// Track cache hit
func (m *Metrics) TrackCacheHit() {
	m.redis.Incr(m.ctx, "cache_hits")
	m.redis.Expire(m.ctx, "cache_hits", time.Hour)
}

// Track cache miss
func (m *Metrics) TrackCacheMiss() {
	m.redis.Incr(m.ctx, "cache_misses")
	m.redis.Expire(m.ctx, "cache_misses", time.Hour)
}

// Get metrics for monitoring dashboard
func (m *Metrics) GetMetricsHistory(hours int) ([]SystemMetrics, error) {
	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour).Unix()

	keys, err := m.redis.ZRangeByScore(m.ctx, "metrics_timeline", &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", cutoff),
		Max: "+inf",
	}).Result()

	if err != nil {
		return nil, err
	}

	var metrics []SystemMetrics
	for _, key := range keys {
		data, err := m.redis.Get(m.ctx, key).Result()
		if err != nil {
			continue
		}

		var metric SystemMetrics
		if err := json.Unmarshal([]byte(data), &metric); err == nil {
			metrics = append(metrics, metric)
		}
	}

	return metrics, nil
}

// Alert system
func (m *Metrics) CheckAlerts() {
	metrics := m.collectSystemMetrics()

	// High CPU usage alert
	if metrics.CPUUsage > 80 {
		m.sendAlert("HIGH_CPU", fmt.Sprintf("CPU usage: %.2f%%", metrics.CPUUsage))
	}

	// High memory usage alert
	if metrics.MemoryUsage > 80 {
		m.sendAlert("HIGH_MEMORY", fmt.Sprintf("Memory usage: %.2f%%", metrics.MemoryUsage))
	}

	// High queue length alert
	if metrics.QueueLength > 1000 {
		m.sendAlert("HIGH_QUEUE", fmt.Sprintf("Queue length: %d", metrics.QueueLength))
	}

	// Low cache hit ratio alert
	if metrics.CacheHitRatio < 50 {
		m.sendAlert("LOW_CACHE_HIT", fmt.Sprintf("Cache hit ratio: %.2f%%", metrics.CacheHitRatio))
	}
}

func (m *Metrics) sendAlert(alertType, message string) {
	alert := map[string]interface{}{
		"type":      alertType,
		"message":   message,
		"timestamp": time.Now(),
	}

	data, _ := json.Marshal(alert)
	m.redis.LPush(m.ctx, "alerts", data)

	log.Printf("ALERT [%s]: %s", alertType, message)
}

// Performance monitoring middleware
func (b *Bot) PerformanceMiddleware() tele.MiddlewareFunc {
	metrics := NewMetrics(b.redis)

	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			start := time.Now()

			// Track request
			metrics.TrackRequest()

			// Execute handler
			err := next(c)

			// Track performance
			duration := time.Since(start)

			// Store response time
			key := fmt.Sprintf("response_time:%s", c.Message().Text)
			b.redis.LPush(b.ctx, key, duration.Milliseconds())
			b.redis.LTrim(b.ctx, key, 0, 99) // Keep last 100 measurements
			b.redis.Expire(b.ctx, key, time.Hour)

			// Log slow requests
			if duration > time.Second {
				log.Printf("Slow request: %s took %v", c.Message().Text, duration)
			}

			return err
		}
	}
}
