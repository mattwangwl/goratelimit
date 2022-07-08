package goratelimit

import (
	"context"
)

type task struct {
	ctx      context.Context
	key      string
	resultCh chan taskResult
}

type taskResult struct {
	ok  bool
	err error
}

func (t *task) Reset() task {
	t.ctx = context.Background()
	t.key = ""
	t.resultCh = make(chan taskResult, 1)

	return *t
}

func (t *task) SetResult(result taskResult) {
	select {
	case t.resultCh <- result: // 傳入結果
		close(t.resultCh)
	default:
	}
}

func (t *task) Result() (*Result, error) {
	select {
	case r := <-t.resultCh: // 等待結果
		if r.err != nil {
			return nil, r.err
		}
		return &Result{OK: r.ok}, nil
	case <-t.ctx.Done(): // 超時
		return nil, ErrTaskTimeout
	}
	return nil, ErrTaskClose
}

func newReached() *reached {
	obj := &reached{}
	go obj.clean()
	return obj
}
