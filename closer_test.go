// closer_test.go
package closer

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestNewWithoutSignals creates a new Closer without any OS signals
// and verifies that it is non-nil.
func TestNewWithoutSignals(t *testing.T) {
	c := New()
	if c == nil {
		t.Fatal("expected non-nil Closer")
	}
}

// TestCloseAll verifies that added cleanup functions are executed
// exactly once when CloseAll is called.
func TestCloseAll(t *testing.T) {
	c := New()
	var mu sync.Mutex
	counter := 0

	// cleanup function increments counter
	cleanup := func() error {
		mu.Lock()
		counter++
		mu.Unlock()
		return nil
	}

	// Add three cleanup functions
	c.Add(cleanup, cleanup, cleanup)
	// Run CloseAll in a separate goroutine
	go c.CloseAll()
	c.Wait()

	mu.Lock()
	if counter != 3 {
		t.Errorf("expected counter to be 3, got %d", counter)
	}
	mu.Unlock()
}

// TestMultipleCloseAll ensures that CloseAll is idempotent.
// The cleanup function should only be executed once even if CloseAll is called concurrently.
func TestMultipleCloseAll(t *testing.T) {
	c := New()
	var executed int32

	cleanup := func() error {
		atomic.AddInt32(&executed, 1)
		return nil
	}
	c.Add(cleanup)

	var wg sync.WaitGroup
	wg.Add(2)
	// Call CloseAll concurrently from two goroutines.
	go func() {
		defer wg.Done()
		c.CloseAll()
	}()
	go func() {
		defer wg.Done()
		c.CloseAll()
	}()
	wg.Wait()
	c.Wait()

	if atomic.LoadInt32(&executed) != 1 {
		t.Errorf("expected cleanup function executed once, got %d", executed)
	}
}

// TestCleanupFunctionError verifies that cleanup functions returning errors are nonetheless executed.
func TestCleanupFunctionError(t *testing.T) {
	c := New()
	var executed int32
	errCleanup := func() error {
		atomic.AddInt32(&executed, 1)
		return errors.New("dummy error")
	}

	c.Add(errCleanup)
	c.CloseAll()
	c.Wait()

	if atomic.LoadInt32(&executed) != 1 {
		t.Errorf("expected cleanup function executed once, got %d", executed)
	}
}

// TestGlobalFunctions tests the package-level global functions.
// It resets the globalCloser, adds a cleanup function, calls CloseAll, and waits for completion.
func TestGlobalFunctions(t *testing.T) {
	// Replace the globalCloser with a new instance for testing.
	globalCloser = New()
	var varSet int32

	simpleCleanup := func() error {
		atomic.AddInt32(&varSet, 1)
		return nil
	}
	Add(simpleCleanup)
	CloseAll()
	Wait()

	if atomic.LoadInt32(&varSet) != 1 {
		t.Errorf("expected global cleanup function to execute once, got %d", varSet)
	}
}

// TestCloserWithSignal simulates a signal-triggered close.
//
// NOTE: Actually sending a signal to the process might interfere with tests,
// so we simulate the behavior by sending a signal on a separate goroutine.
func TestCloserWithSignal(t *testing.T) {
	// Create a new Closer that is watching for os.Interrupt.
	c := New(os.Interrupt)
	var flag int32
	cleanup := func() error {
		atomic.AddInt32(&flag, 1)
		return nil
	}
	c.Add(cleanup)

	// Simulate sending an os.Interrupt to trigger the closer.
	// Note: Using time.AfterFunc to mimic delayed OS signal delivery.
	time.AfterFunc(10*time.Millisecond, func() {
		// Send os.Interrupt signal to the current process.
		// In tests, this call should not terminate the process due to signal.Notify in New().
		p, err := os.FindProcess(os.Getpid())
		if err != nil {
			t.Error(fmt.Errorf("failed to find process: %w", err))
		}
		p.Signal(os.Interrupt)
	})

	// Wait for the cleanup to be called.
	c.Wait()

	if atomic.LoadInt32(&flag) != 1 {
		t.Errorf("expected cleanup function to execute once due to signal trigger, got %d", flag)
	}
}
