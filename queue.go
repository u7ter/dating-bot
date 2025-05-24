package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

type MatchQueue struct {
	redis *redis.Client
	ctx   context.Context
}

type MatchJob struct {
	User1ID   int64     `json:"user1_id"`
	User2ID   int64     `json:"user2_id"`
	Timestamp time.Time `json:"timestamp"`
	Retries   int       `json:"retries"`
}

type NotificationJob struct {
	UserID    int64     `json:"user_id"`
	Type      string    `json:"type"`
	Data      string    `json:"data"`
	Timestamp time.Time `json:"timestamp"`
	Retries   int       `json:"retries"`
}

func NewMatchQueue(redisClient *redis.Client) *MatchQueue {
	return &MatchQueue{
		redis: redisClient,
		ctx:   context.Background(),
	}
}

// Add match job to queue
func (mq *MatchQueue) AddMatchJob(user1ID, user2ID int64) error {
	job := MatchJob{
		User1ID:   user1ID,
		User2ID:   user2ID,
		Timestamp: time.Now(),
		Retries:   0,
	}

	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return mq.redis.LPush(mq.ctx, "match_queue", data).Err()
}

// Add notification job to queue
func (mq *MatchQueue) AddNotificationJob(userID int64, notificationType, data string) error {
	job := NotificationJob{
		UserID:    userID,
		Type:      notificationType,
		Data:      data,
		Timestamp: time.Now(),
		Retries:   0,
	}

	jobData, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return mq.redis.LPush(mq.ctx, "notification_queue", jobData).Err()
}

// Process match jobs
func (b *Bot) matchWorker(workerID int) {
	defer b.wg.Done()

	log.Printf("Match worker %d started", workerID)

	for {
		select {
		case <-b.ctx.Done():
			log.Printf("Match worker %d stopping", workerID)
			return
		default:
			// Block for up to 5 seconds waiting for a job
			result, err := b.redis.BRPop(b.ctx, 5*time.Second, "match_queue").Result()
			if err != nil {
				if err != redis.Nil {
					log.Printf("Match worker %d error: %v", workerID, err)
				}
				continue
			}

			if len(result) < 2 {
				continue
			}

			var job MatchJob
			if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
				log.Printf("Match worker %d: failed to unmarshal job: %v", workerID, err)
				continue
			}

			// Process the match
			if err := b.processMatchJob(&job); err != nil {
				log.Printf("Match worker %d: failed to process match: %v", workerID, err)

				// Retry logic
				if job.Retries < 3 {
					job.Retries++
					if data, err := json.Marshal(job); err == nil {
						b.redis.LPush(b.ctx, "match_queue_retry", data)
					}
				} else {
					// Send to dead letter queue
					if data, err := json.Marshal(job); err == nil {
						b.redis.LPush(b.ctx, "match_queue_failed", data)
					}
				}
			}
		}
	}
}

// Process notification jobs
func (b *Bot) notificationWorker() {
	defer b.wg.Done()

	log.Println("Notification worker started")

	for {
		select {
		case <-b.ctx.Done():
			log.Println("Notification worker stopping")
			return
		default:
			result, err := b.redis.BRPop(b.ctx, 5*time.Second, "notification_queue").Result()
			if err != nil {
				if err != redis.Nil {
					log.Printf("Notification worker error: %v", err)
				}
				continue
			}

			if len(result) < 2 {
				continue
			}

			var job NotificationJob
			if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
				log.Printf("Notification worker: failed to unmarshal job: %v", err)
				continue
			}

			if err := b.processNotificationJob(&job); err != nil {
				log.Printf("Notification worker: failed to process notification: %v", err)

				if job.Retries < 3 {
					job.Retries++
					if data, err := json.Marshal(job); err == nil {
						b.redis.LPush(b.ctx, "notification_queue_retry", data)
					}
				}
			}
		}
	}
}

// Cleanup worker for expired data
func (b *Bot) cleanupWorker() {
	defer b.wg.Done()

	log.Println("Cleanup worker started")
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-b.ctx.Done():
			log.Println("Cleanup worker stopping")
			return
		case <-ticker.C:
			b.performCleanup()
		}
	}
}

func (b *Bot) processMatchJob(job *MatchJob) error {
	// Use circuit breaker for database operations
	result, err := b.circuitBreaker.Execute(func() (interface{}, error) {
		return nil, b.createMatchInDB(job.User1ID, job.User2ID)
	})

	if err != nil {
		return err
	}

	// Update cache
	b.userCache.IncrementMatchCount(job.User1ID)
	b.userCache.IncrementMatchCount(job.User2ID)

	// Add notification jobs
	b.matchQueue.AddNotificationJob(job.User1ID, "new_match", fmt.Sprintf("%d", job.User2ID))
	b.matchQueue.AddNotificationJob(job.User2ID, "new_match", fmt.Sprintf("%d", job.User1ID))

	// Update daily stats
	b.userCache.IncrementDailyStats("matches_created")

	return nil
}

func (b *Bot) processNotificationJob(job *NotificationJob) error {
	switch job.Type {
	case "new_match":
		return b.sendMatchNotification(job.UserID, job.Data)
	case "new_message":
		return b.sendMessageNotification(job.UserID, job.Data)
	case "profile_view":
		return b.sendProfileViewNotification(job.UserID, job.Data)
	default:
		log.Printf("Unknown notification type: %s", job.Type)
	}
	return nil
}

func (b *Bot) performCleanup() {
	log.Println("Starting cleanup process...")

	// Clean up old user states
	pattern := "state:*"
	keys, err := b.redis.Keys(b.ctx, pattern).Result()
	if err == nil {
		for _, key := range keys {
			ttl, _ := b.redis.TTL(b.ctx, key).Result()
			if ttl < 0 { // No expiration set
				b.redis.Expire(b.ctx, key, time.Hour*24)
			}
		}
	}

	// Clean up old rate limit data
	pattern = "rate_limit:*"
	keys, err = b.redis.Keys(b.ctx, pattern).Result()
	if err == nil {
		for _, key := range keys {
			// Remove entries older than 1 hour
			cutoff := time.Now().Add(-time.Hour).UnixNano()
			b.redis.ZRemRangeByScore(b.ctx, key, "0", fmt.Sprintf("%d", cutoff))
		}
	}

	// Clean up old activity data
	cutoff := time.Now().Add(-time.Hour * 24).Unix()
	pattern = "activity:*"
	keys, err = b.redis.Keys(b.ctx, pattern).Result()
	if err == nil {
		for _, key := range keys {
			timestamp, _ := b.redis.Get(b.ctx, key).Int64()
			if timestamp < cutoff {
				b.redis.Del(b.ctx, key)
			}
		}
	}

	log.Println("Cleanup process completed")
}

// Priority queue for important notifications
func (mq *MatchQueue) AddPriorityNotification(userID int64, notificationType, data string, priority int) error {
	job := NotificationJob{
		UserID:    userID,
		Type:      notificationType,
		Data:      data,
		Timestamp: time.Now(),
		Retries:   0,
	}

	jobData, err := json.Marshal(job)
	if err != nil {
		return err
	}

	queueName := fmt.Sprintf("notification_queue_p%d", priority)
	return mq.redis.LPush(mq.ctx, queueName, jobData).Err()
}

// Batch processing for better performance
func (mq *MatchQueue) AddBatchMatchJobs(jobs []MatchJob) error {
	pipe := mq.redis.Pipeline()

	for _, job := range jobs {
		data, err := json.Marshal(job)
		if err != nil {
			continue
		}
		pipe.LPush(mq.ctx, "match_queue", data)
	}

	_, err := pipe.Exec(mq.ctx)
	return err
}

// Delayed job processing
func (mq *MatchQueue) AddDelayedJob(job interface{}, delay time.Duration, queueName string) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	executeAt := time.Now().Add(delay).Unix()
	return mq.redis.ZAdd(mq.ctx, "delayed_jobs", &redis.Z{
		Score:  float64(executeAt),
		Member: fmt.Sprintf("%s:%s", queueName, string(data)),
	}).Err()
}

// Process delayed jobs
func (b *Bot) processDelayedJobs() {
	now := time.Now().Unix()

	// Get jobs that should be executed now
	jobs, err := b.redis.ZRangeByScore(b.ctx, "delayed_jobs", &redis.ZRangeBy{
		Min: "0",
		Max: fmt.Sprintf("%d", now),
	}).Result()

	if err != nil {
		return
	}

	for _, job := range jobs {
		parts := strings.SplitN(job, ":", 2)
		if len(parts) != 2 {
			continue
		}

		queueName := parts[0]
		jobData := parts[1]

		// Move to appropriate queue
		b.redis.LPush(b.ctx, queueName, jobData)

		// Remove from delayed jobs
		b.redis.ZRem(b.ctx, "delayed_jobs", job)
	}
}
