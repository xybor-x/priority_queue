[![xybor founder](https://img.shields.io/badge/xybor-huykingsofm-red)](https://github.com/huykingsofm)
[![Go Reference](https://pkg.go.dev/badge/github.com/xybor-x/priority_queue.svg)](https://pkg.go.dev/github.com/xybor-x/priority_queue)
[![GitHub Repo stars](https://img.shields.io/github/stars/xybor-x/priority_queue?color=yellow)](https://github.com/xybor-x/priority_queue)
[![GitHub top language](https://img.shields.io/github/languages/top/xybor-x/priority_queue?color=lightblue)](https://go.dev/)
[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/xybor-x/priority_queue)](https://go.dev/blog/go1.18)
[![GitHub release (release name instead of tag name)](https://img.shields.io/github/v/release/xybor-x/priority_queue?include_prereleases)](https://github.com/xybor-x/priority_queue/releases/latest)
[![Codacy Badge](https://app.codacy.com/project/badge/Grade/2fe10924ab114a08bbbab5d583fd610c)](https://www.codacy.com/gh/xybor-x/priority_queue/dashboard?utm_source=github.com&utm_medium=referral&utm_content=xybor-x/priority_queue&utm_campaign=Badge_Grade)
[![Codacy Badge](https://app.codacy.com/project/badge/Coverage/2fe10924ab114a08bbbab5d583fd610c)](https://www.codacy.com/gh/xybor-x/priority_queue/dashboard?utm_source=github.com&utm_medium=referral&utm_content=xybor-x/priority_queue&utm_campaign=Badge_Grade)
[![Go Report](https://goreportcard.com/badge/github.com/xybor-x/priority_queue)](https://goreportcard.com/report/github.com/xybor-x/priority_queue)

# Introduction

An fast, thread-safe implementation of Priority Queue in Golang

# Get started

```golang
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
```
