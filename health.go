package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
	Metrics   map[string]int64  `json:"metrics"`
}

func (b *Bot) startHealthServer() {
	http.HandleFunc("/health", b.healthHandler)
	http.HandleFunc("/metrics", b.metricsHandler)

	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	log.Println("Health server starting on :8080")
	if err := server.ListenAndServe(); err != nil {
		log.Printf("Health server error: %v", err)
	}
}

func (b *Bot) healthHandler(w http.ResponseWriter, r *http.Request) {
	status := b.checkHealth()

	w.Header().Set("Content-Type", "application/json")

	if status.Status == "healthy" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(status)
}

func (b *Bot) checkHealth() HealthStatus {
	services := make(map[string]string)
	metrics := make(map[string]int64)

	// Check database
	if err := b.db.Ping(); err != nil {
		services["database"] = "unhealthy"
	} else {
		services["database"] = "healthy"
	}

	// Check Redis
	if _, err := b.redis.Ping(b.ctx).Result(); err != nil {
		services["redis"] = "unhealthy"
	} else {
		services["redis"] = "healthy"
	}

	// Check queue lengths
	matchQueueLen, _ := b.redis.LLen(b.ctx, "match_queue").Result()
	notificationQueueLen, _ := b.redis.LLen(b.ctx, "notification_queue").Result()

	metrics["match_queue_length"] = matchQueueLen
	metrics["notification_queue_length"] = notificationQueueLen
	metrics["goroutines"] = int64(runtime.NumGoroutine())

	// Get active users
	activeUsers, _ := b.redis.Get(b.ctx, "system:active_users").Int64()
	metrics["active_users"] = activeUsers

	// Determine overall status
	status := "healthy"
	for _, serviceStatus := range services {
		if serviceStatus != "healthy" {
			status = "unhealthy"
			break
		}
	}

	// Check if queues are too long
	if matchQueueLen > 1000 || notificationQueueLen > 1000 {
		status = "degraded"
	}

	return HealthStatus{
		Status:    status,
		Timestamp: time.Now(),
		Services:  services,
		Metrics:   metrics,
	}
}

func (b *Bot) metricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := NewMetrics(b.redis)

	// Get recent metrics
	recentMetrics, err := metrics.GetMetricsHistory(1)
	if err != nil {
		http.Error(w, "Failed to get metrics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recentMetrics)
}
