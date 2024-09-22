package priorityqueue

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

const NoAging = time.Duration(0)
const DefaultMaxPriorityLevel = 1000

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

func (e Element[T]) To() T {
	return e.value
}

func (e Element[T]) OriginalLevel() int {
	return e.originalLevel
}

func (e Element[T]) OriginalPriority() any {
	return e.originalPriority
}

func (e Element[T]) Level() int {
	return e.level
}

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

type PriorityQueue[T any] struct {
	mutex sync.RWMutex

	level2priority []any
	priority2level map[any]int

	commonAgingTimeSlice time.Duration
	agingTimeSlice       map[any]time.Duration

	mq *MultiLevelQueue[Element[T]]

	lastAging               time.Time
	agingInterval           time.Duration
	autoDetectAgingInterval bool
}

func New[T any](maxPriorityLevel int, core *MultiLevelQueue[Element[T]]) *PriorityQueue[T] {
	return &PriorityQueue[T]{
		mutex: sync.RWMutex{},

		level2priority: make([]any, maxPriorityLevel+1),
		priority2level: make(map[any]int),

		commonAgingTimeSlice: 0,
		agingTimeSlice:       make(map[any]time.Duration),

		mq: core,

		lastAging:               time.Now(),
		agingInterval:           -1,
		autoDetectAgingInterval: true,
	}
}

func Default[T any]() *PriorityQueue[T] {
	return New(
		DefaultMaxPriorityLevel,
		DefaultMultiLevelQueue[Element[T]](),
	)
}

func (pq *PriorityQueue[T]) SetCommonAgingTimeSlice(timeslice time.Duration) error {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	pq.commonAgingTimeSlice = timeslice
	if pq.autoDetectAgingInterval && timeslice > 0 && (timeslice < pq.agingInterval || pq.agingInterval < 0) {
		pq.agingInterval = timeslice
	}

	return nil
}

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

func (pq *PriorityQueue[T]) SetAgingInterval(interval time.Duration) {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	pq.agingInterval = interval
	pq.autoDetectAgingInterval = false
}

func (pq *PriorityQueue[T]) SetPriority(priority any, level int) error {
	if level >= len(pq.level2priority) {
		panic(fmt.Sprintf("exceed maxmium value of level (%d)", len(pq.level2priority)))
	}

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
	return pq.mq.AddLevel(priority)
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

func (pq *PriorityQueue[T]) Dequeue() (Element[T], error) {
	var defaultElement Element[T]

	if err := pq.aging(); err != nil {
		return defaultElement, fmt.Errorf("%w: %w", ErrAging, err)
	}

	for _, priority := range pq.level2priority {
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

func (pq *PriorityQueue[T]) WaitDequeue(ctx context.Context) (Element[T], error) {
	v, err := pq.Dequeue()
	if !errors.Is(err, ErrEmpty) {
		return v, err
	}

	return pq.mq.WaitDequeueAll(ctx)
}

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

func (pq *PriorityQueue[T]) ForceAging() error {
	pq.mutex.Lock()
	pq.lastAging = time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC)
	pq.mutex.Unlock()

	return pq.aging()
}

func (pq *PriorityQueue[T]) Length(priority any) int {
	return pq.mq.Length(priority)
}

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

	for level := len(pq.level2priority) - 1; level > 0; level-- {
		priority := pq.level2priority[level]

		if priority == nil {
			continue
		}

		timeslice := pq.commonAgingTimeSlice
		if value, ok := pq.agingTimeSlice[priority]; ok {
			timeslice = value
		}

		if timeslice <= 0 {
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

	for i := level - 1; i >= 0; i-- {
		if pq.level2priority[i] != nil {
			return pq.level2priority[i]
		}
	}

	return nil
}
