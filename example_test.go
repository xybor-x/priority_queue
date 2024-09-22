package priorityqueue_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	priorityqueue "github.com/xybor-x/priority_queue"
)

type ExamplePriority int

const (
	Urgent ExamplePriority = iota
	Necessary
	Lazy
	Background
)

func Test_Example_Dequeue(t *testing.T) {
	pqueue := priorityqueue.Default[int]()

	// Setup priority level.
	pqueue.SetPriority(Urgent, 0)
	pqueue.SetPriority(Necessary, 5)
	pqueue.SetPriority(Lazy, 10)
	pqueue.SetPriority(Background, 100)

	// Specially for the Necessary level, after 500 miliseconds since the element is
	// enqueued into the queue, its priority level will be raised up.
	pqueue.SetAgingTimeSlice(Necessary, 500*time.Millisecond)

	// Enqueue three elements to Lazy level.
	pqueue.Enqueue(Lazy, 4, 5, 6)

	// Enqueue an element to Urgent level.
	pqueue.Enqueue(Urgent, 1)

	// Enqueue two elements to Necessary level.
	pqueue.Enqueue(Necessary, 2, 3)

	// Use Dequeue() to get the first highest priority element.
	v, err := pqueue.Dequeue()
	assert.NoError(t, err)
	assert.Equal(t, 1, v.To())

	// Or use JustDequeue(), a shortcut of Dequeue().
	assert.Equal(t, 2, pqueue.JustDequeue().To())
	assert.Equal(t, 3, pqueue.JustDequeue().To())
	assert.Equal(t, 4, pqueue.JustDequeue().To())
	assert.Equal(t, 5, pqueue.JustDequeue().To())
	assert.Equal(t, 6, pqueue.JustDequeue().To())
	assert.Nil(t, pqueue.JustDequeue())
}

func Test_Example_WaitDequeue(t *testing.T) {
	pqueue := priorityqueue.Default[int]()

	// Setup priority level.
	pqueue.SetPriority(Urgent, 0)
	pqueue.SetPriority(Necessary, 5)
	pqueue.SetPriority(Lazy, 10)
	pqueue.SetPriority(Background, 100)

	// Specially for the Necessary level, after 500 miliseconds since the element is
	// enqueued into the queue, its priority level will be raised up.
	pqueue.SetAgingTimeSlice(Necessary, 500*time.Millisecond)

	// Enqueue three element to Lazy level.
	pqueue.Enqueue(Lazy, 3)

	// Enqueue an element to Urgent level.
	pqueue.Enqueue(Urgent, 1)

	// Enqueue two elements to Necessary level.
	pqueue.Enqueue(Necessary, 2)

	// Use WaitDequeue to dequeue the queue, but it will block if the queue is
	// empty.
	ctx := context.Background()
	v, err := pqueue.WaitDequeue(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, v.To())

	// Or use the shortcut JustWaitDequeue.
	assert.Equal(t, 2, pqueue.JustWaitDequeue(ctx).To())
	assert.Equal(t, 3, pqueue.JustWaitDequeue(ctx).To())

	// Use context to avoid infinite blocking.
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	_, err = pqueue.WaitDequeue(ctx)
	assert.ErrorIs(t, err, priorityqueue.ErrTimeout)

	// JustWaitDequeue also returns nil if the queue is empty and timeout.
	ctx = context.WithoutCancel(ctx)
	ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	assert.Nil(t, pqueue.JustWaitDequeue(ctx))

	go func() {
		time.Sleep(100 * time.Millisecond)
		pqueue.Enqueue(Lazy, 4)
	}()

	ctx = context.WithoutCancel(ctx)
	ctx, cancel = context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	assert.Equal(t, 4, pqueue.JustWaitDequeue(ctx).To())
}
