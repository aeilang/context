package mycontext

import (
	"errors"
	"sync"
	"time"
)

type Context interface {
	// ok为true时， 有截止时间，为deadline
	// ok为false时， 没有截止时间。
	Deadline() (deadline time.Time, ok bool)

	// context取消时, v := <-Done(), v为struct{}{}
	// context没取消，v := <-Done() 一直堵塞,等待
	Done() <-chan struct{}

	// context没取消，返回nil
	// context取消了，返回非nil的error, 表明取消的原因，有主动取消和截止到期两种
	Err() error

	// 如果key相等，返回对应的val,
	// 如果都不相等，返回nil
	Value(key any) any
}

type emptyCtx struct{}

func (emptyCtx) Deadline() (deadline time.Time, ok bool) {
	return
}

func (emptyCtx) Done() <-chan struct{} {
	return nil
}

func (emptyCtx) Err() error {
	return nil
}

func (emptyCtx) Value(key any) any {
	return nil
}

func Background() Context {
	return emptyCtx{}
}

func TODO() Context {
	return emptyCtx{}
}

type cancelCtx struct {
	Context

	once     sync.Once
	mu       sync.Mutex
	done     chan struct{}
	children map[canceler]struct{}
	err      error
}

type canceler interface {
	cancel(err error)
	Done() <-chan struct{}
}

func (c *cancelCtx) Done() <-chan struct{} {
	return c.done
}

func (c *cancelCtx) Err() error {
	return c.err
}

func (c *cancelCtx) cancel(err error) {
	c.once.Do(func() {
		close(c.done)
		c.mu.Lock()
		c.err = err
		for child := range c.children {
			child.cancel(err)
		}
		c.mu.Unlock()
	})
}

var ErrCancel = errors.New("主动取消")
var ErrDeadline = errors.New("到期了")

func WithCancel(parent Context) (Context, CancelFunc) {
	if parent == nil {
		panic("not nil")
	}

	child := &cancelCtx{
		Context:  parent,
		done:     make(chan struct{}),
		children: make(map[canceler]struct{}),
	}

	propagateCancel(parent, child)

	return child, func() {
		child.cancel(ErrCancel)
	}
}

type CancelFunc func()

type timerCtx struct {
	cancelCtx
	timer    *time.Timer
	deadline time.Time
}

func (t *timerCtx) Deadline() (deadline time.Time, ok bool) {
	return t.deadline, true
}

func propagateCancel(parent Context, child canceler) {
	if parent, ok := parent.(*cancelCtx); ok {
		parent.mu.Lock()
		parent.children[child] = struct{}{}
		parent.mu.Unlock()
	}

	go func() {
		select {
		case <-parent.Done():
			child.cancel(parent.Err())
		case <-child.Done():
		}
	}()
}

func WithDeadline(parent Context, deadline time.Time) (Context, CancelFunc) {
	child := &timerCtx{
		cancelCtx: cancelCtx{
			Context:  parent,
			done:     make(chan struct{}),
			children: make(map[canceler]struct{}),
		},
		deadline: deadline,
	}

	propagateCancel(parent, child)

	if time.Now().After(deadline) {
		child.cancel(ErrDeadline)
	}

	child.timer = time.AfterFunc(time.Until(deadline), func() {
		child.cancel(ErrDeadline)
	})

	return child, func() {
		child.cancel(ErrCancel)
	}
}

func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc) {
	return WithDeadline(parent, time.Now().Add(timeout))
}

type valCtx struct {
	Context
	key, val any
}

func WithValue(parent Context, key, value any) Context {
	if parent == nil {
		panic("not nil")
	}

	ctx := &valCtx{
		Context: parent,
		key:     key,
		val:     value,
	}

	return ctx
}

func (v *valCtx) Value(key any) any {
	if v.key == key {
		return key
	}

	return value(v.Context, key)
}

func value(c Context, key any) any {
	for {
		switch ctx := c.(type) {
		case *valCtx:
			if ctx.key == key {
				return ctx.val
			}

			c = ctx.Context

		case *emptyCtx:
			return nil
		case *cancelCtx:
			c = ctx.Context
		case *timerCtx:
			c = ctx.Context
		default:
			return ctx.Value(key)
		}
	}
}
