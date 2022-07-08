package goratelimit

import (
	"time"
)

type Config struct {
	RedisPrefix string // redis key 的前綴字串，預設 goRateLimit

	Limit int           // 限制次數，預設 1
	Per   time.Duration // 間隔時間，最小單位 1 ms，預設 1 sec

	TaskTimeout    time.Duration // 判斷放行的超時時間，最小單位 1 ms，預設 1 sec
	TaskBufferSize int           // 判斷放行的隊列大小，預設 128
	TaskRetryTime  int           // 高併發或時間差的狀況下 redis 計次無效時重新嘗試次數，預設 3 次

	WorkerSize int // 消化 task 的 func 同時執行的數量，預設 1
}

func (c *Config) getRedisPrefix() string {
	if c.RedisPrefix == "" {
		return "goRateLimit"
	}

	return c.RedisPrefix
}

func (c *Config) getLimit() int {
	if c.Limit < 1 {
		return 1
	}

	return c.Limit
}

func (c *Config) getPre() time.Duration {
	if c.Per < time.Millisecond {
		return time.Second
	}

	return c.Per
}

func (c *Config) getTaskTimeout() time.Duration {
	if c.TaskTimeout < time.Millisecond {
		return time.Second
	}

	return c.TaskTimeout
}

func (c *Config) getTaskBufferSize() int {
	if c.TaskBufferSize < 1 {
		return 128
	}

	return c.TaskBufferSize
}

func (c *Config) getTaskRetryTime() int {
	if c.TaskRetryTime < 1 {
		return 3
	}

	return c.TaskRetryTime
}

func (c *Config) getWorkerSize() int {
	if c.WorkerSize < 1 {
		return 1
	}

	return c.WorkerSize
}
