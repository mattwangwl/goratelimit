# goratelimit
Rate-limit using Redis

## Quick start

```go
    func main() {
        rdb := redis.NewClient(&redis.Options{
            Addr:     "localhost:6379",
            Password: "",
            DB:       0,
        })
    
        limiter, err := goratelimit.New(rdb, goratelimit.Config{})
        if err != nil {
            panic(err)
        }
    
        for {
            if r, err := limiter.Allow(context.Background(), Key); err != nil {
                log.Println("error:", err)
            } else {
                if r.OK {
                    log.Println("pass", time.Now().String())
                }
            }
            <-time.After(50 * time.Millisecond)
        }
    }
```
