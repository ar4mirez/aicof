# Go Common Pitfalls

## Don't Do This

```go
// Ignoring errors
result, _ := doSomething()

// Not using context for cancellation
func LongRunningTask() {
    time.Sleep(10 * time.Minute)
}

// Goroutine leak (no way to stop)
go func() {
    for {
        doWork()
    }
}()

// Range loop variable capture (pre-Go 1.22)
for _, item := range items {
    go func() {
        process(item) // Wrong: captures loop variable
    }()
}

// Not closing resources
file, _ := os.Open("file.txt")
defer file.Close() // Better, but still ignoring error
```

## Do This Instead

```go
// Proper error handling
result, err := doSomething()
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}

// Use context for cancellation
func LongRunningTask(ctx context.Context) error {
    select {
    case <-time.After(10 * time.Minute):
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

// Goroutine with cancellation
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go func() {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            doWork()
        }
    }
}()

// Correct loop variable capture (pre-Go 1.22)
for _, item := range items {
    item := item // Capture loop variable
    go func() {
        process(item)
    }()
}

// Proper resource cleanup
file, err := os.Open("file.txt")
if err != nil {
    return err
}
defer func() {
    if err := file.Close(); err != nil {
        log.Printf("failed to close file: %v", err)
    }
}()
```
