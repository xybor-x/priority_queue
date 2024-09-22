package priorityqueue

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type MultiLevelQueue[T any] struct {
	mutex sync.RWMutex

	queues    map[any]Queuer[T]
	queueInit func() Queuer[T]

	// totalLength is an atomic integer representing for the length of all
	// leveled queues.
	totalLength int32
}

func NewMultiLevelQueue[T any](queuer func() Queuer[T]) *MultiLevelQueue[T] {
	return &MultiLevelQueue[T]{
		mutex:       sync.RWMutex{},
		totalLength: 0,
		queues:      make(map[any]Queuer[T]),
		queueInit:   queuer,
	}
}

func DefaultMultiLevelQueue[T any]() *MultiLevelQueue[T] {
	return NewMultiLevelQueue(func() Queuer[T] {
		return NewQueue[T]()
	})
}

func (mq *MultiLevelQueue[T]) AddLevel(level any) error {
	// Pre-check
	if mq.HasLevel(level) {
		return fmt.Errorf("%w: %v", ErrExistedLevel, level)
	}

	mq.mutex.Lock()
	defer mq.mutex.Unlock()

	if _, ok := mq.queues[level]; ok {
		return fmt.Errorf("%w: %v", ErrExistedLevel, level)
	}

	mq.queues[level] = mq.queueInit()
	return nil
}

func (mq *MultiLevelQueue[T]) HasLevel(level any) bool {
	mq.mutex.RLock()
	defer mq.mutex.RUnlock()

	_, ok := mq.queues[level]
	return ok
}

func (mq *MultiLevelQueue[T]) Enqueue(level any, obj ...T) error {
	// Pre-check
	if !mq.HasLevel(level) {
		return fmt.Errorf("%w: %v", ErrNotExistedLevel, level)
	}

	err := mq.queues[level].Enqueue(obj...)
	if err == nil {
		atomic.AddInt32(&mq.totalLength, 1)
	}

	return err
}

func (mq *MultiLevelQueue[T]) DequeueIf(level any, cond func(T) (bool, error)) (T, error) {
	// Pre-check
	var defaultT T
	if !mq.HasLevel(level) {
		return defaultT, fmt.Errorf("%w: %v", ErrNotExistedLevel, level)
	}

	t, err := mq.queues[level].DequeueIf(cond)
	if err == nil {
		atomic.AddInt32(&mq.totalLength, -1)
	}

	return t, err
}

func (mq *MultiLevelQueue[T]) Dequeue(level any) (T, error) {
	return mq.DequeueIf(level, nil)
}

func (mq *MultiLevelQueue[T]) WaitDequeueIf(
	ctx context.Context,
	level any,
	cond func(T) (bool, error),
	retrigger func(T) <-chan time.Time,
) (T, error) {
	var defaultT T

	// Pre-check
	if !mq.HasLevel(level) {
		return defaultT, fmt.Errorf("%w: %v", ErrNotExistedLevel, level)
	}

	t, err := mq.queues[level].WaitDequeueIf(ctx, cond, retrigger)
	if err == nil {
		atomic.AddInt32(&mq.totalLength, -1)
	}

	return t, err
}

func (mq *MultiLevelQueue[T]) WaitDequeue(ctx context.Context, level any) (T, error) {
	return mq.WaitDequeueIf(ctx, level, nil, nil)
}

func (mq *MultiLevelQueue[T]) WaitDequeueAll(ctx context.Context) (T, error) {
	centerResult := make(chan WaitResult[T])

	ctx, cancel := context.WithCancel(ctx)

	alreadyHasResult := false
	mu := sync.Mutex{}

	for _, queue := range mq.queues {
		go func(ctx context.Context, queue Queuer[T]) {
			shouldSendToCenter := false

			t, err := queue.WaitDequeueIf(ctx, func(t T) (bool, error) {
				mu.Lock()
				defer mu.Unlock()

				if alreadyHasResult {
					return false, nil
				}

				shouldSendToCenter = true
				alreadyHasResult = true

				return true, nil
			}, nil)

			if shouldSendToCenter {
				centerResult <- WaitResult[T]{Value: t, Err: err}
				close(centerResult)
			}
		}(ctx, queue)
	}

	defer cancel()

	select {
	case result := <-centerResult:
		return result.Value, result.Err
	case <-ctx.Done():
		var defaultT T
		return defaultT, ErrTimeout
	}
}

func (mq *MultiLevelQueue[T]) Length(level any) int {
	if !mq.HasLevel(level) {
		return 0
	}

	return mq.queues[level].Length()
}

func (mq *MultiLevelQueue[T]) TotalLength() int {
	return int(mq.totalLength)
}
