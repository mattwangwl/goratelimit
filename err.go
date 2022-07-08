package goratelimit

import (
	"errors"
)

var (
	ErrRedisClientIsNil    = errors.New("redis client is nil")
	ErrTaskTimeout         = errors.New("task timeout")
	ErrTaskClose           = errors.New("task close")
	ErrIncrSliceOutOfRange = errors.New("incr slice out of range")
)
