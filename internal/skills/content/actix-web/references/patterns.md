# Actix-web Patterns Reference

> Concrete, copy-pasteable patterns that complement the core guardrails in SKILL.md.
> For project structure, routing, and error handling basics, see SKILL.md first.

---

## Handler Patterns

### CRUD Handler Set

```rust
use actix_web::{web, HttpMessage, HttpRequest, HttpResponse};
use serde::Deserialize;
use uuid::Uuid;

use crate::{
    errors::AppResult,
    middleware::auth::AuthenticatedUser,
    models::user::{CreateUserDto, LoginDto, UpdateUserDto},
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
    state: web::Data<AppState>,
    body: web::Json<CreateUserDto>,
) -> AppResult<HttpResponse> {
    let service = UserService::new(state.pool.clone(), state.config.clone());
    let response = service.register(body.into_inner()).await?;
    Ok(HttpResponse::Created().json(response))
}

pub async fn login(
    state: web::Data<AppState>,
    body: web::Json<LoginDto>,
) -> AppResult<HttpResponse> {
    let service = UserService::new(state.pool.clone(), state.config.clone());
    let response = service.login(body.into_inner()).await?;
    Ok(HttpResponse::Ok().json(response))
}

pub async fn get_current_user(
    state: web::Data<AppState>,
    req: HttpRequest,
) -> AppResult<HttpResponse> {
    let auth_user = req
        .extensions()
        .get::<AuthenticatedUser>()
        .cloned()
        .ok_or_else(|| AppError::Unauthorized("Not authenticated".to_string()))?;

    let service = UserService::new(state.pool.clone(), state.config.clone());
    let user = service.get_user(auth_user.user_id).await?;
    Ok(HttpResponse::Ok().json(user))
}

pub async fn list_users(
    state: web::Data<AppState>,
    query: web::Query<PaginationQuery>,
) -> AppResult<HttpResponse> {
    let service = UserService::new(state.pool.clone(), state.config.clone());
    let users = service.list_users(query.page, query.per_page).await?;
    Ok(HttpResponse::Ok().json(users))
}

pub async fn get_user(
    state: web::Data<AppState>,
    path: web::Path<Uuid>,
) -> AppResult<HttpResponse> {
    let service = UserService::new(state.pool.clone(), state.config.clone());
    let user = service.get_user(path.into_inner()).await?;
    Ok(HttpResponse::Ok().json(user))
}

pub async fn update_user(
    state: web::Data<AppState>,
    path: web::Path<Uuid>,
    body: web::Json<UpdateUserDto>,
) -> AppResult<HttpResponse> {
    let service = UserService::new(state.pool.clone(), state.config.clone());
    let user = service.update_user(path.into_inner(), body.into_inner()).await?;
    Ok(HttpResponse::Ok().json(user))
}

pub async fn delete_user(
    state: web::Data<AppState>,
    path: web::Path<Uuid>,
) -> AppResult<HttpResponse> {
    let service = UserService::new(state.pool.clone(), state.config.clone());
    service.delete_user(path.into_inner()).await?;
    Ok(HttpResponse::NoContent().finish())
}
```

### Health Check Handler

```rust
use actix_web::HttpResponse;
use serde_json::json;

pub async fn health_check() -> HttpResponse {
    HttpResponse::Ok().json(json!({
        "status": "healthy",
        "timestamp": chrono::Utc::now().to_rfc3339()
    }))
}
```

---

## Models and DTOs

### User Model With Validation

```rust
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

## Database Integration

### Repository Pattern

```rust
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
        let user = sqlx::query_as::<_, User>("SELECT * FROM users WHERE id = $1")
            .bind(id)
            .fetch_optional(&self.pool)
            .await?;
        Ok(user)
    }

    pub async fn find_by_email(&self, email: &str) -> Result<Option<User>, AppError> {
        let user = sqlx::query_as::<_, User>("SELECT * FROM users WHERE email = $1")
            .bind(email)
            .fetch_optional(&self.pool)
            .await?;
        Ok(user)
    }

    pub async fn find_all(&self, limit: i64, offset: i64) -> Result<Vec<User>, AppError> {
        let users = sqlx::query_as::<_, User>(
            "SELECT * FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2",
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
            RETURNING *
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
            SET name = COALESCE($2, name),
                is_active = COALESCE($3, is_active),
                updated_at = NOW()
            WHERE id = $1
            RETURNING *
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
```

---

## Authentication

### JWT Middleware (Full Implementation)

```rust
use actix_web::{
    dev::{forward_ready, Service, ServiceRequest, ServiceResponse, Transform},
    http::header::AUTHORIZATION,
    web, Error, HttpMessage,
};
use futures::future::{ok, LocalBoxFuture, Ready};
use jsonwebtoken::{decode, encode, DecodingKey, EncodingKey, Header, Validation};
use serde::{Deserialize, Serialize};
use std::rc::Rc;

use crate::{errors::AppError, AppState};

#[derive(Debug, Clone)]
pub struct AuthenticatedUser {
    pub user_id: uuid::Uuid,
    pub email: String,
    pub role: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct Claims {
    pub sub: uuid::Uuid,
    pub email: String,
    pub role: String,
    pub exp: i64,
    pub iat: i64,
}

impl Claims {
    pub fn new(user_id: uuid::Uuid, email: String, role: String, expiration_hours: i64) -> Self {
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
    .map_err(|e| AppError::Internal(format!("Failed to create token: {}", e)))
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

// Middleware factory
pub struct AuthMiddleware;

impl<S, B> Transform<S, ServiceRequest> for AuthMiddleware
where
    S: Service<ServiceRequest, Response = ServiceResponse<B>, Error = Error> + 'static,
    S::Future: 'static,
    B: 'static,
{
    type Response = ServiceResponse<B>;
    type Error = Error;
    type Transform = AuthMiddlewareService<S>;
    type InitError = ();
    type Future = Ready<Result<Self::Transform, Self::InitError>>;

    fn new_transform(&self, service: S) -> Self::Future {
        ok(AuthMiddlewareService {
            service: Rc::new(service),
        })
    }
}

// Middleware service
pub struct AuthMiddlewareService<S> {
    service: Rc<S>,
}

impl<S, B> Service<ServiceRequest> for AuthMiddlewareService<S>
where
    S: Service<ServiceRequest, Response = ServiceResponse<B>, Error = Error> + 'static,
    S::Future: 'static,
    B: 'static,
{
    type Response = ServiceResponse<B>;
    type Error = Error;
    type Future = LocalBoxFuture<'static, Result<Self::Response, Self::Error>>;

    forward_ready!(service);

    fn call(&self, req: ServiceRequest) -> Self::Future {
        let service = Rc::clone(&self.service);

        Box::pin(async move {
            let auth_header = req
                .headers()
                .get(AUTHORIZATION)
                .and_then(|h| h.to_str().ok())
                .ok_or_else(|| {
                    AppError::Unauthorized("Missing authorization header".to_string())
                })?;

            let token = auth_header
                .strip_prefix("Bearer ")
                .ok_or_else(|| {
                    AppError::Unauthorized("Invalid authorization format".to_string())
                })?;

            let state = req
                .app_data::<web::Data<AppState>>()
                .ok_or_else(|| AppError::Internal("App state not found".to_string()))?;

            let claims = verify_token(token, &state.config.jwt_secret)?;

            req.extensions_mut().insert(AuthenticatedUser {
                user_id: claims.sub,
                email: claims.email,
                role: claims.role,
            });

            service.call(req).await
        })
    }
}
```

---

## Service Layer

### Business Logic Pattern

```rust
use sqlx::PgPool;
use uuid::Uuid;
use validator::Validate;

use crate::{
    config::Config,
    errors::AppError,
    middleware::auth::{create_token, Claims},
    models::user::{AuthResponse, CreateUserDto, LoginDto, UpdateUserDto, UserResponse},
    repositories::user_repository::UserRepository,
};

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
        dto.validate()?;

        if self.repository.find_by_email(&dto.email).await?.is_some() {
            return Err(AppError::Conflict("Email already registered".to_string()));
        }

        let password_hash = bcrypt::hash(&dto.password, bcrypt::DEFAULT_COST)
            .map_err(|e| AppError::Internal(format!("Failed to hash password: {}", e)))?;

        let user = self
            .repository
            .create(&dto.email, &password_hash, &dto.name)
            .await?;

        let claims = Claims::new(
            user.id,
            user.email.clone(),
            user.role.clone(),
            self.config.jwt_expiration_hours,
        );
        let token = create_token(&claims, &self.config.jwt_secret)?;

        Ok(AuthResponse {
            user: user.into(),
            token,
        })
    }

    pub async fn login(&self, dto: LoginDto) -> Result<AuthResponse, AppError> {
        dto.validate()?;

        let user = self
            .repository
            .find_by_email(&dto.email)
            .await?
            .ok_or_else(|| AppError::Unauthorized("Invalid credentials".to_string()))?;

        if !user.is_active {
            return Err(AppError::Forbidden("Account is deactivated".to_string()));
        }

        let password_valid = bcrypt::verify(&dto.password, &user.password_hash)
            .map_err(|e| AppError::Internal(format!("Failed to verify password: {}", e)))?;

        if !password_valid {
            return Err(AppError::Unauthorized("Invalid credentials".to_string()));
        }

        let claims = Claims::new(
            user.id,
            user.email.clone(),
            user.role.clone(),
            self.config.jwt_expiration_hours,
        );
        let token = create_token(&claims, &self.config.jwt_secret)?;

        Ok(AuthResponse {
            user: user.into(),
            token,
        })
    }

    pub async fn get_user(&self, id: Uuid) -> Result<UserResponse, AppError> {
        self.repository
            .find_by_id(id)
            .await?
            .map(UserResponse::from)
            .ok_or_else(|| AppError::NotFound(format!("User {} not found", id)))
    }

    pub async fn list_users(
        &self,
        page: i64,
        per_page: i64,
    ) -> Result<Vec<UserResponse>, AppError> {
        let offset = (page - 1) * per_page;
        let users = self.repository.find_all(per_page, offset).await?;
        Ok(users.into_iter().map(UserResponse::from).collect())
    }

    pub async fn update_user(
        &self,
        id: Uuid,
        dto: UpdateUserDto,
    ) -> Result<UserResponse, AppError> {
        dto.validate()?;

        self.repository
            .find_by_id(id)
            .await?
            .ok_or_else(|| AppError::NotFound(format!("User {} not found", id)))?;

        let user = self
            .repository
            .update(id, dto.name.as_deref(), dto.is_active)
            .await?;

        Ok(user.into())
    }

    pub async fn delete_user(&self, id: Uuid) -> Result<(), AppError> {
        self.repository
            .find_by_id(id)
            .await?
            .ok_or_else(|| AppError::NotFound(format!("User {} not found", id)))?;

        self.repository.delete(id).await
    }
}
```

---

## Error Handling (Full Implementation)

### Complete Error Type With Conversions

```rust
use actix_web::{http::StatusCode, HttpResponse, ResponseError};
use serde::Serialize;
use std::fmt;

#[derive(Debug)]
pub enum AppError {
    NotFound(String),
    BadRequest(String),
    Unauthorized(String),
    Forbidden(String),
    Conflict(String),
    Validation(String),
    Internal(String),
    Database(sqlx::Error),
}

#[derive(Serialize)]
struct ErrorResponse {
    success: bool,
    error: ErrorDetail,
}

#[derive(Serialize)]
struct ErrorDetail {
    code: u16,
    message: String,
}

impl fmt::Display for AppError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            AppError::NotFound(msg) => write!(f, "Not found: {}", msg),
            AppError::BadRequest(msg) => write!(f, "Bad request: {}", msg),
            AppError::Unauthorized(msg) => write!(f, "Unauthorized: {}", msg),
            AppError::Forbidden(msg) => write!(f, "Forbidden: {}", msg),
            AppError::Conflict(msg) => write!(f, "Conflict: {}", msg),
            AppError::Validation(msg) => write!(f, "Validation error: {}", msg),
            AppError::Internal(msg) => write!(f, "Internal error: {}", msg),
            AppError::Database(err) => write!(f, "Database error: {}", err),
        }
    }
}

impl ResponseError for AppError {
    fn error_response(&self) -> HttpResponse {
        let (status, message) = match self {
            AppError::NotFound(msg) => (StatusCode::NOT_FOUND, msg.clone()),
            AppError::BadRequest(msg) => (StatusCode::BAD_REQUEST, msg.clone()),
            AppError::Unauthorized(msg) => (StatusCode::UNAUTHORIZED, msg.clone()),
            AppError::Forbidden(msg) => (StatusCode::FORBIDDEN, msg.clone()),
            AppError::Conflict(msg) => (StatusCode::CONFLICT, msg.clone()),
            AppError::Validation(msg) => (StatusCode::UNPROCESSABLE_ENTITY, msg.clone()),
            AppError::Internal(msg) => {
                log::error!("Internal error: {}", msg);
                (StatusCode::INTERNAL_SERVER_ERROR, "Internal server error".into())
            }
            AppError::Database(err) => {
                log::error!("Database error: {:?}", err);
                (StatusCode::INTERNAL_SERVER_ERROR, "Database error".into())
            }
        };

        HttpResponse::build(status).json(ErrorResponse {
            success: false,
            error: ErrorDetail {
                code: status.as_u16(),
                message,
            },
        })
    }
}

// Automatic conversions
impl From<sqlx::Error> for AppError {
    fn from(err: sqlx::Error) -> Self {
        AppError::Database(err)
    }
}

impl From<anyhow::Error> for AppError {
    fn from(err: anyhow::Error) -> Self {
        AppError::Internal(err.to_string())
    }
}

impl From<validator::ValidationErrors> for AppError {
    fn from(err: validator::ValidationErrors) -> Self {
        AppError::Validation(err.to_string())
    }
}

pub type AppResult<T> = Result<T, AppError>;
```

---

## Testing Patterns

### Integration Test Setup

```rust
use actix_web::{test, web, App};
use serde_json::json;

async fn setup_test_app() -> impl actix_web::dev::Service<
    actix_http::Request,
    Response = actix_web::dev::ServiceResponse,
    Error = actix_web::Error,
> {
    let pool = sqlx::PgPool::connect("postgres://test:test@localhost/test_db")
        .await
        .unwrap();

    let config = Config {
        host: "127.0.0.1".to_string(),
        port: 8080,
        database_url: String::new(),
        jwt_secret: "test_secret".to_string(),
        jwt_expiration_hours: 24,
    };

    let state = web::Data::new(AppState { pool, config });

    test::init_service(
        App::new()
            .app_data(state)
            .configure(routes::configure),
    )
    .await
}
```

### Health Check Test

```rust
#[actix_rt::test]
async fn test_health_check() {
    let app = setup_test_app().await;

    let req = test::TestRequest::get().uri("/api/v1/health").to_request();
    let resp = test::call_service(&app, req).await;

    assert!(resp.status().is_success());
}
```

### Registration Test

```rust
#[actix_rt::test]
async fn test_register_user() {
    let app = setup_test_app().await;

    let body = json!({
        "email": "test@example.com",
        "password": "password123",
        "name": "Test User"
    });

    let req = test::TestRequest::post()
        .uri("/api/v1/auth/register")
        .set_json(&body)
        .to_request();

    let resp = test::call_service(&app, req).await;
    assert_eq!(resp.status(), actix_web::http::StatusCode::CREATED);
}
```

### Authentication Failure Tests

```rust
#[actix_rt::test]
async fn test_login_invalid_credentials() {
    let app = setup_test_app().await;

    let body = json!({
        "email": "nonexistent@example.com",
        "password": "wrongpassword"
    });

    let req = test::TestRequest::post()
        .uri("/api/v1/auth/login")
        .set_json(&body)
        .to_request();

    let resp = test::call_service(&app, req).await;
    assert_eq!(resp.status(), actix_web::http::StatusCode::UNAUTHORIZED);
}

#[actix_rt::test]
async fn test_protected_route_without_token() {
    let app = setup_test_app().await;

    let req = test::TestRequest::get()
        .uri("/api/v1/users")
        .to_request();

    let resp = test::call_service(&app, req).await;
    assert_eq!(resp.status(), actix_web::http::StatusCode::UNAUTHORIZED);
}
```

### Testing With Authenticated Requests

```rust
#[actix_rt::test]
async fn test_get_users_with_valid_token() {
    let app = setup_test_app().await;

    // First register a user to get a token
    let register_body = json!({
        "email": "admin@example.com",
        "password": "password123",
        "name": "Admin User"
    });

    let register_req = test::TestRequest::post()
        .uri("/api/v1/auth/register")
        .set_json(&register_body)
        .to_request();

    let register_resp: AuthResponse = test::call_and_read_body_json(&app, register_req).await;

    // Use the token to access protected route
    let req = test::TestRequest::get()
        .uri("/api/v1/users")
        .insert_header(("Authorization", format!("Bearer {}", register_resp.token)))
        .to_request();

    let resp = test::call_service(&app, req).await;
    assert!(resp.status().is_success());
}
```

---

## Framework Comparison

| Feature | Actix-web | Axum | Rocket |
|---------|-----------|------|--------|
| Performance | Fastest | Very Fast | Fast |
| Async | Yes | Yes | Yes |
| Type Safety | Good | Excellent | Excellent |
| Middleware | Custom (Transform/Service) | Tower | Fairings |
| Maturity | Very Mature | Growing | Mature |
| Documentation | Excellent | Good | Excellent |
| Community | Large | Growing | Large |
