package main

import (
	"context"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/mattwangwl/goratelimit"
)

const (
	Concurrency = 100
	Key         = "test"

	LimiterLimit          = 1
	LimiterPer            = 1 * time.Second
	LimiterTaskTimeout    = 10 * time.Second
	LimiterTaskBufferSize = 1000
	LimiterWorkerSize     = 100

	RedisAddr     = "localhost:6379"
	RedisPassword = ""
	RedisDB       = 0
)

func main() {
	rdb := newRdb()

	limiter, err := goratelimit.New(rdb, goratelimit.Config{
		Limit:          LimiterLimit,
		Per:            LimiterPer,
		TaskTimeout:    LimiterTaskTimeout,
		TaskBufferSize: LimiterTaskBufferSize,
		WorkerSize:     LimiterWorkerSize,
	})
	if err != nil {
		panic(err)
	}

	for i := 0; i < Concurrency; i++ {
		go func() {
			for {
				if r, err := limiter.Allow(context.Background(), Key); err != nil {
					log.Println("error:", err)
				} else {
					if r.OK {
						log.Println("pass", time.Now().String())
					}
				}

				runtime.Gosched()
				<-time.After(50 * time.Millisecond)
			}
		}()
	}
	go func() {
		timeKey := "goRateLimit:" + Key + ":time"
		countKey := "goRateLimit:" + Key + ":count"

		t := time.NewTicker(1 * time.Second)
		for range t.C {
			log.Println("---------- watch start ----------")
			log.Println(rdb.Get(timeKey))
			log.Println(rdb.TTL(timeKey))
			log.Println(rdb.LLen(countKey))
			log.Println(rdb.TTL(countKey))
			log.Println("---------- watch edn ----------")
		}
	}()

	// http://localhost:6969/debug/pprof
	http.ListenAndServe("localhost:6969", nil)
}

func newRdb() *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     RedisAddr,
		Password: RedisPassword,
		DB:       RedisDB,
	})

	if err := rdb.Ping().Err(); err != nil {
		panic(err)
	}

	return rdb
}
