package bootstrap

import (
	"context"
	"sync"
)

// Runnable is a interface that can be run and shutdown
type Runnable interface {
	OnStart(ctx context.Context) error
	OnStop(ctx context.Context) error
}

type Executable interface {
	Execute(ctx context.Context) error
}

type ResourceExecutable[T Executable] struct {
	once   sync.Once
	value  T
	setter func(Executable)
}

func (r *ResourceExecutable[T]) Resolve(init func() T) T {
	r.once.Do(func() {
		r.value = init()
	})
	return r.value
}

// Closable is a interface that can be closed
type Closable interface {
	Close() error
}

// Resource is a generic resource that can be initialized and retrieved
type Resource[T any] struct {
	once  sync.Once
	value T
}

func (r *Resource[T]) Resolve(init func() T) T {
	r.once.Do(func() {
		r.value = init()
	})
	return r.value
}

// ResourceRunnable is a resource that can be run and shutdown
type ResourceRunnable[T Runnable] struct {
	once   sync.Once
	value  T
	setter func(Runnable)
}

// Get returns the value, initializing it if necessary
func (l *ResourceRunnable[T]) Resolve(init func() T) T {
	l.once.Do(func() {
		l.value = init()
		if l.setter != nil {
			l.setter(l.value)
		}
	})
	return l.value
}

// ResourceClosable is a resource that can be closed
type ResourceClosable[T Closable] struct {
	once       sync.Once
	value      T
	err        error
	resolved   bool // Track if resource has been initialized
	registered bool // Track if closer has been registered
	mu         sync.Mutex
	register   func(func() error)
}

// Resolve returns the value, initializing it if necessary
func (l *ResourceClosable[T]) Resolve(init func() T) T {
	l.once.Do(func() {
		l.value = init()
		l.resolved = true
	})

	// Try to register after initialization (outside once.Do to allow late registration)
	l.tryRegister()

	return l.value
}

// tryRegister attempts to register the closer if not already registered
func (l *ResourceClosable[T]) tryRegister() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.resolved && l.register != nil && !l.registered {
		l.registered = true
		l.register(func() error {
			return l.value.Close()
		})
	}
}

// EnsureRegistered registers the closer if the resource was resolved before register was set
// This is called by Bootstrap after setting the register function
// Must be exported for reflection to work
func (l *ResourceClosable[T]) EnsureRegistered() {
	l.tryRegister()
}
