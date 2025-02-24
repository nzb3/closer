# Closer

---

A lightweight Go package for managing graceful shutdown of applications. It provides concurrent execution of cleanup operations and handles OS signals for graceful termination.

## Features

- Thread-safe operation
- Concurrent execution of cleanup functions
- OS signal handling
- Global and instance-based usage
- Simple API


## Installation

```bash
go get github.com/nzb3/closer
```


## Usage

### Basic Usage with Global Closer

```go
package main

import (
    "context"
    "github.com/nzb3/closer"
    "syscall"
)

func main() {
    // Add cleanup functions
    closer.Add(
        func() error {
            // Cleanup database connection
            return nil
        },
        func() error {
            // Cleanup HTTP server
            return nil
        },
    )

    // Wait for all cleanup operations to complete
    closer.Wait()
}
```


### Using with OS Signals

```go
package main

import (
    "github.com/nzb3/closer"
    "syscall"
)

func main() {
    // Create new closer instance with signal handling
    c := closer.New(syscall.SIGINT, syscall.SIGTERM)

    // Add cleanup functions
    c.Add(func() error {
        // Your cleanup code here
        return nil
    })

    // Wait for signal and cleanup completion
    c.Wait()
}
```


### HTTP Server Example

```go
package main

import (
    "context"
    "github.com/nzb3/closer"
    "net/http"
    "syscall"
)

func main() {
    srv := &http.Server{Addr: ":8080"}
    
    // Create closer with signal handling
    c := closer.New(syscall.SIGINT, syscall.SIGTERM)
    
    // Add server shutdown to closer
    c.Add(func() error {
        return srv.Shutdown(context.Background())
    })

    // Start server
    go srv.ListenAndServe()

    // Wait for shutdown
    c.Wait()
}
```


## License

[MIT license](LICENSE)

