// Package closer provides a mechanism for graceful shutdown by managing multiple closing functions.
// It allows for concurrent execution of cleanup operations and handles OS signals for graceful termination.
package closer

import (
	"log"
	"os"
	"os/signal"
	"sync"
)

// globalCloser is the default instance of Closer used for package-level functions.
var globalCloser = New()

// Add registers one or more closing functions to the global closer instance.
// These functions will be executed concurrently when CloseAll is called.
func Add(f ...closeFunc) {
	globalCloser.Add(f...)
}

// Wait blocks until all registered closing functions have completed execution.
func Wait() {
	globalCloser.Wait()
}

// CloseAll triggers the execution of all registered closing functions in the global closer instance.
// All functions are executed concurrently, and any errors are logged.
func CloseAll() {
	globalCloser.CloseAll()
}

// closeFunc represents a function that performs cleanup operations and may return an error.
type closeFunc func() error

// Closer manages a collection of closing functions and provides thread-safe operations
// for adding and executing these functions.
type Closer struct {
	mu    sync.Mutex    // protects access to funcs slice
	once  sync.Once     // ensures CloseAll is executed only once
	done  chan struct{} // signals when all closing functions have completed
	funcs []closeFunc   // collection of functions to be executed on close
}

// New creates a new Closer instance. If OS signals are provided, it will automatically
// trigger CloseAll when any of these signals are received.
//
// Example:
//
//	closer := New(syscall.SIGINT, syscall.SIGTERM)
func New(sigs ...os.Signal) *Closer {
	c := &Closer{done: make(chan struct{}, 1)}
	if len(sigs) > 0 {
		go func() {
			ch := make(chan os.Signal, 1)
			signal.Notify(ch, sigs...)
			<-ch
			signal.Stop(ch)
			c.CloseAll()
		}()
	}
	return c
}

// Add registers one or more closing functions to be executed when CloseAll is called.
// This method is thread-safe and can be called concurrently.
func (c *Closer) Add(f ...closeFunc) {
	c.mu.Lock()
	c.funcs = append(c.funcs, f...)
	c.mu.Unlock()
}

// Wait blocks until all registered closing functions have completed execution.
// This method is typically called after CloseAll to ensure all cleanup operations have finished.
func (c *Closer) Wait() {
	<-c.done
}

// CloseAll executes all registered closing functions concurrently.
// It ensures that:
// - Each function is executed exactly once
// - All functions are executed concurrently
// - Any errors returned by closing functions are logged
// - The done channel is closed after all functions complete
// This method is thread-safe and idempotent.
func (c *Closer) CloseAll() {
	c.once.Do(func() {
		defer close(c.done)
		c.mu.Lock()
		funcs := c.funcs
		c.funcs = nil
		c.mu.Unlock()

		wg := sync.WaitGroup{}
		errs := make(chan error, len(funcs))
		for _, fn := range funcs {
			wg.Add(1)
			go func(fn closeFunc) {
				defer wg.Done()
				errs <- fn()
			}(fn)
		}

		go func() {
			wg.Wait()
			close(errs)
		}()

		for err := range errs {
			if err != nil {
				log.Println("error returned from closer")
			}
		}

		c.done <- struct{}{}
	})
}
