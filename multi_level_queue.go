package priorityqueue

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

type MultiLevelQueue[T any] struct {
	mutex sync.RWMutex

	queues    map[any]Queuer[T]
	queueInit func() Queuer[T]

	// totalLength is an atomic integer representing for the total length of
	// multi-level queue.
	totalLength int32
}

func newMultilevelQueue[T any](queuer func() Queuer[T]) *MultiLevelQueue[T] {
	return &MultiLevelQueue[T]{
		mutex:       sync.RWMutex{},
		totalLength: 0,
		queues:      make(map[any]Queuer[T]),
		queueInit:   queuer,
	}
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

func (mq *MultiLevelQueue[T]) Enqueue(level any, obj T) error {
	// Pre-check
	if !mq.HasLevel(level) {
		return fmt.Errorf("%w: %v", ErrNotExistedLevel, level)
	}

	err := mq.queues[level].Enqueue(obj)
	if err == nil {
		atomic.AddInt32(&mq.totalLength, 1)
	}

	return err
}

func (mq *MultiLevelQueue[T]) Dequeue(level any) (T, error) {
	// Pre-check
	var defaultT T
	if !mq.HasLevel(level) {
		return defaultT, fmt.Errorf("%w: %v", ErrNotExistedLevel, level)
	}

	t, err := mq.queues[level].Dequeue()
	if err == nil {
		atomic.AddInt32(&mq.totalLength, -1)
	}

	return t, err
}

func (mq *MultiLevelQueue[T]) WaitDequeue(ctx context.Context, level any) (T, error) {
	// Pre-check
	var defaultT T
	if !mq.HasLevel(level) {
		return defaultT, fmt.Errorf("%w: %v", ErrNotExistedLevel, level)
	}

	t, err := mq.queues[level].WaitDequeue(ctx)
	if err == nil {
		atomic.AddInt32(&mq.totalLength, -1)
	}

	return t, err
}

func (mq *MultiLevelQueue[T]) Length(level any) (int, error) {
	if !mq.HasLevel(level) {
		return 0, fmt.Errorf("%w: %v", ErrNotExistedLevel, level)
	}

	return mq.queues[level].Length(), nil
}

func (mq *MultiLevelQueue[T]) TotalLength(level any) int {
	return int(mq.totalLength)
}
