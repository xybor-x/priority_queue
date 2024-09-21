package priorityqueue

import "time"

type Element[T any] struct {
	value    T
	level    int
	priority any
	created  time.Time
}

func newElement[T any](value T, level int, priority any) Element[T] {
	return Element[T]{
		value:    value,
		level:    level,
		priority: priority,
		created:  time.Now(),
	}
}

func (e Element[T]) Value() T {
	return e.value
}

func (e Element[T]) Level() int {
	return e.level
}

func (e Element[T]) Priority() any {
	return e.priority
}

func (e Element[T]) ShouldLevelUp(timeslice time.Duration) bool {
	return time.Now().After(e.created.Add(timeslice))
}
