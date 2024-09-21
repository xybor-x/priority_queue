package priorityqueue_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	priorityqueue "github.com/xybor-x/priority_queue"
)

func Test_DefaultQueue_Normal(t *testing.T) {
	queue := priorityqueue.NewDefaultQueue[int]()
	assert.NoError(t, queue.Enqueue(1))
	assert.NoError(t, queue.Enqueue(2))
	assert.NoError(t, queue.Enqueue(3))

	v, err := queue.Dequeue()
	assert.NoError(t, err)
	assert.Equal(t, 1, v)

	v, err = queue.Dequeue()
	assert.NoError(t, err)
	assert.Equal(t, 2, v)

	v, err = queue.Dequeue()
	assert.NoError(t, err)
	assert.Equal(t, 3, v)

	_, err = queue.Dequeue()
	assert.ErrorIs(t, err, priorityqueue.ErrEmpty)
}

func Test_DefaultQueue_FrontAndBackInNonEmptyQueue(t *testing.T) {
	queue := priorityqueue.NewDefaultQueue[int]()
	assert.NoError(t, queue.Enqueue(1))
	assert.NoError(t, queue.Enqueue(2))
	assert.NoError(t, queue.Enqueue(3))

	v, err := queue.Front()
	assert.NoError(t, err)
	assert.Equal(t, 1, v)

	v, err = queue.Back()
	assert.NoError(t, err)
	assert.Equal(t, 3, v)
}

func Test_DefaultQueue_FrontAndBackInEmptyQueue(t *testing.T) {
	queue := priorityqueue.NewDefaultQueue[int]()

	_, err := queue.Front()
	assert.ErrorIs(t, err, priorityqueue.ErrEmpty)

	_, err = queue.Back()
	assert.ErrorIs(t, err, priorityqueue.ErrEmpty)
}

func Test_DefaultQueue_LengthAndClearQueue(t *testing.T) {
	queue := priorityqueue.NewDefaultQueue[int]()
	assert.NoError(t, queue.Enqueue(1))
	assert.NoError(t, queue.Enqueue(2))
	assert.NoError(t, queue.Enqueue(3))

	assert.Equal(t, 3, queue.Length())

	queue.Clear()

	assert.Equal(t, 0, queue.Length())
}

func Test_DefaultQueue_WaitDequeue(t *testing.T) {
	queue := priorityqueue.NewDefaultQueue[int]()
	assert.NoError(t, queue.Enqueue(1))
	assert.NoError(t, queue.Enqueue(2))
	assert.NoError(t, queue.Enqueue(3))

	v, err := queue.WaitDequeue(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, v)

	v, err = queue.WaitDequeue(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 2, v)

	v, err = queue.WaitDequeue(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 3, v)

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	_, err = queue.WaitDequeue(ctx)
	assert.ErrorIs(t, err, priorityqueue.ErrTimeout)
}

func Test_DefaultQueue_DequeueIfFail(t *testing.T) {
	queue := priorityqueue.NewDefaultQueue[int]()
	assert.NoError(t, queue.Enqueue(1))

	v, err := queue.DequeueIf(func(t int) (bool, error) { return false, nil })
	assert.ErrorIs(t, err, priorityqueue.ErrNotMeetCondition)
	assert.Equal(t, 1, v)
}

func Test_DefaultQueue_DequeueIfError(t *testing.T) {
	queue := priorityqueue.NewDefaultQueue[int]()
	assert.NoError(t, queue.Enqueue(1))

	expectedErr := errors.New("wrong")

	_, err := queue.DequeueIf(func(t int) (bool, error) { return false, expectedErr })
	assert.ErrorIs(t, err, expectedErr)
}

func Test_DefaultQueue_EnqueueAfterWaitDequeue(t *testing.T) {
	queue := priorityqueue.NewDefaultQueue[int]()
	assert.NoError(t, queue.Enqueue(1))

	v, err := queue.WaitDequeue(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, v)

	go func() {
		time.Sleep(500 * time.Millisecond)
		assert.NoError(t, queue.Enqueue(2))
	}()

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	v, err = queue.WaitDequeue(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 2, v)
}

func Test_DefaultQueue_MultipleWaitDequeue(t *testing.T) {
	queue := priorityqueue.NewDefaultQueue[int]()

	go func() {
		time.Sleep(500 * time.Millisecond)
		assert.NoError(t, queue.Enqueue(1))
	}()

	wg := sync.WaitGroup{}
	totalSuccess := int32(0)

	for i := range 3 {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()

			v, err := queue.WaitDequeue(ctx)

			if err == nil {
				assert.Equal(t, 1, v)
				atomic.AddInt32(&totalSuccess, 1)
			} else {
				assert.ErrorIs(t, err, priorityqueue.ErrTimeout)
			}
		}(i)
	}

	wg.Wait()

	// Only one WaitDequeue is success, other ones should got a timeout error.
	assert.Equal(t, int32(1), totalSuccess)
}

func Test_DefaultQueue_DequeueIf(t *testing.T) {
	queue := priorityqueue.NewDefaultQueue[int]()
	assert.NoError(t, queue.Enqueue(1))
	assert.NoError(t, queue.Enqueue(2))
	assert.NoError(t, queue.Enqueue(3))

	v, err := queue.DequeueIf(func(t int) (bool, error) { return t < 3, nil })
	assert.NoError(t, err)
	assert.Equal(t, 1, v)

	v, err = queue.DequeueIf(func(t int) (bool, error) { return t < 3, nil })
	assert.NoError(t, err)
	assert.Equal(t, 2, v)

	v, err = queue.DequeueIf(func(t int) (bool, error) { return t < 3, nil })
	assert.ErrorIs(t, err, priorityqueue.ErrNotMeetCondition)
	assert.Equal(t, 3, v)
}

func Test_DefaultQueue_WaitDequeueIf(t *testing.T) {
	queue := priorityqueue.NewDefaultQueue[int]()

	go func() {
		time.Sleep(500 * time.Millisecond)
		assert.NoError(t, queue.Enqueue(1))
		assert.NoError(t, queue.Enqueue(1))
	}()

	wg := sync.WaitGroup{}
	totalSuccess := int32(0)

	for i := range 3 {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()

			v, err := queue.WaitDequeueIf(ctx, func(t int) (bool, error) {
				return t == index, nil
			}, nil)

			if err == nil {
				assert.Equal(t, 1, v)
				atomic.AddInt32(&totalSuccess, 1)
			} else {
				assert.ErrorIs(t, err, priorityqueue.ErrNotMeetCondition)
			}
		}(i)
	}

	wg.Wait()

	// Only one WaitDequeue is success, other ones is not meet condition.
	assert.Equal(t, int32(1), totalSuccess)
}

func Test_DefaultQueue_WaitDequeueIfRetrigger(t *testing.T) {
	queue := priorityqueue.NewDefaultQueue[time.Time]()
	assert.NoError(t, queue.Enqueue(time.Now().Add(200*time.Millisecond)))

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, err := queue.WaitDequeueIf(ctx,
		func(t time.Time) (bool, error) {
			return time.Now().After(t), nil
		},
		func(t time.Time) <-chan time.Time {
			// Retrigger the checking when suitable.
			return time.After(time.Since(t))
		},
	)

	assert.NoError(t, err)
}

func Test_DefaultQueue_WaitDequeueIfRetriggerButTimeout(t *testing.T) {
	queue := priorityqueue.NewDefaultQueue[time.Time]()
	assert.NoError(t, queue.Enqueue(time.Now().Add(3*time.Second)))

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()

	_, err := queue.WaitDequeueIf(ctx,
		func(t time.Time) (bool, error) {
			return time.Now().After(t), nil
		},
		func(t time.Time) <-chan time.Time {
			// Retrigger the checking when suitable.
			return time.After(time.Since(t))
		},
	)

	assert.ErrorIs(t, err, priorityqueue.ErrTimeout)
}
