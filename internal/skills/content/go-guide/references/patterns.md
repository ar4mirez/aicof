# Go Patterns Reference

## Contents

- [HTTP Server](#http-server)
- [Middleware](#middleware)
- [Database Repository](#database-repository)
- [Connection Pool](#connection-pool)
- [Worker Pool](#worker-pool)
- [Rate Limiting](#rate-limiting)
- [Performance](#performance)

## HTTP Server

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "time"
)

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/health", healthHandler)
    mux.HandleFunc("/users", usersHandler)

    srv := &http.Server{
        Addr:         ":8080",
        Handler:      mux,
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 10 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // Graceful shutdown
    go func() {
        sigint := make(chan os.Signal, 1)
        signal.Notify(sigint, os.Interrupt)
        <-sigint

        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()

        if err := srv.Shutdown(ctx); err != nil {
            log.Printf("HTTP server shutdown error: %v", err)
        }
    }()

    log.Println("Starting server on :8080")
    if err := srv.ListenAndServe(); err != http.ErrServerClosed {
        log.Fatalf("HTTP server error: %v", err)
    }
}
```

## Middleware

```go
type Middleware func(http.Handler) http.Handler

func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
    })
}

func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if token == "" {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        // Validate token
        next.ServeHTTP(w, r)
    })
}

// Chain: LoggingMiddleware(AuthMiddleware(handler))
```

## Database Repository

```go
type UserRepository struct {
    db *sql.DB
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*User, error) {
    query := `SELECT id, email, created_at FROM users WHERE id = $1`

    var user User
    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &user.ID, &user.Email, &user.CreatedAt,
    )

    if err == sql.ErrNoRows {
        return nil, &NotFoundError{Resource: "user", ID: id}
    }
    if err != nil {
        return nil, fmt.Errorf("query user: %w", err)
    }

    return &user, nil
}

func (r *UserRepository) Create(ctx context.Context, user *User) error {
    query := `INSERT INTO users (email, created_at) VALUES ($1, $2) RETURNING id`

    err := r.db.QueryRowContext(ctx, query, user.Email, time.Now()).Scan(&user.ID)
    if err != nil {
        return fmt.Errorf("create user: %w", err)
    }

    return nil
}
```

## Connection Pool

```go
func NewDB(connString string) (*sql.DB, error) {
    db, err := sql.Open("postgres", connString)
    if err != nil {
        return nil, err
    }

    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(25)
    db.SetConnMaxLifetime(5 * time.Minute)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := db.PingContext(ctx); err != nil {
        return nil, fmt.Errorf("ping database: %w", err)
    }

    return db, nil
}
```

## Worker Pool

```go
func workerPool(ctx context.Context, jobs <-chan Job, results chan<- Result, workers int) {
    var wg sync.WaitGroup

    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for job := range jobs {
                select {
                case <-ctx.Done():
                    return
                case results <- processJob(job):
                }
            }
        }()
    }

    wg.Wait()
    close(results)
}
```

## Rate Limiting

```go
import "golang.org/x/time/rate"

limiter := rate.NewLimiter(rate.Every(time.Second), 10) // 10 req/sec

func handler(w http.ResponseWriter, r *http.Request) {
    if !limiter.Allow() {
        http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
        return
    }
    // Handle request
}
```

## Performance

### String Building

```go
var sb strings.Builder
for _, s := range items {
    sb.WriteString(s)
}
result := sb.String()
```

### Object Pooling

```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

buf := bufferPool.Get().(*bytes.Buffer)
defer bufferPool.Put(buf)
buf.Reset()
```

### Performance Guardrails

- Use `sync.Pool` for frequently allocated objects
- Avoid string concatenation in loops (use `strings.Builder`)
- Use buffered channels when appropriate
- Profile before optimizing: `go test -bench`, `pprof`
