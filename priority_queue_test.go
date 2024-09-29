package priorityqueue_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	priorityqueue "github.com/xybor-x/priority_queue"
)

type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
	PriorityBackground
	PriorityValid
	PriorityInvalid
)

func Test_PriorityQueue_EnqueueDequeue(t *testing.T) {
	queue := priorityqueue.Default[int]()

	assert.NoError(t, queue.SetPriority(PriorityHigh, 0))
	assert.NoError(t, queue.SetPriority(PriorityMedium, 1))
	assert.NoError(t, queue.SetPriority(PriorityLow, 2))

	assert.NoError(t, queue.Enqueue(PriorityMedium, 2))
	assert.NoError(t, queue.Enqueue(PriorityLow, 1))
	assert.NoError(t, queue.Enqueue(PriorityHigh, 3))

	v, err := queue.Dequeue()
	assert.NoError(t, err)
	assert.Equal(t, 3, v.To())
	assert.Equal(t, PriorityHigh, v.Priority())

	v, err = queue.Dequeue()
	assert.NoError(t, err)
	assert.Equal(t, 2, v.To())
	assert.Equal(t, PriorityMedium, v.Priority())

	v, err = queue.Dequeue()
	assert.NoError(t, err)
	assert.Equal(t, 1, v.To())
	assert.Equal(t, PriorityLow, v.Priority())

	_, err = queue.Dequeue()
	assert.ErrorIs(t, err, priorityqueue.ErrEmpty)
}

func Test_PriorityQueue_SetInvalidPriorityLevel(t *testing.T) {
	queue := priorityqueue.Default[int]()

	assert.NoError(t, queue.SetPriority(PriorityHigh, 0))

	assert.ErrorIs(t, queue.SetPriority(PriorityHigh, 1), priorityqueue.ErrExistedLevel)
	assert.ErrorIs(t, queue.SetPriority(PriorityMedium, 0), priorityqueue.ErrExistedLevel)
	assert.Panics(t, func() { queue.SetPriority(PriorityLow, priorityqueue.DefaultMaxPriorityLevel+1) })
}

func Test_PriorityQueue_EnqueueAnInvalidPriority(t *testing.T) {
	queue := priorityqueue.Default[int]()

	assert.NoError(t, queue.SetPriority(PriorityValid, 0))

	assert.NoError(t, queue.Enqueue(PriorityValid, 1))

	assert.ErrorIs(t, queue.Enqueue(PriorityInvalid, 2), priorityqueue.ErrNotExistedLevel)
}

func Test_PriorityQueue_Aging(t *testing.T) {
	queue := priorityqueue.Default[int]()

	assert.NoError(t, queue.SetPriority(PriorityHigh, 0))
	assert.NoError(t, queue.SetPriority(PriorityMedium, 1))
	assert.NoError(t, queue.SetPriority(PriorityLow, 2))
	assert.NoError(t, queue.SetDefaultAgingTimeSlice(500*time.Millisecond))
	assert.NoError(t, queue.SetAgingTimeSlice(PriorityMedium, 100*time.Millisecond))

	assert.NoError(t, queue.Enqueue(PriorityLow, 1))
	assert.NoError(t, queue.Enqueue(PriorityMedium, 2))

	// After 100 miliseconds, the element 2 will be leveled up to the high
	// priority.
	time.Sleep(100 * time.Millisecond)

	assert.NoError(t, queue.Enqueue(PriorityHigh, 3))

	v, err := queue.Dequeue()
	assert.NoError(t, err)
	assert.Equal(t, 2, v.To())
	assert.Equal(t, PriorityMedium, v.OriginalPriority())

	// After next 400 miliseconds (total 500 miliseconds from enqueue), the
	// element 1 will be leveled up to the medium priority.
	time.Sleep(400 * time.Millisecond)

	assert.NoError(t, queue.Enqueue(PriorityMedium, 4))

	v, err = queue.Dequeue()
	assert.NoError(t, err)
	assert.Equal(t, 3, v.To())
	assert.Equal(t, PriorityHigh, v.OriginalPriority())

	v, err = queue.Dequeue()
	assert.NoError(t, err)
	assert.Equal(t, 1, v.To())
	assert.Equal(t, PriorityLow, v.OriginalPriority())

	v, err = queue.Dequeue()
	assert.NoError(t, err)
	assert.Equal(t, 4, v.To())
	assert.Equal(t, PriorityMedium, v.OriginalPriority())
}

func Test_PriorityQueue_WaitDequeue(t *testing.T) {
	queue := priorityqueue.Default[int]()

	assert.NoError(t, queue.SetPriority(PriorityHigh, 0))
	assert.NoError(t, queue.SetPriority(PriorityMedium, 1))
	assert.NoError(t, queue.SetPriority(PriorityLow, 2))

	assert.NoError(t, queue.Enqueue(PriorityLow, 1))
	assert.NoError(t, queue.Enqueue(PriorityMedium, 2))
	assert.NoError(t, queue.Enqueue(PriorityHigh, 3))

	ctx := context.Background()

	v, err := queue.WaitDequeue(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 3, v.To())
	assert.Equal(t, PriorityHigh, v.Priority())

	v, err = queue.WaitDequeue(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 2, v.To())
	assert.Equal(t, PriorityMedium, v.Priority())

	v, err = queue.WaitDequeue(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, v.To())
	assert.Equal(t, PriorityLow, v.Priority())

	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	_, err = queue.WaitDequeue(ctx)
	assert.ErrorIs(t, err, priorityqueue.ErrTimeout)

	go func() {
		time.Sleep(100 * time.Millisecond)
		queue.Enqueue(PriorityLow, 4)
	}()

	ctx = context.Background()
	v, err = queue.WaitDequeue(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 4, v.To())
	assert.Equal(t, PriorityLow, v.Priority())
}
