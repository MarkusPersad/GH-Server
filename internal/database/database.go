package database

import (
	"GH-Server/internal/model"
	"GH-Server/pkg/zaplog"
	"context"
	"database/sql"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/joho/godotenv/autoload"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var (
	dbHost                  = os.Getenv("DB_HOST")
	dbPort                  = os.Getenv("DB_PORT")
	dbUser                  = os.Getenv("DB_USER")
	dbPassword              = os.Getenv("DB_PASS")
	dbName                  = os.Getenv("DB_NAME")
	dbSslMode               = os.Getenv("DB_SSL")
	dbMaxOpenConnections, _ = strconv.Atoi(os.Getenv("DB_MAX_OPEN_CONNS"))
	dbMaxIdleConnections, _ = strconv.Atoi(os.Getenv("DB_IDLE_CONNS"))
	dbMaxLifeTime, _        = strconv.Atoi(os.Getenv("DB_MAX_LIFETIME"))
	dbMaxIdleTime, _        = strconv.Atoi(os.Getenv("DB_MAX_IDLE_TIME"))
	rdbHost                 = os.Getenv("RDB_HOST")
	rdbPort                 = os.Getenv("RDB_PORT")
	rdbPassword             = os.Getenv("RDB_PASS")
	rdbProtocol, _          = strconv.Atoi(os.Getenv("PROTOCOL"))
	rdbDb, _                = strconv.Atoi(os.Getenv("RDB_DB"))
	rdbMaxRetries, _        = strconv.Atoi(os.Getenv("RDB_MAX_RETRIES"))
	instance                *service
)

type service struct {
	db  *gorm.DB
	rdb *redis.Client
}

type Service interface {
	Health() map[string]string
	UserService
	GetRedisClient() *redis.Client
}

func New() Service {
	if instance != nil {
		return instance
	}
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Shanghai",
			dbHost, dbUser, dbPassword, dbName, dbPort, dbSslMode,
		),
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		SkipDefaultTransaction: true,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if Db, err := db.DB(); err != nil {
		zaplog.Zap.Panic(fmt.Sprintf("failed to connect database: %v", err))
	} else {
		Db.SetConnMaxLifetime(time.Duration(dbMaxLifeTime) * time.Minute)
		Db.SetConnMaxIdleTime(time.Duration(dbMaxIdleTime) * time.Minute)
		Db.SetMaxIdleConns(dbMaxIdleConnections)
		Db.SetMaxOpenConns(dbMaxOpenConnections)
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", rdbHost, rdbPort),
		Password: rdbPassword,
		Protocol: rdbProtocol,
		DB:       rdbDb,
		MaxRetries: rdbMaxRetries,
	})
	if err != nil {
		zaplog.Zap.Panic(fmt.Sprintf("failed to connect database: %v", err))
	}
	instance = &service{db: db, rdb: rdb}

	if err := db.AutoMigrate(&model.User{}); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("failed to migrate database: %v", err))
	}
	return instance
}

func(s *service) GetRedisClient() *redis.Client{
	return s.rdb
}

func (s *service) Health() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	stats := make(map[string]string)
	var Db *sql.DB
	var err error
	if Db, err = s.db.DB(); err != nil {
		zaplog.Zap.Panic(fmt.Sprintf("failed to connect database: %v", err))
	} else {
		if err = Db.PingContext(ctx); err != nil {
			stats["status"] = "down"
			stats["error"] = fmt.Sprintf("db down: %v", err)
			zaplog.Zap.Fatal(fmt.Sprintf("db down: %v", err))
			return stats
		}
	}
	stats["DB status"] = "up"
	stats["DB message"] = "It's healthy"

	// Get database stats (like open connections, in use, idle, etc.)
	dbStats := Db.Stats()
	stats["DB open_connections"] = strconv.Itoa(dbStats.OpenConnections)
	stats["DB in_use"] = strconv.Itoa(dbStats.InUse)
	stats["DB idle"] = strconv.Itoa(dbStats.Idle)
	stats["DB wait_count"] = strconv.FormatInt(dbStats.WaitCount, 10)
	stats["DB wait_duration"] = dbStats.WaitDuration.String()
	stats["DB max_idle_closed"] = strconv.FormatInt(dbStats.MaxIdleClosed, 10)
	stats["DB max_lifetime_closed"] = strconv.FormatInt(dbStats.MaxLifetimeClosed, 10)

	// Evaluate stats to provide a health message
	if dbStats.OpenConnections > 40 { // Assuming 50 is the max for this example
		stats["DB message"] = "The database is experiencing heavy load."
	}

	if dbStats.WaitCount > 1000 {
		stats["DB message"] = "The database has a high number of wait events, indicating potential bottlenecks."
	}

	if dbStats.MaxIdleClosed > int64(dbStats.OpenConnections)/2 {
		stats["DB message"] = "Many idle connections are being closed, consider revising the connection pool settings."
	}

	if dbStats.MaxLifetimeClosed > int64(dbStats.OpenConnections)/2 {
		stats["DB message"] = "Many connections are being closed due to max lifetime, consider increasing max lifetime or revising the connection usage pattern."
	}

	stats = s.checkRedisHealth(ctx, stats)

	return stats
}

// checkRedisHealth checks the health of the Redis server and adds the relevant statistics to the stats map.
func (s *service) checkRedisHealth(ctx context.Context, stats map[string]string) map[string]string {
	// Ping the Redis server to check its availability.
	pong, err := s.rdb.Ping(ctx).Result()
	// Note: By extracting and simplifying like this, `log.Fatalf("db down: %v", err)`
	// can be changed into a standard error instead of a fatal error.
	if err != nil {
		zaplog.Zap.Fatal(fmt.Sprintf("redis down: %v", err))
	}

	// Redis is up
	stats["redis_status"] = "up"
	stats["redis_message"] = "It's healthy"
	stats["redis_ping_response"] = pong

	// Retrieve Redis server information.
	info, err := s.rdb.Info(ctx).Result()
	if err != nil {
		stats["redis_message"] = fmt.Sprintf("Failed to retrieve Redis info: %v", err)
		return stats
	}

	// Parse the Redis info response.
	redisInfo := parseRedisInfo(info)

	// Get the pool stats of the Redis client.
	poolStats := s.rdb.PoolStats()

	// Prepare the stats map with Redis server information and pool statistics.
	// Note: The "stats" map in the code uses string keys and values,
	// which is suitable for structuring and serializing the data for the frontend (e.g., JSON, XML, HTMX).
	// Using string types allows for easy conversion and compatibility with various data formats,
	// making it convenient to create health stats for monitoring or other purposes.
	// Also note that any raw "memory" (e.g., used_memory) value here is in bytes and can be converted to megabytes or gigabytes as a float64.
	stats["redis_version"] = redisInfo["redis_version"]
	stats["redis_mode"] = redisInfo["redis_mode"]
	stats["redis_connected_clients"] = redisInfo["connected_clients"]
	stats["redis_used_memory"] = redisInfo["used_memory"]
	stats["redis_used_memory_peak"] = redisInfo["used_memory_peak"]
	stats["redis_uptime_in_seconds"] = redisInfo["uptime_in_seconds"]
	stats["redis_hits_connections"] = strconv.FormatUint(uint64(poolStats.Hits), 10)
	stats["redis_misses_connections"] = strconv.FormatUint(uint64(poolStats.Misses), 10)
	stats["redis_timeouts_connections"] = strconv.FormatUint(uint64(poolStats.Timeouts), 10)
	stats["redis_total_connections"] = strconv.FormatUint(uint64(poolStats.TotalConns), 10)
	stats["redis_idle_connections"] = strconv.FormatUint(uint64(poolStats.IdleConns), 10)
	stats["redis_stale_connections"] = strconv.FormatUint(uint64(poolStats.StaleConns), 10)
	stats["redis_max_memory"] = redisInfo["maxmemory"]

	// Calculate the number of active connections.
	// Note: We use math.Max to ensure that activeConns is always non-negative,
	// avoiding the need for an explicit check for negative values.
	// This prevents a potential underflow situation.
	activeConns := uint64(math.Max(float64(poolStats.TotalConns-poolStats.IdleConns), 0))
	stats["redis_active_connections"] = strconv.FormatUint(activeConns, 10)

	// Calculate the pool size percentage.
	poolSize := s.rdb.Options().PoolSize
	connectedClients, _ := strconv.Atoi(redisInfo["connected_clients"])
	poolSizePercentage := float64(connectedClients) / float64(poolSize) * 100
	stats["redis_pool_size_percentage"] = fmt.Sprintf("%.2f%%", poolSizePercentage)

	// Evaluate Redis stats and update the stats map with relevant messages.
	return s.evaluateRedisStats(redisInfo, stats)
}

// evaluateRedisStats evaluates the Redis server statistics and updates the stats map with relevant messages.
func (s *service) evaluateRedisStats(redisInfo, stats map[string]string) map[string]string {
	poolSize := s.rdb.Options().PoolSize
	poolStats := s.rdb.PoolStats()
	connectedClients, _ := strconv.Atoi(redisInfo["connected_clients"])
	highConnectionThreshold := int(float64(poolSize) * 0.8)

	// Check if the number of connected clients is high.
	if connectedClients > highConnectionThreshold {
		stats["redis_message"] = "Redis has a high number of connected clients"
	}

	// Check if the number of stale connections exceeds a threshold.
	minStaleConnectionsThreshold := 500
	if int(poolStats.StaleConns) > minStaleConnectionsThreshold {
		stats["redis_message"] = fmt.Sprintf("Redis has %d stale connections.", poolStats.StaleConns)
	}

	// Check if Redis is using a significant amount of memory.
	usedMemory, _ := strconv.ParseInt(redisInfo["used_memory"], 10, 64)
	maxMemory, _ := strconv.ParseInt(redisInfo["maxmemory"], 10, 64)
	if maxMemory > 0 {
		usedMemoryPercentage := float64(usedMemory) / float64(maxMemory) * 100
		if usedMemoryPercentage >= 90 {
			stats["redis_message"] = "Redis is using a significant amount of memory"
		}
	}

	// Check if Redis has been recently restarted.
	uptimeInSeconds, _ := strconv.ParseInt(redisInfo["uptime_in_seconds"], 10, 64)
	if uptimeInSeconds < 3600 {
		stats["redis_message"] = "Redis has been recently restarted"
	}

	// Check if the number of idle connections is high.
	idleConns := int(poolStats.IdleConns)
	highIdleConnectionThreshold := int(float64(poolSize) * 0.7)
	if idleConns > highIdleConnectionThreshold {
		stats["redis_message"] = "Redis has a high number of idle connections"
	}

	// Check if the connection pool utilization is high.
	poolUtilization := float64(poolStats.TotalConns-poolStats.IdleConns) / float64(poolSize) * 100
	highPoolUtilizationThreshold := 90.0
	if poolUtilization > highPoolUtilizationThreshold {
		stats["redis_message"] = "Redis connection pool utilization is high"
	}

	return stats
}

// parseRedisInfo parses the Redis info response and returns a map of key-value pairs.
func parseRedisInfo(info string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			result[key] = value
		}
	}
	return result
}
