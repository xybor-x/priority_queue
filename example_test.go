package priorityqueue_test

import (
	"testing"
)

func Test_Example(t *testing.T) {
	// pqueue := priorityqueue.New[int]()

	// Setup three priority level, including "urgent", "necessary", "lazy".
	// pqueue.SetLevel("urgent", 0)
	// pqueue.SetLevel("necessary", 5)
	// pqueue.SetLevel("lazy", 10)

	// After the queue is popped 10 times, priority level of all elements will
	// be raised up.
	// pqueue.SetCommonAging(priorityqueue.Times(10))

	// Specially for the "necessary" level, after 5 minutes since the element is
	// pushed into the queue, its priority level will be raised up.
	// pqueue.SetAging("necessary", priority.Duration(5 * times.Minute))

	// Push three element to "lazy" level.
	// pqueue.Push("lazy", 4, 5, 6)

	// Push an element to "urgent" level.
	// pqueue.Push("urgent", 1)

	// Push two elements to "necessary" level.
	// pqueue.Push("necessary", 2, 3)

	// assert.Equal(t, pqueue.Pop(), 1)
	// assert.Equal(t, pqueue.Pop(), 2)
	// assert.Equal(t, pqueue.PopWithLevel("lazy"), 4)
	// assert.Equal(t, pqueue.Pop(), 3)
	// assert.Equal(t, pqueue.Pop(), 5)
	// assert.Equal(t, pqueue.RemainingSize(), 1)
}
