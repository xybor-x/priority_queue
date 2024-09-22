package priorityqueue

import (
	"context"
	"errors"
	"sync"
	"time"
)

type WaitResult[T any] struct {
	Value T
	Err   error
}

// Queuer represents for a thread-safe queue.
type Queuer[T any] interface {
	// Enqueue push an element into the queue.
	Enqueue(obj ...T) error

	// DequeueIf returns and removes the element at the begining of the queue if
	// the element meets the condition. This function is non-blocking.
	DequeueIf(func(t T) (bool, error)) (T, error)

	// Dequeue returns and removes the element at the begining of the queue.
	// This function is non-blocking.
	Dequeue() (T, error)

	// WaitDequeueIf is the same as DequeueIf, but this function is blocking.
	//
	// If the condition was not sastified, this function will check again when
	// the returned channel of retrigger function sends a signal.
	//
	// If you want returns immediately when the condition was not sastified,
	// pass a nil value to the retrigger parameter.
	//
	// For example:
	// 	// No retrigger.
	// 	queue.WaitDequeueIf(ctx, nil, nil)
	//
	//	// Retrigger after some time.
	// 	queue.WaitDequeueIf(
	//  	ctx,
	// 		func(t time.Time) (bool, error) {
	// 			return time.Now().After(t), nil
	// 		},
	//		func(t time.Time) <- chan time.Time {
	//			// Approach 1: Check after every 5 minutes.
	//			return time.After(5*time.Minutes)
	//
	//			// Approach 2: Re-check at t.
	//			return time.After(time.Since(t))
	// 		}
	// 	)
	WaitDequeueIf(
		ctx context.Context,
		cond func(t T) (bool, error),
		retrigger func(T) <-chan time.Time,
	) (T, error)

	// WaitDequeue is the same as Dequeue, but this function is blocking.
	WaitDequeue(ctx context.Context) (T, error)

	// Front returns the element at the begining of the queue.
	Front() (T, error)

	// Back returns the element at the end of the queue.
	Back() (T, error)

	// Clear removes all elements in the queue.
	Clear()

	// Length returns the number of elements in the queue.
	Length() int
}

// DefaultQueue implementation.
var _ Queuer[int] = (*Queue[int])(nil)

type Queue[T any] struct {
	mutex sync.RWMutex
	queue []T

	wakeupCh chan any
}

func NewQueue[T any]() *Queue[T] {
	return &Queue[T]{
		mutex:    sync.RWMutex{},
		queue:    make([]T, 0),
		wakeupCh: make(chan any),
	}
}

func (q *Queue[T]) Enqueue(obj ...T) error {
	q.mutex.Lock()
	defer func() {
		q.mutex.Unlock()
		q.wakeupAWaitingDequeue()
	}()

	q.queue = append(q.queue, obj...)
	return nil
}

func (q *Queue[T]) DequeueIf(cond func(t T) (bool, error)) (T, error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	var defaultT T
	if len(q.queue) == 0 {
		return defaultT, ErrEmpty
	}

	if cond != nil {
		ok, err := cond(q.queue[0])
		if err != nil {
			return defaultT, err
		}

		if !ok {
			return q.queue[0], ErrNotMeetCondition
		}
	}

	v := q.queue[0]
	q.queue = q.queue[1:]

	return v, nil
}

func (q *Queue[T]) Dequeue() (T, error) {
	return q.DequeueIf(nil)
}

func (q *Queue[T]) WaitDequeueIf(
	ctx context.Context,
	cond func(t T) (bool, error),
	retrigger func(T) <-chan time.Time,
) (T, error) {

	for {
		t, err := q.DequeueIf(cond)
		if !errors.Is(err, ErrEmpty) && !errors.Is(err, ErrNotMeetCondition) {
			return t, err
		}

		if errors.Is(err, ErrNotMeetCondition) {
			q.wakeupAWaitingDequeue()

			if retrigger == nil {
				return t, err
			}

			select {
			case <-retrigger(t):
			case <-ctx.Done():
				return t, ErrTimeout
			}
		} else {
			if !q.sleepUntilBeWokenUp(ctx) {
				return t, ErrTimeout
			}
		}
	}
}

func (q *Queue[T]) WaitDequeue(ctx context.Context) (T, error) {
	return q.WaitDequeueIf(ctx, nil, nil)
}

func (q *Queue[T]) Front() (T, error) {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	if len(q.queue) == 0 {
		var defaultT T
		return defaultT, ErrEmpty
	}

	return q.queue[0], nil
}

func (q *Queue[T]) Back() (T, error) {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	if len(q.queue) == 0 {
		var defaultT T
		return defaultT, ErrEmpty
	}

	return q.queue[len(q.queue)-1], nil
}

func (q *Queue[T]) Clear() {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.queue = make([]T, 0)
}

func (q *Queue[T]) Length() int {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	return len(q.queue)
}

func (q *Queue[T]) wakeupAWaitingDequeue() {
	select {
	case q.wakeupCh <- nil:
	default:
	}
}

func (q *Queue[T]) sleepUntilBeWokenUp(ctx context.Context) bool {
	select {
	case <-q.wakeupCh:
		return true
	case <-ctx.Done():
		return false
	}
}
