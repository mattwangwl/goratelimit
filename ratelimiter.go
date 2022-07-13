package goratelimit

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v7"
)

func New(rdb *redis.Client, conf Config) (IRateLimiter, error) {
	if rdb == nil {
		return nil, ErrRedisClientIsNil
	}

	obj := &rateLimiter{
		rdb:           rdb,
		reached:       newReached(),
		redisPrefix:   conf.getRedisPrefix(),
		limit:         conf.getLimit(),
		per:           conf.getPre(),
		taskTimeout:   conf.getTaskTimeout(),
		taskCh:        make(chan task, conf.getTaskBufferSize()),
		taskRetryTime: conf.getTaskRetryTime(),
		pool: sync.Pool{
			New: func() interface{} {
				return task{}
			},
		},
	}

	for i := 0; i < conf.getWorkerSize(); i++ {
		go obj.do()
	}

	return obj, nil
}

type IRateLimiter interface {
	Allow(ctx context.Context, key string) (*Result, error)
}

type rateLimiter struct {
	rdb     *redis.Client
	reached *reached

	redisPrefix string

	limit int
	per   time.Duration

	taskTimeout   time.Duration
	taskCh        chan task
	taskRetryTime int

	pool sync.Pool
}

func (r *rateLimiter) Allow(ctx context.Context, key string) (*Result, error) {
	t := r.newTask(ctx, key)
	defer func() { r.pool.Put(t.Reset()) }()

	r.taskCh <- t

	return t.Result()
}

func (r *rateLimiter) newTask(ctx context.Context, key string) task {
	t := r.pool.Get().(task)

	// 超時設定，利用 context 來傳送超時
	withTimeout := func(ctx context.Context) context.Context {
		ctx, _ = context.WithDeadline(ctx, time.Now().Add(r.taskTimeout))
		return ctx
	}

	t.ctx = withTimeout(ctx)
	t.key = key
	t.resultCh = make(chan taskResult, 1)

	return t
}

func (r *rateLimiter) do() {
	for t := range r.taskCh {
		func(task task) {
			select {
			case <-task.ctx.Done():
				task.SetResult(taskResult{ok: false, err: ErrTaskTimeout})
				return
			default:
			}

			key := r.redisPrefix + ":" + task.key + ":time"       // 紀錄開始時間 key
			keyCount := r.redisPrefix + ":" + task.key + ":count" // 記錄流量的 key

			now := time.Now().UTC()

			if ok := r.reached.Allow(key, now); !ok {
				task.SetResult(taskResult{ok: false, err: nil})
				return
			}

			// redis 計次
			nowCount, err := r.counterFromRedis(key, keyCount, now)
			if err != nil {
				task.SetResult(taskResult{ok: false, err: err})
				return
			}

			// 判斷是否取不到或超過限額
			if nowCount < 1 || nowCount > r.limit {
				if t, err := r.rdb.Get(key).Int64(); err != nil {
				} else {
					expireAt := time.Unix(0, t).Add(r.per)
					if now.Before(expireAt) {
						r.reached.Reached(key, expireAt)
					}
				}

				task.SetResult(taskResult{ok: false, err: nil})
				return
			}

			// 放行
			task.SetResult(taskResult{ok: true, err: nil})
			return
		}(t)
	}
}

func (r *rateLimiter) counterFromRedis(key, keyCount string, now time.Time) (int, error) {
	retry := r.taskRetryTime
	for retry > 0 {
		ok, err := r.rdb.Exists(keyCount).Result()
		if err != nil {
			return 0, err
		}

		if ok == 0 {
			// 開始計次，使用 Redis transaction (multi)
			pipe := r.rdb.TxPipeline()
			// 將主 key 往 Redis 寫入，並判斷是否已存在
			pipe.SetNX(key, now.UnixNano(), r.per)
			pipe.RPush(keyCount, "")
			pipe.Expire(keyCount, r.per)
			cmds, err := pipe.Exec()
			if err != nil {
				return 0, err
			}

			// 取得目前訪問次數
			nowCount, err := r.getNowCount(cmds)
			if err != nil {
				return 0, err
			}

			return nowCount, nil
		} else {
			// 開始計次，當 key 不在就寫入不了
			nowCount, err := r.rdb.RPushX(keyCount, 1).Result()
			if err != nil {
				return 0, err
			}
			if nowCount == 0 {
				retry--
				continue
			}
			return int(nowCount), nil
		}
	}

	return 0, nil
}

// 取得 Redis incr 當前值
func (r *rateLimiter) getNowCount(cmds []redis.Cmder) (val int, err error) {
	defer func() {
		if rc := recover(); rc != nil {
			val = 0
			err = ErrIncrSliceOutOfRange
		}
	}()

	for _, cmd := range cmds {
		if cmd.Name() == "rpush" {
			t := cmd.String()[strings.LastIndex(cmd.String(), ":")+2:]
			c, err := strconv.Atoi(t)
			if err != nil {
				return 0, err
			}
			return c, nil
		}
	}

	return 0, nil
}

type Result struct {
	OK bool // 放行
}
