# Axum Patterns Reference

> Concrete, copy-pasteable patterns that complement the core guardrails in SKILL.md.
> For routing, extractors, error handling basics, and project structure, see SKILL.md first.

---

## Handler Patterns

### Full CRUD Handler Set

```rust
// src/handlers/users.rs
use axum::{
    extract::{Path, Query, State},
    http::StatusCode,
    Json,
};
use serde::Deserialize;
use uuid::Uuid;

use crate::{
    errors::AppResult,
    extractors::auth::AuthUser,
    models::user::{AuthResponse, CreateUserDto, LoginDto, UpdateUserDto, UserResponse},
    services::user_service::UserService,
    AppState,
};

#[derive(Deserialize)]
pub struct PaginationQuery {
    #[serde(default = "default_page")]
    pub page: i64,
    #[serde(default = "default_per_page")]
    pub per_page: i64,
}

fn default_page() -> i64 { 1 }
fn default_per_page() -> i64 { 20 }

pub async fn register(
    State(state): State<AppState>,
    Json(dto): Json<CreateUserDto>,
) -> AppResult<(StatusCode, Json<AuthResponse>)> {
    let service = UserService::new(state.pool, state.config);
    let response = service.register(dto).await?;
    Ok((StatusCode::CREATED, Json(response)))
}

pub async fn login(
    State(state): State<AppState>,
    Json(dto): Json<LoginDto>,
) -> AppResult<Json<AuthResponse>> {
    let service = UserService::new(state.pool, state.config);
    let response = service.login(dto).await?;
    Ok(Json(response))
}

pub async fn get_current_user(
    State(state): State<AppState>,
    auth_user: AuthUser,
) -> AppResult<Json<UserResponse>> {
    let service = UserService::new(state.pool, state.config);
    let user = service.get_user(auth_user.user_id).await?;
    Ok(Json(user))
}

pub async fn list_users(
    State(state): State<AppState>,
    Query(pagination): Query<PaginationQuery>,
    _auth_user: AuthUser,
) -> AppResult<Json<Vec<UserResponse>>> {
    let service = UserService::new(state.pool, state.config);
    let users = service.list_users(pagination.page, pagination.per_page).await?;
    Ok(Json(users))
}

pub async fn get_user(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    _auth_user: AuthUser,
) -> AppResult<Json<UserResponse>> {
    let service = UserService::new(state.pool, state.config);
    let user = service.get_user(id).await?;
    Ok(Json(user))
}

pub async fn update_user(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    _auth_user: AuthUser,
    Json(dto): Json<UpdateUserDto>,
) -> AppResult<Json<UserResponse>> {
    let service = UserService::new(state.pool, state.config);
    let user = service.update_user(id, dto).await?;
    Ok(Json(user))
}

pub async fn delete_user(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    _auth_user: AuthUser,
) -> AppResult<StatusCode> {
    let service = UserService::new(state.pool, state.config);
    service.delete_user(id).await?;
    Ok(StatusCode::NO_CONTENT)
}
```

### Health Check Handler

```rust
pub mod health {
    use axum::Json;
    use serde_json::{json, Value};

    pub async fn health_check() -> Json<Value> {
        Json(json!({
            "status": "healthy",
            "timestamp": chrono::Utc::now().to_rfc3339()
        }))
    }
}
```

---

## Database Integration

### Repository Pattern (SQLx)

```rust
// src/repositories/user_repository.rs
use sqlx::PgPool;
use uuid::Uuid;

use crate::{errors::AppError, models::user::User};

pub struct UserRepository {
    pool: PgPool,
}

impl UserRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    pub async fn find_by_id(&self, id: Uuid) -> Result<Option<User>, AppError> {
        let user = sqlx::query_as::<_, User>(
            r#"
            SELECT id, email, password_hash, name, role, is_active, created_at, updated_at
            FROM users
            WHERE id = $1
            "#,
        )
        .bind(id)
        .fetch_optional(&self.pool)
        .await?;

        Ok(user)
    }

    pub async fn find_by_email(&self, email: &str) -> Result<Option<User>, AppError> {
        let user = sqlx::query_as::<_, User>(
            r#"
            SELECT id, email, password_hash, name, role, is_active, created_at, updated_at
            FROM users
            WHERE email = $1
            "#,
        )
        .bind(email)
        .fetch_optional(&self.pool)
        .await?;

        Ok(user)
    }

    pub async fn find_all(&self, limit: i64, offset: i64) -> Result<Vec<User>, AppError> {
        let users = sqlx::query_as::<_, User>(
            r#"
            SELECT id, email, password_hash, name, role, is_active, created_at, updated_at
            FROM users
            ORDER BY created_at DESC
            LIMIT $1 OFFSET $2
            "#,
        )
        .bind(limit)
        .bind(offset)
        .fetch_all(&self.pool)
        .await?;

        Ok(users)
    }

    pub async fn create(
        &self,
        email: &str,
        password_hash: &str,
        name: &str,
    ) -> Result<User, AppError> {
        let user = sqlx::query_as::<_, User>(
            r#"
            INSERT INTO users (email, password_hash, name)
            VALUES ($1, $2, $3)
            RETURNING id, email, password_hash, name, role, is_active, created_at, updated_at
            "#,
        )
        .bind(email)
        .bind(password_hash)
        .bind(name)
        .fetch_one(&self.pool)
        .await?;

        Ok(user)
    }

    pub async fn update(
        &self,
        id: Uuid,
        name: Option<&str>,
        is_active: Option<bool>,
    ) -> Result<User, AppError> {
        let user = sqlx::query_as::<_, User>(
            r#"
            UPDATE users
            SET
                name = COALESCE($2, name),
                is_active = COALESCE($3, is_active),
                updated_at = NOW()
            WHERE id = $1
            RETURNING id, email, password_hash, name, role, is_active, created_at, updated_at
            "#,
        )
        .bind(id)
        .bind(name)
        .bind(is_active)
        .fetch_one(&self.pool)
        .await?;

        Ok(user)
    }

    pub async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        sqlx::query("DELETE FROM users WHERE id = $1")
            .bind(id)
            .execute(&self.pool)
            .await?;

        Ok(())
    }
}
```

### Database Migration

```sql
-- migrations/001_create_users.sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(100) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
```

---

## Models and Validation

### Domain Models With DTOs

```rust
// src/models/user.rs
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;
use uuid::Uuid;
use validator::Validate;

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct User {
    pub id: Uuid,
    pub email: String,
    #[serde(skip_serializing)]
    pub password_hash: String,
    pub name: String,
    pub role: String,
    pub is_active: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Deserialize, Validate)]
pub struct CreateUserDto {
    #[validate(email(message = "Invalid email format"))]
    pub email: String,

    #[validate(length(min = 8, message = "Password must be at least 8 characters"))]
    pub password: String,

    #[validate(length(min = 1, max = 100, message = "Name must be 1-100 characters"))]
    pub name: String,
}

#[derive(Debug, Deserialize, Validate)]
pub struct UpdateUserDto {
    #[validate(length(min = 1, max = 100, message = "Name must be 1-100 characters"))]
    pub name: Option<String>,

    pub is_active: Option<bool>,
}

#[derive(Debug, Deserialize, Validate)]
pub struct LoginDto {
    #[validate(email(message = "Invalid email format"))]
    pub email: String,

    #[validate(length(min = 1, message = "Password is required"))]
    pub password: String,
}

#[derive(Debug, Serialize)]
pub struct UserResponse {
    pub id: Uuid,
    pub email: String,
    pub name: String,
    pub role: String,
    pub is_active: bool,
    pub created_at: DateTime<Utc>,
}

impl From<User> for UserResponse {
    fn from(user: User) -> Self {
        Self {
            id: user.id,
            email: user.email,
            name: user.name,
            role: user.role,
            is_active: user.is_active,
            created_at: user.created_at,
        }
    }
}

#[derive(Debug, Serialize)]
pub struct AuthResponse {
    pub user: UserResponse,
    pub token: String,
}
```

---

## Authentication

### JWT Claims and Token Management

```rust
// src/extractors/auth.rs
use jsonwebtoken::{decode, encode, DecodingKey, EncodingKey, Header, Validation};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::errors::AppError;

#[derive(Debug, Serialize, Deserialize)]
pub struct Claims {
    pub sub: Uuid,
    pub email: String,
    pub role: String,
    pub exp: i64,
    pub iat: i64,
}

impl Claims {
    pub fn new(user_id: Uuid, email: String, role: String, expiration_hours: i64) -> Self {
        let now = chrono::Utc::now();
        Self {
            sub: user_id,
            email,
            role,
            exp: (now + chrono::Duration::hours(expiration_hours)).timestamp(),
            iat: now.timestamp(),
        }
    }
}

pub fn create_token(claims: &Claims, secret: &str) -> Result<String, AppError> {
    encode(
        &Header::default(),
        claims,
        &EncodingKey::from_secret(secret.as_bytes()),
    )
    .map_err(|e| AppError::Internal(anyhow::anyhow!("Failed to create token: {}", e)))
}

pub fn verify_token(token: &str, secret: &str) -> Result<Claims, AppError> {
    decode::<Claims>(
        token,
        &DecodingKey::from_secret(secret.as_bytes()),
        &Validation::default(),
    )
    .map(|data| data.claims)
    .map_err(|_| AppError::Unauthorized("Invalid token".to_string()))
}
```

### Service Layer With Auth

```rust
// src/services/user_service.rs
use validator::Validate;

pub struct UserService {
    repository: UserRepository,
    config: Config,
}

impl UserService {
    pub fn new(pool: PgPool, config: Config) -> Self {
        Self {
            repository: UserRepository::new(pool),
            config,
        }
    }

    pub async fn register(&self, dto: CreateUserDto) -> Result<AuthResponse, AppError> {
        dto.validate()
            .map_err(|e| AppError::Validation(e.to_string()))?;

        if self.repository.find_by_email(&dto.email).await?.is_some() {
            return Err(AppError::Conflict("Email already registered".into()));
        }

        let password_hash = hash_password(&dto.password)?;
        let user = self.repository.create(&dto.email, &password_hash, &dto.name).await?;

        let claims = Claims::new(
            user.id, user.email.clone(), user.role.clone(),
            self.config.jwt_expiration_hours,
        );
        let token = create_token(&claims, &self.config.jwt_secret)?;

        Ok(AuthResponse { user: user.into(), token })
    }

    pub async fn login(&self, dto: LoginDto) -> Result<AuthResponse, AppError> {
        dto.validate()
            .map_err(|e| AppError::Validation(e.to_string()))?;

        let user = self.repository.find_by_email(&dto.email).await?
            .ok_or_else(|| AppError::Unauthorized("Invalid credentials".into()))?;

        if !user.is_active {
            return Err(AppError::Forbidden("Account is deactivated".into()));
        }

        if !verify_password(&dto.password, &user.password_hash)? {
            return Err(AppError::Unauthorized("Invalid credentials".into()));
        }

        let claims = Claims::new(
            user.id, user.email.clone(), user.role.clone(),
            self.config.jwt_expiration_hours,
        );
        let token = create_token(&claims, &self.config.jwt_secret)?;

        Ok(AuthResponse { user: user.into(), token })
    }
}

fn hash_password(password: &str) -> Result<String, AppError> {
    bcrypt::hash(password, bcrypt::DEFAULT_COST)
        .map_err(|e| AppError::Internal(anyhow::anyhow!("Failed to hash password: {}", e)))
}

fn verify_password(password: &str, hash: &str) -> Result<bool, AppError> {
    bcrypt::verify(password, hash)
        .map_err(|e| AppError::Internal(anyhow::anyhow!("Failed to verify password: {}", e)))
}
```

---

## Configuration

```rust
// src/config.rs
use serde::Deserialize;

#[derive(Clone, Deserialize)]
pub struct Config {
    pub port: u16,
    pub database_url: String,
    pub jwt_secret: String,
    pub jwt_expiration_hours: i64,
}

impl Config {
    pub fn load() -> anyhow::Result<Self> {
        let config = config::Config::builder()
            .add_source(config::Environment::default())
            .build()?;

        Ok(config.try_deserialize()?)
    }
}
```

---

## WebSocket Support

### WebSocket Handler

```rust
use axum::{
    extract::ws::{Message, WebSocket, WebSocketUpgrade},
    response::IntoResponse,
};
use futures::{SinkExt, StreamExt};

pub async fn ws_handler(ws: WebSocketUpgrade) -> impl IntoResponse {
    ws.on_upgrade(handle_socket)
}

async fn handle_socket(mut socket: WebSocket) {
    while let Some(Ok(msg)) = socket.next().await {
        match msg {
            Message::Text(text) => {
                // Echo the message back
                if socket.send(Message::Text(format!("Echo: {}", text))).await.is_err() {
                    break;
                }
            }
            Message::Close(_) => break,
            _ => {}
        }
    }
}
```

### WebSocket With Shared State

```rust
use std::sync::Arc;
use tokio::sync::broadcast;

pub struct WsState {
    pub tx: broadcast::Sender<String>,
}

pub async fn ws_handler(
    ws: WebSocketUpgrade,
    State(state): State<Arc<WsState>>,
) -> impl IntoResponse {
    ws.on_upgrade(move |socket| handle_socket(socket, state))
}

async fn handle_socket(mut socket: WebSocket, state: Arc<WsState>) {
    let mut rx = state.tx.subscribe();

    let (mut sender, mut receiver) = socket.split();

    // Spawn task to forward broadcast messages to the client
    let send_task = tokio::spawn(async move {
        while let Ok(msg) = rx.recv().await {
            if sender.send(Message::Text(msg)).await.is_err() {
                break;
            }
        }
    });

    // Read messages from client and broadcast them
    while let Some(Ok(Message::Text(text))) = receiver.next().await {
        let _ = state.tx.send(text);
    }

    send_task.abort();
}
```

---

## Testing Patterns

### Integration Tests With axum::test

```rust
// tests/integration_tests.rs
use axum::{
    body::Body,
    http::{Request, StatusCode},
};
use serde_json::json;
use tower::ServiceExt;

use myproject::{routes::create_router, AppState};

async fn setup_test_app() -> axum::Router {
    let pool = sqlx::PgPool::connect("postgres://test:test@localhost/test_db")
        .await
        .unwrap();

    let config = myproject::config::Config {
        port: 3000,
        database_url: String::new(),
        jwt_secret: "test_secret".to_string(),
        jwt_expiration_hours: 24,
    };

    let state = AppState::new(pool, config);
    create_router(state)
}

#[tokio::test]
async fn test_health_check() {
    let app = setup_test_app().await;

    let response = app
        .oneshot(Request::builder().uri("/health").body(Body::empty()).unwrap())
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::OK);
}

#[tokio::test]
async fn test_register_user() {
    let app = setup_test_app().await;

    let body = json!({
        "email": "test@example.com",
        "password": "password123",
        "name": "Test User"
    });

    let response = app
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/register")
                .header("Content-Type", "application/json")
                .body(Body::from(serde_json::to_string(&body).unwrap()))
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::CREATED);
}

#[tokio::test]
async fn test_login_invalid_credentials() {
    let app = setup_test_app().await;

    let body = json!({
        "email": "nonexistent@example.com",
        "password": "wrongpassword"
    });

    let response = app
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/login")
                .header("Content-Type", "application/json")
                .body(Body::from(serde_json::to_string(&body).unwrap()))
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::UNAUTHORIZED);
}
```

### Unit Testing Services With Mocks

```rust
#[cfg(test)]
mod tests {
    use mockall::predicate::*;

    // Define a trait for the repository to enable mocking
    #[automock]
    #[async_trait]
    trait UserRepo {
        async fn find_by_id(&self, id: Uuid) -> Result<Option<User>, AppError>;
        async fn find_by_email(&self, email: &str) -> Result<Option<User>, AppError>;
        async fn create(&self, email: &str, hash: &str, name: &str) -> Result<User, AppError>;
    }

    #[tokio::test]
    async fn register_rejects_duplicate_email() {
        let mut mock_repo = MockUserRepo::new();
        mock_repo
            .expect_find_by_email()
            .with(eq("taken@example.com"))
            .returning(|_| Ok(Some(make_test_user())));

        let service = UserService::with_repo(mock_repo, test_config());
        let dto = CreateUserDto {
            email: "taken@example.com".into(),
            password: "validpass123".into(),
            name: "Test".into(),
        };

        let result = service.register(dto).await;
        assert!(matches!(result, Err(AppError::Conflict(_))));
    }

    #[tokio::test]
    async fn login_fails_for_inactive_user() {
        let mut mock_repo = MockUserRepo::new();
        let mut user = make_test_user();
        user.is_active = false;

        mock_repo
            .expect_find_by_email()
            .returning(move |_| Ok(Some(user.clone())));

        let service = UserService::with_repo(mock_repo, test_config());
        let dto = LoginDto {
            email: "test@example.com".into(),
            password: "password123".into(),
        };

        let result = service.login(dto).await;
        assert!(matches!(result, Err(AppError::Forbidden(_))));
    }
}
```

### Testing Rules

- Use `oneshot` for stateless request tests (does not start a server)
- Create a shared `setup_test_app()` function to avoid duplication
- Use a dedicated test database; run migrations before tests
- Mock repositories (not services) for unit tests
- Test both success and error paths for every endpoint
- Assert on status codes, response bodies, and headers
- Use `#[tokio::test]` for all async tests

---

## Comparison: Axum vs Actix-web vs Rocket

| Feature | Axum | Actix-web | Rocket |
|---------|------|-----------|--------|
| Async Runtime | Tokio | Actix (Tokio) | Tokio |
| Type Safety | Compile-time extractors | Runtime | Compile-time |
| Middleware | Tower | Custom | Fairings |
| Performance | Very Fast | Fastest | Fast |
| Ecosystem | Tower/Hyper | Actix | Rocket |
| Learning Curve | Moderate | Moderate | Gentle |
| Maturity | Newer | Mature | Mature |
