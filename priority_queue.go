package priorityqueue

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"
)

const NoAging = time.Duration(0)

// Element is a wrapper of an element in priority queue. It contains some
// metadata.
type Element[T any] struct {
	value T

	originalLevel    int
	originalPriority any

	level    int
	priority any

	createdAt time.Time
}

func newElement[T any](value T, level int, priority any) Element[T] {
	return Element[T]{
		value:            value,
		level:            level,
		priority:         priority,
		originalLevel:    level,
		originalPriority: priority,
		createdAt:        time.Now(),
	}
}

// To returns the inner value of the Element.
func (e Element[T]) To() T {
	return e.value
}

// OriginalLevel returns the original priority level value.
func (e Element[T]) OriginalLevel() int {
	return e.originalLevel
}

// OriginalPriority returns the original priority.
func (e Element[T]) OriginalPriority() any {
	return e.originalPriority
}

// Level returns the current priority level value (after aging).
func (e Element[T]) Level() int {
	return e.level
}

// Priority returns the current priority (after aging).
func (e Element[T]) Priority() any {
	return e.priority
}

func (e Element[T]) renew(level int, priority any) Element[T] {
	return Element[T]{
		value:            e.value,
		level:            level,
		priority:         priority,
		originalLevel:    e.originalLevel,
		originalPriority: e.originalPriority,
		createdAt:        time.Now(),
	}
}

// PriorityQueue implements a fast, thread-safe priority queue.
type PriorityQueue[T any] struct {
	mutex sync.RWMutex

	sortedLevels   []int
	level2priority map[int]any
	priority2level map[any]int

	commonAgingTimeSlice time.Duration
	agingTimeSlice       map[any]time.Duration

	mq *MultiLevelQueue[Element[T]]

	lastAging               time.Time
	agingInterval           time.Duration
	autoDetectAgingInterval bool
}

// New returns an empty PriorityQueue with the customized metadata.
func New[T any](core *MultiLevelQueue[Element[T]]) *PriorityQueue[T] {
	return &PriorityQueue[T]{
		mutex: sync.RWMutex{},

		sortedLevels:   make([]int, 0),
		level2priority: make(map[int]any),
		priority2level: make(map[any]int),

		commonAgingTimeSlice: 0,
		agingTimeSlice:       make(map[any]time.Duration),

		mq: core,

		lastAging:               time.Now(),
		agingInterval:           -1,
		autoDetectAgingInterval: true,
	}
}

// Default returns an empty PriorityQueue with the default MultiLevelQueue.
func Default[T any]() *PriorityQueue[T] {
	return New(
		DefaultMultiLevelQueue[Element[T]](),
	)
}

// SetDefaultAgingTimeSlice sets the default aging time slice for Priority
// levels which have not set the aging yet. If an element existed for more than
// this time slice, it will be moved to the next higher level.
func (pq *PriorityQueue[T]) SetDefaultAgingTimeSlice(timeslice time.Duration) error {
	if timeslice < 0 {
		return errors.New("invalid timeslice value")
	}

	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	pq.commonAgingTimeSlice = timeslice
	if pq.autoDetectAgingInterval && timeslice > 0 && (timeslice < pq.agingInterval || pq.agingInterval < 0) {
		pq.agingInterval = timeslice
	}

	return nil
}

// SetAgingTimeSlice sets the aging time slice for a specific level. If an
// element existed for more than this time slice, it will be moved to the next
// higher level.
func (pq *PriorityQueue[T]) SetAgingTimeSlice(priority any, timeslice time.Duration) error {
	if !pq.HasPriority(priority) {
		return fmt.Errorf("%w: priority %v", ErrNotExistedLevel, priority)
	}

	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	pq.agingTimeSlice[priority] = timeslice
	if pq.autoDetectAgingInterval && timeslice > 0 && (timeslice < pq.agingInterval || pq.agingInterval < 0) {
		pq.agingInterval = timeslice
	}

	return nil
}

// SetAgingInterval sets the interval. The PriorityQueue will check the aging
// every the interval passes. If the interval is unset, it will be chosen
// automatically (equal to the least aging timeslice of all levels)
func (pq *PriorityQueue[T]) SetAgingInterval(interval time.Duration) {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	pq.agingInterval = interval
	pq.autoDetectAgingInterval = false
}

// SetPriority assigns a new Priority with a level value.
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

	pq.sortedLevels = append(pq.sortedLevels, level)
	slices.Sort(pq.sortedLevels)

	return pq.mq.AddLevel(priority)
}

// HasPriority returns true if the Priority has existed in PriorityQueue.
func (pq *PriorityQueue[T]) HasPriority(priority any) bool {
	pq.mutex.RLock()
	defer pq.mutex.RUnlock()

	_, ok := pq.priority2level[priority]
	return ok
}

// HasLevel returns true if the level value has existed in PriorityQueue.
func (pq *PriorityQueue[T]) HasLevel(level int) bool {
	pq.mutex.RLock()
	defer pq.mutex.RUnlock()

	return pq.level2priority[level] != nil
}

// Enqueue pushes some values into PriorityQueue with a specific Priority.
func (pq *PriorityQueue[T]) Enqueue(priority any, values ...T) error {
	if !pq.HasPriority(priority) {
		return fmt.Errorf("%w: not found priority %d", ErrNotExistedLevel, priority)
	}

	if err := pq.aging(); err != nil {
		return err
	}

	elements := []Element[T]{}
	for i := range values {
		elements = append(elements, newElement(values[i], pq.priority2level[priority], priority))
	}

	return pq.mq.Enqueue(priority, elements...)
}

// Dequeue pops the first element which has the highest priority in
// PriorityQueue.
func (pq *PriorityQueue[T]) Dequeue() (Element[T], error) {
	var defaultElement Element[T]

	if err := pq.aging(); err != nil {
		return defaultElement, fmt.Errorf("failed to aging: %w", err)
	}

	for _, level := range pq.sortedLevels {
		priority := pq.level2priority[level]
		if priority == nil {
			continue
		}

		element, err := pq.mq.Dequeue(priority)
		if err != nil {
			if !errors.Is(err, ErrEmpty) {
				return defaultElement, err
			}

			continue
		}

		return element, nil
	}

	return defaultElement, ErrEmpty
}

// JustDequeue is a wrapper method of Dequeue, it returns nil when the queue is
// empty, and panics if got other errors.
func (pq *PriorityQueue[T]) JustDequeue() *Element[T] {
	v, err := pq.Dequeue()
	if err == nil {
		return &v
	}

	if errors.Is(err, ErrEmpty) {
		return nil
	}

	panic(err)
}

// WaitDequeue returns first element which has the highest priority. Differ from
// Dequeue, this method will blocks the current process if the queue is empty.
func (pq *PriorityQueue[T]) WaitDequeue(ctx context.Context) (Element[T], error) {
	v, err := pq.Dequeue()
	if !errors.Is(err, ErrEmpty) {
		return v, err
	}

	return pq.mq.WaitDequeueAll(ctx)
}

// JustWaitDequeue is a wrapper function of WaitDequeue, it returns nil if the
// timeout is reached, and panic if got other errors.
func (pq *PriorityQueue[T]) JustWaitDequeue(ctx context.Context) *Element[T] {
	v, err := pq.WaitDequeue(ctx)
	if err == nil {
		return &v
	}

	if errors.Is(err, ErrEmpty) || errors.Is(err, ErrTimeout) {
		return nil
	}

	panic(err)
}

// ForceAging do checking aging immediately no matter of the aging-interval.
func (pq *PriorityQueue[T]) ForceAging() error {
	pq.mutex.Lock()
	pq.lastAging = time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC)
	pq.mutex.Unlock()

	return pq.aging()
}

// Length returns the number of elements with a given priority.
func (pq *PriorityQueue[T]) Length(priority any) int {
	return pq.mq.Length(priority)
}

// TotalLength returns the total number of elements in PriorityQueue.
func (pq *PriorityQueue[T]) TotalLength() int {
	return pq.mq.TotalLength()
}

func (pq *PriorityQueue[T]) aging() error {
	defer func() {
		pq.mutex.Lock()
		defer pq.mutex.Unlock()

		pq.lastAging = time.Now()
	}()

	pq.mutex.RLock()
	lastAging := pq.lastAging
	pq.mutex.RUnlock()

	if pq.agingInterval < 0 || time.Now().Before(lastAging.Add(pq.agingInterval)) {
		return nil
	}

	for i := len(pq.sortedLevels) - 1; i >= 0; i-- {
		level := pq.sortedLevels[i]
		priority := pq.level2priority[level]

		if priority == nil {
			continue
		}

		timeslice := pq.commonAgingTimeSlice
		if value, ok := pq.agingTimeSlice[priority]; ok {
			timeslice = value
		}

		if timeslice == 0 {
			continue
		}

		higherPriority := pq.findHigherPriority(priority)
		if higherPriority == nil {
			break
		}

		higherLevel := pq.priority2level[higherPriority]

		for {
			element, err := pq.mq.DequeueIf(priority, func(e Element[T]) (bool, error) {
				return time.Now().After(e.createdAt.Add(timeslice)), nil
			})

			if errors.Is(err, ErrEmpty) || errors.Is(err, ErrNotMeetCondition) {
				break
			}

			if err := pq.mq.Enqueue(higherPriority, element.renew(higherLevel, higherPriority)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (pq *PriorityQueue[T]) findHigherPriority(priority any) any {
	level, ok := pq.priority2level[priority]
	if !ok {
		panic("not found priority")
	}

	for i := len(pq.sortedLevels) - 1; i >= 1; i-- {
		if pq.sortedLevels[i] == level {
			return pq.level2priority[pq.sortedLevels[i-1]]
		}
	}

	return nil
}
