package goratelimit

import (
	"sync"
	"time"
)

/*
在地端紀錄，是否已到達限制，可降低 redis 流量
*/
type reached struct {
	m sync.Map
}

type reachedData struct {
	ExpireAt time.Time
	Reached  bool
}

func (r *reached) Reached(key string, expireAt time.Time) {
	r.m.Store(key, &reachedData{ExpireAt: expireAt, Reached: true})
}

func (r *reached) Allow(key string, now time.Time) bool {
	v, ok := r.m.Load(key)
	if ok {
		data := v.(*reachedData)
		if now.Before(data.ExpireAt) && data.Reached {
			return false
		}
	}
	return true
}

func (r *reached) Delete(key string) {
	r.m.Delete(key)
}

func (r *reached) clean() {
	for {
		now := time.Now().UTC()
		r.m.Range(func(key, value interface{}) bool {
			data := value.(*reachedData)
			if now.After(data.ExpireAt) {
				r.m.Delete(key)
			}

			return true
		})

		<-time.After(10 * time.Second)
	}
}
