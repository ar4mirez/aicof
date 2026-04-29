# Rust Patterns Reference

> Concrete, copy-pasteable patterns that complement the core guardrails in SKILL.md.
> For error handling basics (thiserror, anyhow, `?`), see SKILL.md first.

---

## Error Handling Patterns

### Layered Error Enums (Domain to API)

```rust
// domain/error.rs -- inner layer, no HTTP awareness
#[derive(Debug, thiserror::Error)]
pub enum DomainError {
    #[error("entity {kind} with id {id} not found")]
    NotFound { kind: &'static str, id: String },

    #[error("validation failed: {0}")]
    Validation(String),

    #[error("operation not permitted")]
    Forbidden,
}

// api/error.rs -- outer layer, maps domain errors to HTTP
#[derive(Debug, thiserror::Error)]
pub enum ApiError {
    #[error(transparent)]
    Domain(#[from] DomainError),

    #[error(transparent)]
    Internal(#[from] anyhow::Error),
}

impl axum::response::IntoResponse for ApiError {
    fn into_response(self) -> axum::response::Response {
        let (status, msg) = match &self {
            Self::Domain(DomainError::NotFound { .. }) => (StatusCode::NOT_FOUND, self.to_string()),
            Self::Domain(DomainError::Validation(m)) => (StatusCode::BAD_REQUEST, m.clone()),
            Self::Domain(DomainError::Forbidden) => (StatusCode::FORBIDDEN, self.to_string()),
            Self::Internal(_) => (StatusCode::INTERNAL_SERVER_ERROR, "internal error".into()),
        };
        (status, axum::Json(serde_json::json!({ "error": msg }))).into_response()
    }
}
```

### Result Type Alias Per Module

```rust
// Declare once at crate or module root to avoid repeating the error type.
pub type Result<T> = std::result::Result<T, DomainError>;

// Usage -- callers write `Result<User>` instead of `Result<User, DomainError>`
pub fn find_user(id: &str) -> Result<User> { /* ... */ }
```

---

## Builder Pattern

### Derive-Free Builder With Compile-Time Safety

```rust
/// Config is the final, immutable product.
pub struct Config {
    host: String,
    port: u16,
    max_retries: u32,
}

/// Builder accumulates optional fields; `build()` validates.
pub struct ConfigBuilder {
    host: Option<String>,
    port: Option<u16>,
    max_retries: u32, // has a sensible default
}

impl ConfigBuilder {
    pub fn new() -> Self {
        Self { host: None, port: None, max_retries: 3 }
    }

    pub fn host(mut self, host: impl Into<String>) -> Self {
        self.host = Some(host.into());
        self
    }

    pub fn port(mut self, port: u16) -> Self {
        self.port = Some(port);
        self
    }

    pub fn max_retries(mut self, n: u32) -> Self {
        self.max_retries = n;
        self
    }

    /// Consumes the builder; returns `Err` if required fields are missing.
    pub fn build(self) -> Result<Config, &'static str> {
        Ok(Config {
            host: self.host.ok_or("host is required")?,
            port: self.port.ok_or("port is required")?,
            max_retries: self.max_retries,
        })
    }
}
```

---

## Iterator Adapter Chains

### Idiomatic Collection Transforms

```rust
// Filter-map-collect: transform a vec of raw strings into validated domain objects.
let users: Vec<User> = raw_rows
    .into_iter()
    .filter(|r| !r.deleted)
    .map(|r| User::try_from(r))
    .collect::<Result<Vec<_>, _>>()?; // short-circuits on first error

// Chunk processing with itertools (avoids loading everything into memory).
use itertools::Itertools;
for batch in ids.iter().chunks(100).into_iter() {
    let chunk: Vec<_> = batch.collect();
    db.bulk_insert(&chunk).await?;
}

// Fold to accumulate a summary value.
let total_cents: u64 = line_items
    .iter()
    .map(|item| u64::from(item.quantity) * item.unit_price_cents)
    .sum();
```

### Custom Iterator

```rust
/// Yields pages of results from an API until exhausted.
pub struct Paginator<F> {
    fetch: F,
    cursor: Option<String>,
    done: bool,
}

impl<F, Fut> Paginator<F>
where
    F: FnMut(Option<&str>) -> Fut,
    Fut: std::future::Future<Output = Result<Page, anyhow::Error>>,
{
    pub fn new(fetch: F) -> Self {
        Self { fetch, cursor: None, done: false }
    }

    pub async fn next_page(&mut self) -> Result<Option<Page>, anyhow::Error> {
        if self.done { return Ok(None); }
        let page = (self.fetch)(self.cursor.as_deref()).await?;
        self.cursor = page.next_cursor.clone();
        self.done = page.next_cursor.is_none();
        Ok(Some(page))
    }
}
```

---

## Async Patterns

### Graceful Shutdown With Tokio

```rust
use tokio::signal;
use tokio::sync::watch;

pub async fn run(listener: TcpListener) -> anyhow::Result<()> {
    let (shutdown_tx, shutdown_rx) = watch::channel(false);

    let server_handle = tokio::spawn(serve(listener, shutdown_rx));

    // Wait for ctrl-c or SIGTERM.
    signal::ctrl_c().await?;
    tracing::info!("shutdown signal received");
    let _ = shutdown_tx.send(true);

    server_handle.await??;
    Ok(())
}
```

### Select With Cancellation

```rust
use tokio::time::{self, Duration};

/// Races a database query against a timeout and a shutdown signal.
async fn fetch_with_deadline<T>(
    query: impl std::future::Future<Output = Result<T, sqlx::Error>>,
    mut shutdown: watch::Receiver<bool>,
) -> anyhow::Result<T> {
    tokio::select! {
        result = query => Ok(result?),
        _ = time::sleep(Duration::from_secs(5)) => {
            anyhow::bail!("query timed out after 5s");
        }
        _ = shutdown.changed() => {
            anyhow::bail!("operation cancelled by shutdown");
        }
    }
}
```

### Spawning Background Work

```rust
/// Spawn a background task and log errors instead of panicking.
fn spawn_logged<F>(name: &'static str, fut: F) -> tokio::task::JoinHandle<()>
where
    F: std::future::Future<Output = anyhow::Result<()>> + Send + 'static,
{
    tokio::spawn(async move {
        if let Err(e) = fut.await {
            tracing::error!(task = name, error = %e, "background task failed");
        }
    })
}
```

---

## Smart Pointer Usage

### When to Reach for Each Pointer

| Pointer | Use when | Not for |
|---------|----------|---------|
| `Box<T>` | Heap allocation of single value, trait objects (`Box<dyn Error>`), recursive types | Shared ownership |
| `Rc<T>` | Single-threaded shared ownership (e.g., tree nodes with multiple parents) | Anything `Send`/async |
| `Arc<T>` | Multi-threaded shared ownership, shared state across tasks | Single-threaded hot paths (use `Rc`) |
| `RefCell<T>` | Interior mutability with runtime borrow checks (single-threaded) | Async code, multi-threaded code |
| `Mutex<T>` | Interior mutability across threads; prefer `tokio::sync::Mutex` in async | Uncontended single-threaded access |

### Arc + Mutex for Shared Async State

```rust
use std::sync::Arc;
use tokio::sync::Mutex;

#[derive(Clone)]
pub struct AppState {
    inner: Arc<Mutex<AppStateInner>>,
}

struct AppStateInner {
    request_count: u64,
}

impl AppState {
    pub fn new() -> Self {
        Self { inner: Arc::new(Mutex::new(AppStateInner { request_count: 0 })) }
    }

    pub async fn increment(&self) -> u64 {
        let mut state = self.inner.lock().await;
        state.request_count += 1;
        state.request_count
    }
}
```

### Box for Trait Objects and Recursive Types

```rust
// Trait object -- erase concrete type behind a pointer.
type Handler = Box<dyn Fn(&Request) -> Response + Send + Sync>;

// Recursive type -- the compiler needs a known size.
pub enum Expr {
    Literal(f64),
    Add(Box<Expr>, Box<Expr>),
    Mul(Box<Expr>, Box<Expr>),
}
```
