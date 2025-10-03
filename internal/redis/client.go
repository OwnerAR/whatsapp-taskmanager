package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type Client struct {
	rdb *redis.Client
}

type SessionData struct {
	UserID      uint   `json:"user_id"`
	PhoneNumber string `json:"phone_number"`
	Command     string `json:"command"`
	Step        int    `json:"step"`
	Data        map[string]interface{} `json:"data"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TempData struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
	TTL   int         `json:"ttl"`
}

func Initialize(redisURL string) (*Client, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	rdb := redis.NewClient(opt)

	// Test connection
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{rdb: rdb}, nil
}

// Session management
func (c *Client) SetSession(sessionID string, data *SessionData, ttl time.Duration) error {
	ctx := context.Background()
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	return c.rdb.Set(ctx, "session:"+sessionID, jsonData, ttl).Err()
}

func (c *Client) GetSession(sessionID string) (*SessionData, error) {
	ctx := context.Background()
	val, err := c.rdb.Get(ctx, "session:"+sessionID).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session SessionData
	if err := json.Unmarshal([]byte(val), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	return &session, nil
}

func (c *Client) DeleteSession(sessionID string) error {
	ctx := context.Background()
	return c.rdb.Del(ctx, "session:"+sessionID).Err()
}

func (c *Client) UpdateSession(sessionID string, data *SessionData, ttl time.Duration) error {
	return c.SetSession(sessionID, data, ttl)
}

// Temporary data management
func (c *Client) SetTempData(key string, value interface{}, ttl time.Duration) error {
	ctx := context.Background()
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal temp data: %w", err)
	}

	return c.rdb.Set(ctx, "temp:"+key, jsonData, ttl).Err()
}

func (c *Client) GetTempData(key string, dest interface{}) error {
	ctx := context.Background()
	val, err := c.rdb.Get(ctx, "temp:"+key).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("temp data not found")
		}
		return fmt.Errorf("failed to get temp data: %w", err)
	}

	return json.Unmarshal([]byte(val), dest)
}

func (c *Client) DeleteTempData(key string) error {
	ctx := context.Background()
	return c.rdb.Del(ctx, "temp:"+key).Err()
}

// Task progress caching
func (c *Client) SetTaskProgress(taskID uint, progress int, ttl time.Duration) error {
	ctx := context.Background()
	key := fmt.Sprintf("task_progress:%d", taskID)
	return c.rdb.Set(ctx, key, progress, ttl).Err()
}

func (c *Client) GetTaskProgress(taskID uint) (int, error) {
	ctx := context.Background()
	key := fmt.Sprintf("task_progress:%d", taskID)
	val, err := c.rdb.Get(ctx, key).Int()
	if err != nil {
		if err == redis.Nil {
			return 0, fmt.Errorf("task progress not found")
		}
		return 0, fmt.Errorf("failed to get task progress: %w", err)
	}
	return val, nil
}

// Daily task cache
func (c *Client) SetDailyTaskProgress(taskID uint, date string, progress int, ttl time.Duration) error {
	ctx := context.Background()
	key := fmt.Sprintf("daily_task:%d:%s", taskID, date)
	return c.rdb.Set(ctx, key, progress, ttl).Err()
}

func (c *Client) GetDailyTaskProgress(taskID uint, date string) (int, error) {
	ctx := context.Background()
	key := fmt.Sprintf("daily_task:%d:%s", taskID, date)
	val, err := c.rdb.Get(ctx, key).Int()
	if err != nil {
		if err == redis.Nil {
			return 0, fmt.Errorf("daily task progress not found")
		}
		return 0, fmt.Errorf("failed to get daily task progress: %w", err)
	}
	return val, nil
}

// Monthly task cache
func (c *Client) SetMonthlyTaskProgress(taskID uint, monthYear string, progress int, ttl time.Duration) error {
	ctx := context.Background()
	key := fmt.Sprintf("monthly_task:%d:%s", taskID, monthYear)
	return c.rdb.Set(ctx, key, progress, ttl).Err()
}

func (c *Client) GetMonthlyTaskProgress(taskID uint, monthYear string) (int, error) {
	ctx := context.Background()
	key := fmt.Sprintf("monthly_task:%d:%s", taskID, monthYear)
	val, err := c.rdb.Get(ctx, key).Int()
	if err != nil {
		if err == redis.Nil {
			return 0, fmt.Errorf("monthly task progress not found")
		}
		return 0, fmt.Errorf("failed to get monthly task progress: %w", err)
	}
	return val, nil
}

// Close Redis connection
func (c *Client) Close() error {
	return c.rdb.Close()
}
