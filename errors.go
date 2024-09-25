package priorityqueue

import (
	"errors"
)

var ErrExistedLevel = errors.New("level has already existed")
var ErrNotExistedLevel = errors.New("not found level")
var ErrEmpty = errors.New("queue is empty")
var ErrTimeout = errors.New("timeout")
var ErrNotMeetCondition = errors.New("not meet condition")
