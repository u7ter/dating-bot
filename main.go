package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
	"github.com/sony/gobreaker"
	tele "gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"
)

type Bot struct {
	*tele.Bot
	db             *sql.DB
	redis          *redis.Client
	rateLimiter    *RateLimiter
	circuitBreaker *gobreaker.CircuitBreaker
	userCache      *UserCache
	matchQueue     *MatchQueue
	ctx            context.Context
	cancel         context.CancelFunc
	wg             *sync.WaitGroup
}

type User struct {
	ID          int64     `json:"id"`
	TelegramID  int64     `json:"telegram_id"`
	Name        string    `json:"name"`
	Age         int       `json:"age"`
	Gender      string    `json:"gender"`
	Description string    `json:"description"`
	City        string    `json:"city"`
	LookingFor  string    `json:"looking_for"`
	Photos      []string  `json:"photos"`
	CreatedAt   time.Time `json:"created_at"`
	IsActive    bool      `json:"is_active"`
}

type Match struct {
	ID        int64     `json:"id"`
	User1ID   int64     `json:"user1_id"`
	User2ID   int64     `json:"user2_id"`
	CreatedAt time.Time `json:"created_at"`
	IsActive  bool      `json:"is_active"`
}

type Like struct {
	ID        int64     `json:"id"`
	LikerID   int64     `json:"liker_id"`
	LikedID   int64     `json:"liked_id"`
	IsLike    bool      `json:"is_like"`
	CreatedAt time.Time `json:"created_at"`
}

type UserState struct {
	TelegramID int64                  `json:"telegram_id"`
	State      string                 `json:"state"`
	Data       map[string]interface{} `json:"data"`
}

type Config struct {
	TelegramToken    string
	DatabaseURL      string
	RedisURL         string
	MaxDBConnections int
	RateLimit        int
	BurstLimit       int
	WorkerCount      int
}

// Keep memory states as fallback
var userStates = make(map[int64]*UserState)
var userStatesMutex = sync.RWMutex{}

func main() {
	config := loadConfig()

	// Initialize context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database with connection pooling
	db, err := initDBWithPool(config)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Initialize Redis
	redisClient, err := initRedis(config)
	if err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}
	defer redisClient.Close()

	// Initialize bot with middleware
	bot, err := initBotWithMiddleware(config)
	if err != nil {
		log.Fatal("Failed to create bot:", err)
	}

	// Initialize components
	app := &Bot{
		Bot:            bot,
		db:             db,
		redis:          redisClient,
		rateLimiter:    NewRateLimiter(redisClient, config.RateLimit, config.BurstLimit),
		circuitBreaker: initCircuitBreaker(),
		userCache:      NewUserCache(redisClient),
		matchQueue:     NewMatchQueue(redisClient),
		ctx:            ctx,
		cancel:         cancel,
		wg:             &sync.WaitGroup{},
	}

	// Start background workers
	app.startWorkers(config.WorkerCount)

	// Start health server
	go app.startHealthServer()

	// Register handlers
	app.registerHandlers()

	// Start metrics collection
	go app.startMetricsCollection()

	// Graceful shutdown
	go app.handleShutdown()

	log.Println("Bot started with fixed button handling...")
	app.Start()
}

func loadConfig() *Config {
	return &Config{
		TelegramToken:    getEnv("TELEGRAM_BOT_TOKEN", ""),
		DatabaseURL:      getEnv("DATABASE_URL", "root:password@tcp(mysql:3306)/dating_bot?charset=utf8mb4&parseTime=True&loc=Local"),
		RedisURL:         getEnv("REDIS_URL", "redis:6379"),
		MaxDBConnections: getEnvInt("MAX_DB_CONNECTIONS", 100),
		RateLimit:        getEnvInt("RATE_LIMIT", 10),
		BurstLimit:       getEnvInt("BURST_LIMIT", 20),
		WorkerCount:      getEnvInt("WORKER_COUNT", 10),
	}
}

func initDBWithPool(config *Config) (*sql.DB, error) {
	db, err := sql.Open("mysql", config.DatabaseURL)
	if err != nil {
		return nil, err
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxDBConnections)
	db.SetMaxIdleConns(config.MaxDBConnections / 2)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(time.Minute * 30)

	// Wait for database to be ready
	for i := 0; i < 30; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		log.Println("Waiting for database...")
		time.Sleep(2 * time.Second)
	}

	return db, nil
}

func initRedis(config *Config) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         config.RedisURL,
		Password:     getEnv("REDIS_PASSWORD", ""),
		DB:           0,
		PoolSize:     50,
		MinIdleConns: 10,
		MaxRetries:   3,
		DialTimeout:  time.Second * 5,
		ReadTimeout:  time.Second * 3,
		WriteTimeout: time.Second * 3,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	return rdb, err
}

func initBotWithMiddleware(config *Config) (*tele.Bot, error) {
	bot, err := tele.NewBot(tele.Settings{
		Token:  config.TelegramToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		return nil, err
	}

	// Add middleware
	bot.Use(middleware.Logger())
	bot.Use(middleware.Recover())

	return bot, nil
}

func initCircuitBreaker() *gobreaker.CircuitBreaker {
	settings := gobreaker.Settings{
		Name:        "database",
		MaxRequests: 3,
		Interval:    time.Second * 60,
		Timeout:     time.Second * 30,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
	}
	return gobreaker.NewCircuitBreaker(settings)
}

func (b *Bot) startWorkers(workerCount int) {
	for i := 0; i < workerCount; i++ {
		b.wg.Add(1)
		go b.matchWorker(i)
	}

	b.wg.Add(1)
	go b.notificationWorker()

	b.wg.Add(1)
	go b.cleanupWorker()
}

func (b *Bot) handleShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutting down gracefully...")

	b.cancel()
	b.Stop()

	// Wait for workers to finish
	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All workers stopped")
	case <-time.After(30 * time.Second):
		log.Println("Timeout waiting for workers")
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
