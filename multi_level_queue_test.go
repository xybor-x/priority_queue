package priorityqueue_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	priorityqueue "github.com/xybor-x/priority_queue"
)

type Level int

const (
	LevelLow Level = iota
	LevelMedium
	LevelHigh
	LevelValid
	LevelInvalid
)

func Test_DefaultMultiLevelQueue_AddSameLevel(t *testing.T) {
	mqueue := priorityqueue.DefaultMultiLevelQueue[int]()
	assert.NoError(t, mqueue.AddLevel(1))
	assert.ErrorIs(t, mqueue.AddLevel(1), priorityqueue.ErrExistedLevel)
}

func Test_DefaultMultiLevelQueue_AddSameLevelValueButDiffType(t *testing.T) {
	mqueue := priorityqueue.DefaultMultiLevelQueue[int]()
	assert.NoError(t, mqueue.AddLevel(int(1)))
	assert.NoError(t, mqueue.AddLevel(int32(1)))
}

func Test_DefaultMultiLevelQueue_EnqueueDequeue(t *testing.T) {
	mqueue := priorityqueue.DefaultMultiLevelQueue[int]()
	mqueue.AddLevel(LevelLow)
	mqueue.AddLevel(LevelMedium)
	mqueue.AddLevel(LevelHigh)

	assert.NoError(t, mqueue.Enqueue(LevelLow, 1))
	assert.NoError(t, mqueue.Enqueue(LevelLow, 2))
	assert.NoError(t, mqueue.Enqueue(LevelMedium, 3))
	assert.NoError(t, mqueue.Enqueue(LevelHigh, 4))

	v, err := mqueue.Dequeue(LevelLow)
	assert.NoError(t, err)
	assert.Equal(t, 1, v)

	v, err = mqueue.Dequeue(LevelHigh)
	assert.NoError(t, err)
	assert.Equal(t, 4, v)

	v, err = mqueue.Dequeue(LevelMedium)
	assert.NoError(t, err)
	assert.Equal(t, 3, v)

	v, err = mqueue.Dequeue(LevelLow)
	assert.NoError(t, err)
	assert.Equal(t, 2, v)
}

func Test_DefaultMultiLevelQueue_EnqueueAnInvalidLevel(t *testing.T) {
	mqueue := priorityqueue.DefaultMultiLevelQueue[int]()
	mqueue.AddLevel(LevelValid)

	assert.NoError(t, mqueue.Enqueue(LevelValid, 1))
	assert.ErrorIs(t, mqueue.Enqueue(LevelInvalid, 1), priorityqueue.ErrNotExistedLevel)
}

func Test_DefaultMultiLevelQueue_DequeueAnEmtpyQueue(t *testing.T) {
	mqueue := priorityqueue.DefaultMultiLevelQueue[int]()
	mqueue.AddLevel(LevelLow)
	mqueue.AddLevel(LevelMedium)
	mqueue.AddLevel(LevelHigh)

	_, err := mqueue.Dequeue(LevelLow)
	assert.ErrorIs(t, err, priorityqueue.ErrEmpty)

	_, err = mqueue.Dequeue(LevelHigh)
	assert.ErrorIs(t, err, priorityqueue.ErrEmpty)

	_, err = mqueue.Dequeue(LevelMedium)
	assert.ErrorIs(t, err, priorityqueue.ErrEmpty)
}

func Test_DefaultMultiLevelQueue_DequeueAnInvalidLevel(t *testing.T) {
	mqueue := priorityqueue.DefaultMultiLevelQueue[int]()
	mqueue.AddLevel(LevelValid)

	_, err := mqueue.Dequeue(LevelInvalid)
	assert.ErrorIs(t, err, priorityqueue.ErrNotExistedLevel)
}

func Test_DefaultMultiLevelQueue_Length(t *testing.T) {
	mqueue := priorityqueue.DefaultMultiLevelQueue[int]()
	mqueue.AddLevel(LevelLow)
	mqueue.AddLevel(LevelMedium)
	mqueue.AddLevel(LevelHigh)

	assert.NoError(t, mqueue.Enqueue(LevelLow, 1))
	assert.NoError(t, mqueue.Enqueue(LevelLow, 2))
	assert.NoError(t, mqueue.Enqueue(LevelMedium, 3))
	assert.NoError(t, mqueue.Enqueue(LevelHigh, 4))

	assert.Equal(t, 4, mqueue.TotalLength())
	assert.Equal(t, 2, mqueue.Length(LevelLow))
	assert.Equal(t, 1, mqueue.Length(LevelMedium))
	assert.Equal(t, 1, mqueue.Length(LevelHigh))

	v, err := mqueue.Dequeue(LevelLow)
	assert.NoError(t, err)
	assert.Equal(t, 1, v)

	v, err = mqueue.Dequeue(LevelHigh)
	assert.NoError(t, err)
	assert.Equal(t, 4, v)

	v, err = mqueue.Dequeue(LevelMedium)
	assert.NoError(t, err)
	assert.Equal(t, 3, v)

	assert.Equal(t, 1, mqueue.TotalLength())
	assert.Equal(t, 1, mqueue.Length(LevelLow))
	assert.Equal(t, 0, mqueue.Length(LevelMedium))
	assert.Equal(t, 0, mqueue.Length(LevelHigh))
}

func Test_DefaultMultiLevelQueue_LengthOfInvalidLevel(t *testing.T) {
	mqueue := priorityqueue.DefaultMultiLevelQueue[int]()
	mqueue.AddLevel(LevelValid)

	assert.NoError(t, mqueue.Enqueue(LevelValid, 1))

	assert.Equal(t, 1, mqueue.Length(LevelValid))
	assert.Equal(t, 0, mqueue.Length(LevelInvalid))
}

func Test_DefaultMultiLevelQueue_WaitDequeue(t *testing.T) {
	mqueue := priorityqueue.DefaultMultiLevelQueue[int]()
	mqueue.AddLevel(LevelLow)
	mqueue.AddLevel(LevelMedium)

	assert.NoError(t, mqueue.Enqueue(LevelLow, 1))
	assert.NoError(t, mqueue.Enqueue(LevelMedium, 2))

	ctx := context.Background()

	v, err := mqueue.WaitDequeue(ctx, LevelLow)
	assert.NoError(t, err)
	assert.Equal(t, 1, v)

	v, err = mqueue.WaitDequeue(ctx, LevelMedium)
	assert.NoError(t, err)
	assert.Equal(t, 2, v)

	go func() {
		time.Sleep(100 * time.Millisecond)
		assert.NoError(t, mqueue.Enqueue(LevelLow, 3))
	}()

	v, err = mqueue.WaitDequeue(ctx, LevelLow)
	assert.NoError(t, err)
	assert.Equal(t, 3, v)
}

func Test_DefaultMultiLevelQueue_WaitDequeueInvalidLevel(t *testing.T) {
	mqueue := priorityqueue.DefaultMultiLevelQueue[int]()
	mqueue.AddLevel(LevelValid)

	assert.NoError(t, mqueue.Enqueue(LevelValid, 1))

	ctx := context.Background()

	v, err := mqueue.WaitDequeue(ctx, LevelValid)
	assert.NoError(t, err)
	assert.Equal(t, 1, v)

	_, err = mqueue.WaitDequeue(ctx, LevelInvalid)
	assert.ErrorIs(t, err, priorityqueue.ErrNotExistedLevel)
}

func Test_DefaultMultiLevelQueue_WaitDequeueTimeout(t *testing.T) {
	mqueue := priorityqueue.DefaultMultiLevelQueue[int]()
	mqueue.AddLevel(LevelLow)
	mqueue.AddLevel(LevelMedium)

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	_, err := mqueue.WaitDequeue(ctx, LevelLow)
	assert.ErrorIs(t, err, priorityqueue.ErrTimeout)
}

func Test_DefaultMultiLevelQueue_WaitDequeueAll(t *testing.T) {
	mqueue := priorityqueue.DefaultMultiLevelQueue[int]()
	mqueue.AddLevel(LevelLow)
	mqueue.AddLevel(LevelMedium)
	mqueue.AddLevel(LevelHigh)

	assert.NoError(t, mqueue.Enqueue(LevelLow, 1))
	assert.NoError(t, mqueue.Enqueue(LevelMedium, 2))

	ctx := context.Background()

	v, err := mqueue.WaitDequeueAll(ctx)
	assert.NoError(t, err)
	assert.True(t, v == 1 || v == 2)

	v, err = mqueue.WaitDequeueAll(ctx)
	assert.NoError(t, err)
	assert.True(t, v == 1 || v == 2)

	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	_, err = mqueue.WaitDequeueAll(ctx)
	assert.ErrorIs(t, err, priorityqueue.ErrTimeout)

	go func() {
		time.Sleep(100 * time.Millisecond)
		mqueue.Enqueue(LevelMedium, 3)
	}()

	ctx = context.Background()
	v, err = mqueue.WaitDequeueAll(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 3, v)
}
