package priorityqueue

import (
	"fmt"
	"sync"
	"time"
)

const DefaultMaxPriorityLevel = 1024

type PriorityQueue[T any] struct {
	mutex sync.RWMutex

	level2priority []any
	priority2level map[any]int

	commonTimeSlice   time.Duration
	specificTimeSlice map[any]time.Duration

	mq *MultiLevelQueue[Element[T]]
}

func NewWithCustom[T any](maxPriorityLevel int, queuer func() Queuer[Element[T]]) *PriorityQueue[T] {
	return &PriorityQueue[T]{
		mutex: sync.RWMutex{},

		level2priority: make([]any, maxPriorityLevel),
		priority2level: make(map[any]int),

		commonTimeSlice:   0,
		specificTimeSlice: make(map[any]time.Duration),

		mq: newMultilevelQueue[Element[T]](queuer),
	}
}

func New[T any]() *PriorityQueue[T] {
	return NewWithCustom(
		DefaultMaxPriorityLevel,
		func() Queuer[Element[T]] { return NewDefaultQueue[Element[T]]() },
	)
}

func (pq *PriorityQueue[T]) SetCommonAging(timeslice time.Duration) error {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	pq.commonTimeSlice = timeslice

	return nil
}

func (pq *PriorityQueue[T]) SetSpecificAging(priority any, timeslice time.Duration) error {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	pq.specificTimeSlice[priority] = timeslice

	return nil
}

func (pq *PriorityQueue[T]) SetPriority(priority any, level int) error {
	if pq.HasLevel(level) {
		return fmt.Errorf("%w: %d", ErrExistedLevel, level)
	}

	if pq.HasPriority(priority) {
		return fmt.Errorf("%w: priority %v", ErrExistedLevel, priority)
	}

	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	pq.level2priority[level] = priority
	pq.priority2level[priority] = level

	return nil
}

func (pq *PriorityQueue[T]) HasPriority(priority any) bool {
	pq.mutex.RLock()
	defer pq.mutex.RUnlock()

	_, ok := pq.priority2level[priority]
	return ok
}

func (pq *PriorityQueue[T]) HasLevel(level int) bool {
	pq.mutex.RLock()
	defer pq.mutex.RUnlock()

	return pq.level2priority[level] != nil
}

func (pq *PriorityQueue[T]) Enqueue(priority any, value T) error {
	if !pq.HasPriority(priority) {
		return fmt.Errorf("%w: not found priority %d", ErrNotExistedLevel, priority)
	}

	return pq.mq.Enqueue(
		priority,
		newElement(
			value,
			pq.priority2level[priority],
			priority,
		),
	)
}

func (pq *PriorityQueue[T]) Dequeue() (Element[T], error) {
	var defaultElement Element[T]
	for _, priority := range pq.level2priority {
		if priority == nil {
			continue
		}

		l, err := pq.mq.Length(priority)
		if err != nil {
			return defaultElement, err
		}

		if l > 0 {
			return pq.mq.Dequeue(priority)
		}
	}

	return defaultElement, ErrEmpty
}
