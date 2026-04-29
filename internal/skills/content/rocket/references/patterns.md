# Rocket Patterns Reference

> Concrete, copy-pasteable patterns that complement the core guardrails in SKILL.md.
> For overview, project structure, and quick-start, see SKILL.md first.

---

## CRUD Handler Patterns

### Complete User Routes Module

```rust
// src/routes/mod.rs
use rocket::Route;

mod health;
mod users;

pub fn api_routes() -> Vec<Route> {
    routes![
        users::register,
        users::login,
        users::get_current_user,
        users::update_current_user,
        users::get_user,
        users::list_users,
        users::delete_user,
    ]
}

pub fn health_routes() -> Vec<Route> {
    routes![health::health_check, health::ready_check]
}
```

### User Handlers

```rust
// src/routes/users.rs
use crate::db::DbConn;
use crate::error::AppError;
use crate::guards::{AdminUser, AuthUser};
use crate::models::user::{CreateUser, LoginRequest, LoginResponse, UpdateUser, UserResponse};
use crate::services::UserService;
use rocket::serde::json::Json;
use uuid::Uuid;

#[post("/register", data = "<input>")]
pub async fn register(
    mut db: DbConn,
    input: Json<CreateUser>,
) -> Result<Json<UserResponse>, AppError> {
    let user = UserService::create_user(&mut **db, input.into_inner()).await?;
    Ok(Json(user))
}

#[post("/login", data = "<input>")]
pub async fn login(
    mut db: DbConn,
    input: Json<LoginRequest>,
) -> Result<Json<LoginResponse>, AppError> {
    let response = UserService::login(&mut **db, input.into_inner()).await?;
    Ok(Json(response))
}

#[get("/users/me")]
pub async fn get_current_user(
    mut db: DbConn,
    auth: AuthUser,
) -> Result<Json<UserResponse>, AppError> {
    let user = UserService::get_user(&mut **db, auth.user_id).await?;
    Ok(Json(user))
}

#[put("/users/me", data = "<input>")]
pub async fn update_current_user(
    mut db: DbConn,
    auth: AuthUser,
    input: Json<UpdateUser>,
) -> Result<Json<UserResponse>, AppError> {
    let user = UserService::update_user(&mut **db, auth.user_id, input.into_inner()).await?;
    Ok(Json(user))
}

#[get("/users/<id>")]
pub async fn get_user(
    mut db: DbConn,
    _auth: AuthUser,
    id: &str,
) -> Result<Json<UserResponse>, AppError> {
    let user_id = Uuid::parse_str(id)
        .map_err(|_| AppError::Validation("Invalid user ID".to_string()))?;
    let user = UserService::get_user(&mut **db, user_id).await?;
    Ok(Json(user))
}

#[get("/users?<limit>&<offset>")]
pub async fn list_users(
    mut db: DbConn,
    _auth: AdminUser,
    limit: Option<i64>,
    offset: Option<i64>,
) -> Result<Json<Vec<UserResponse>>, AppError> {
    let users = UserService::list_users(
        &mut **db,
        limit.unwrap_or(20),
        offset.unwrap_or(0),
    ).await?;
    Ok(Json(users))
}

#[delete("/users/<id>")]
pub async fn delete_user(
    mut db: DbConn,
    _auth: AdminUser,
    id: &str,
) -> Result<(), AppError> {
    let user_id = Uuid::parse_str(id)
        .map_err(|_| AppError::Validation("Invalid user ID".to_string()))?;
    UserService::delete_user(&mut **db, user_id).await?;
    Ok(())
}
```

### Health Check Handlers

```rust
// src/routes/health.rs
use rocket::serde::json::Json;
use rocket::serde::Serialize;
use crate::db::DbConn;

#[derive(Serialize)]
#[serde(crate = "rocket::serde")]
pub struct HealthResponse {
    pub status: String,
    pub version: String,
}

#[get("/")]
pub fn health_check() -> Json<HealthResponse> {
    Json(HealthResponse {
        status: "ok".to_string(),
        version: env!("CARGO_PKG_VERSION").to_string(),
    })
}

#[get("/ready")]
pub async fn ready_check(mut db: DbConn) -> Result<Json<HealthResponse>, &'static str> {
    sqlx::query("SELECT 1")
        .execute(&mut **db)
        .await
        .map_err(|_| "Database not ready")?;

    Ok(Json(HealthResponse {
        status: "ready".to_string(),
        version: env!("CARGO_PKG_VERSION").to_string(),
    }))
}
```

---

## Model Patterns

### Domain Model With Validation

```rust
// src/models/user.rs
use chrono::{DateTime, Utc};
use rocket::serde::{Deserialize, Serialize};
use sqlx::FromRow;
use uuid::Uuid;
use validator::Validate;

#[derive(Debug, Clone, Serialize, FromRow)]
#[serde(crate = "rocket::serde")]
pub struct User {
    pub id: Uuid,
    pub email: String,
    #[serde(skip_serializing)]
    pub password_hash: String,
    pub name: String,
    pub role: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Deserialize, Validate)]
#[serde(crate = "rocket::serde")]
pub struct CreateUser {
    #[validate(email(message = "Invalid email format"))]
    pub email: String,

    #[validate(length(min = 8, message = "Password must be at least 8 characters"))]
    pub password: String,

    #[validate(length(min = 1, max = 100, message = "Name must be 1-100 characters"))]
    pub name: String,
}

#[derive(Debug, Deserialize, Validate)]
#[serde(crate = "rocket::serde")]
pub struct UpdateUser {
    #[validate(length(min = 1, max = 100, message = "Name must be 1-100 characters"))]
    pub name: Option<String>,

    #[validate(email(message = "Invalid email format"))]
    pub email: Option<String>,
}

#[derive(Debug, Serialize)]
#[serde(crate = "rocket::serde")]
pub struct UserResponse {
    pub id: Uuid,
    pub email: String,
    pub name: String,
    pub role: String,
    pub created_at: DateTime<Utc>,
}

impl From<User> for UserResponse {
    fn from(user: User) -> Self {
        Self {
            id: user.id,
            email: user.email,
            name: user.name,
            role: user.role,
            created_at: user.created_at,
        }
    }
}
```

### Login Request/Response Models

```rust
#[derive(Debug, Deserialize, Validate)]
#[serde(crate = "rocket::serde")]
pub struct LoginRequest {
    #[validate(email)]
    pub email: String,
    pub password: String,
}

#[derive(Debug, Serialize)]
#[serde(crate = "rocket::serde")]
pub struct LoginResponse {
    pub token: String,
    pub user: UserResponse,
}
```

---

## Database Patterns (SQLx)

### Service Layer With SQLx

```rust
// src/services/user_service.rs
use crate::error::AppError;
use crate::models::user::{CreateUser, UpdateUser, User, UserResponse};
use bcrypt::{hash, verify, DEFAULT_COST};
use chrono::Utc;
use sqlx::PgPool;
use uuid::Uuid;
use validator::Validate;

pub struct UserService;

impl UserService {
    pub async fn create_user(pool: &PgPool, input: CreateUser) -> Result<UserResponse, AppError> {
        input.validate().map_err(|e| AppError::Validation(e.to_string()))?;

        // Check uniqueness
        let existing = sqlx::query_scalar::<_, i64>(
            "SELECT COUNT(*) FROM users WHERE email = $1"
        )
        .bind(&input.email)
        .fetch_one(pool)
        .await?;

        if existing > 0 {
            return Err(AppError::Validation("Email already registered".to_string()));
        }

        let password_hash = hash(&input.password, DEFAULT_COST)
            .map_err(|e| AppError::Internal(e.to_string()))?;

        let user = sqlx::query_as::<_, User>(
            r#"INSERT INTO users (id, email, password_hash, name, role, created_at, updated_at)
               VALUES ($1, $2, $3, $4, 'user', $5, $5)
               RETURNING *"#,
        )
        .bind(Uuid::new_v4())
        .bind(&input.email)
        .bind(&password_hash)
        .bind(&input.name)
        .bind(Utc::now())
        .fetch_one(pool)
        .await?;

        Ok(user.into())
    }

    pub async fn get_user(pool: &PgPool, user_id: Uuid) -> Result<UserResponse, AppError> {
        let user = sqlx::query_as::<_, User>("SELECT * FROM users WHERE id = $1")
            .bind(user_id)
            .fetch_optional(pool)
            .await?
            .ok_or_else(|| AppError::NotFound(format!("User {} not found", user_id)))?;

        Ok(user.into())
    }

    pub async fn update_user(
        pool: &PgPool,
        user_id: Uuid,
        input: UpdateUser,
    ) -> Result<UserResponse, AppError> {
        input.validate().map_err(|e| AppError::Validation(e.to_string()))?;

        let user = sqlx::query_as::<_, User>(
            r#"UPDATE users
               SET name = COALESCE($2, name),
                   email = COALESCE($3, email),
                   updated_at = $4
               WHERE id = $1
               RETURNING *"#,
        )
        .bind(user_id)
        .bind(&input.name)
        .bind(&input.email)
        .bind(Utc::now())
        .fetch_optional(pool)
        .await?
        .ok_or_else(|| AppError::NotFound(format!("User {} not found", user_id)))?;

        Ok(user.into())
    }

    pub async fn delete_user(pool: &PgPool, user_id: Uuid) -> Result<(), AppError> {
        let result = sqlx::query("DELETE FROM users WHERE id = $1")
            .bind(user_id)
            .execute(pool)
            .await?;

        if result.rows_affected() == 0 {
            return Err(AppError::NotFound(format!("User {} not found", user_id)));
        }
        Ok(())
    }

    pub async fn list_users(
        pool: &PgPool,
        limit: i64,
        offset: i64,
    ) -> Result<Vec<UserResponse>, AppError> {
        let users = sqlx::query_as::<_, User>(
            "SELECT * FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2"
        )
        .bind(limit)
        .bind(offset)
        .fetch_all(pool)
        .await?;

        Ok(users.into_iter().map(|u| u.into()).collect())
    }
}
```

---

## Authentication Patterns

### JWT Claims and Token Generation

```rust
use jsonwebtoken::{decode, encode, DecodingKey, EncodingKey, Header, Validation};
use rocket::serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Debug, Serialize, Deserialize)]
pub struct Claims {
    pub sub: Uuid,
    pub email: String,
    pub role: String,
    pub exp: i64,
}
```

### Full Auth Guard Implementation

```rust
// src/guards/auth.rs
use crate::config::AppConfig;
use crate::error::AppError;
use rocket::http::Status;
use rocket::request::{FromRequest, Outcome, Request};

#[derive(Debug)]
pub struct AuthUser {
    pub user_id: Uuid,
    pub email: String,
    pub role: String,
}

#[rocket::async_trait]
impl<'r> FromRequest<'r> for AuthUser {
    type Error = AppError;

    async fn from_request(request: &'r Request<'_>) -> Outcome<Self, Self::Error> {
        let auth_header = request.headers().get_one("Authorization");

        let token = match auth_header {
            Some(header) if header.starts_with("Bearer ") => &header[7..],
            _ => return Outcome::Error((
                Status::Unauthorized,
                AppError::Unauthorized("Missing or invalid Authorization header".into()),
            )),
        };

        let config = AppConfig::default();

        let token_data = match decode::<Claims>(
            token,
            &DecodingKey::from_secret(config.jwt_secret.as_bytes()),
            &Validation::default(),
        ) {
            Ok(data) => data,
            Err(e) => return Outcome::Error((
                Status::Unauthorized,
                AppError::Unauthorized(format!("Invalid token: {}", e)),
            )),
        };

        Outcome::Success(AuthUser {
            user_id: token_data.claims.sub,
            email: token_data.claims.email,
            role: token_data.claims.role,
        })
    }
}
```

### Login Service With Token Generation

```rust
use chrono::{Duration, Utc};

pub async fn login(pool: &PgPool, input: LoginRequest) -> Result<LoginResponse, AppError> {
    input.validate().map_err(|e| AppError::Validation(e.to_string()))?;

    let user = sqlx::query_as::<_, User>("SELECT * FROM users WHERE email = $1")
        .bind(&input.email)
        .fetch_optional(pool)
        .await?
        .ok_or_else(|| AppError::Unauthorized("Invalid credentials".into()))?;

    let valid = verify(&input.password, &user.password_hash)
        .map_err(|e| AppError::Internal(e.to_string()))?;

    if !valid {
        return Err(AppError::Unauthorized("Invalid credentials".into()));
    }

    let config = AppConfig::default();
    let expiration = Utc::now() + Duration::hours(config.jwt_expiration_hours);

    let claims = Claims {
        sub: user.id,
        email: user.email.clone(),
        role: user.role.clone(),
        exp: expiration.timestamp(),
    };

    let token = encode(
        &Header::default(),
        &claims,
        &EncodingKey::from_secret(config.jwt_secret.as_bytes()),
    )
    .map_err(|e| AppError::Internal(e.to_string()))?;

    Ok(LoginResponse { token, user: user.into() })
}
```

### Optional Auth Guard

```rust
#[derive(Debug)]
pub struct OptionalUser(pub Option<AuthUser>);

#[rocket::async_trait]
impl<'r> FromRequest<'r> for OptionalUser {
    type Error = std::convert::Infallible;

    async fn from_request(request: &'r Request<'_>) -> Outcome<Self, Self::Error> {
        match AuthUser::from_request(request).await {
            Outcome::Success(user) => Outcome::Success(OptionalUser(Some(user))),
            _ => Outcome::Success(OptionalUser(None)),
        }
    }
}
```

---

## Error Handling Patterns

### Complete AppError With Responder

```rust
// src/error.rs
use rocket::http::Status;
use rocket::response::{self, Responder, Response};
use rocket::serde::json::Json;
use rocket::Request;
use serde::Serialize;
use thiserror::Error;

#[derive(Error, Debug)]
pub enum AppError {
    #[error("Resource not found: {0}")]
    NotFound(String),
    #[error("Unauthorized: {0}")]
    Unauthorized(String),
    #[error("Validation error: {0}")]
    Validation(String),
    #[error("Database error: {0}")]
    Database(#[from] sqlx::Error),
    #[error("Internal server error")]
    Internal(String),
}

#[derive(Serialize)]
pub struct ErrorResponse {
    pub error: String,
    pub message: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub details: Option<Vec<String>>,
}

impl<'r> Responder<'r, 'static> for AppError {
    fn respond_to(self, request: &'r Request<'_>) -> response::Result<'static> {
        let (status, error_response) = match &self {
            AppError::NotFound(msg) => (Status::NotFound, ErrorResponse {
                error: "NOT_FOUND".into(), message: msg.clone(), details: None,
            }),
            AppError::Unauthorized(msg) => (Status::Unauthorized, ErrorResponse {
                error: "UNAUTHORIZED".into(), message: msg.clone(), details: None,
            }),
            AppError::Validation(msg) => (Status::BadRequest, ErrorResponse {
                error: "VALIDATION_ERROR".into(), message: msg.clone(), details: None,
            }),
            AppError::Database(e) => {
                eprintln!("Database error: {:?}", e);
                (Status::InternalServerError, ErrorResponse {
                    error: "DATABASE_ERROR".into(),
                    message: "A database error occurred".into(),
                    details: None,
                })
            }
            AppError::Internal(msg) => {
                eprintln!("Internal error: {}", msg);
                (Status::InternalServerError, ErrorResponse {
                    error: "INTERNAL_ERROR".into(),
                    message: "An internal error occurred".into(),
                    details: None,
                })
            }
        };

        Response::build_from(Json(error_response).respond_to(request)?)
            .status(status)
            .ok()
    }
}
```

### Error Catchers

```rust
#[catch(404)]
pub fn not_found() -> Json<ErrorResponse> {
    Json(ErrorResponse {
        error: "NOT_FOUND".into(),
        message: "The requested resource was not found".into(),
        details: None,
    })
}

#[catch(500)]
pub fn internal_error() -> Json<ErrorResponse> {
    Json(ErrorResponse {
        error: "INTERNAL_ERROR".into(),
        message: "An internal server error occurred".into(),
        details: None,
    })
}

#[catch(401)]
pub fn unauthorized() -> Json<ErrorResponse> {
    Json(ErrorResponse {
        error: "UNAUTHORIZED".into(),
        message: "Authentication required".into(),
        details: None,
    })
}

#[catch(400)]
pub fn bad_request() -> Json<ErrorResponse> {
    Json(ErrorResponse {
        error: "BAD_REQUEST".into(),
        message: "Invalid request".into(),
        details: None,
    })
}
```

---

## Testing Patterns

### Test Client Setup

```rust
// tests/api_tests.rs
use rocket::http::{ContentType, Header, Status};
use rocket::local::asynchronous::Client;
use serde_json::{json, Value};

async fn get_client() -> Client {
    let rocket = myproject::rocket();
    Client::tracked(rocket).await.expect("valid rocket instance")
}
```

### Health Check Test

```rust
#[rocket::async_test]
async fn test_health_check() {
    let client = get_client().await;
    let response = client.get("/health").dispatch().await;

    assert_eq!(response.status(), Status::Ok);

    let body: Value = response.into_json().await.unwrap();
    assert_eq!(body["status"], "ok");
}
```

### Registration Tests

```rust
#[rocket::async_test]
async fn test_register_user() {
    let client = get_client().await;

    let response = client
        .post("/api/register")
        .header(ContentType::JSON)
        .body(json!({
            "email": "test@example.com",
            "password": "password123",
            "name": "Test User"
        }).to_string())
        .dispatch()
        .await;

    assert_eq!(response.status(), Status::Ok);

    let body: Value = response.into_json().await.unwrap();
    assert_eq!(body["email"], "test@example.com");
    assert_eq!(body["name"], "Test User");
}

#[rocket::async_test]
async fn test_register_invalid_email() {
    let client = get_client().await;

    let response = client
        .post("/api/register")
        .header(ContentType::JSON)
        .body(json!({
            "email": "invalid-email",
            "password": "password123",
            "name": "Test User"
        }).to_string())
        .dispatch()
        .await;

    assert_eq!(response.status(), Status::BadRequest);
}
```

### Authenticated Request Tests

```rust
#[rocket::async_test]
async fn test_login_and_access_protected_route() {
    let client = get_client().await;

    // Register
    client
        .post("/api/register")
        .header(ContentType::JSON)
        .body(json!({
            "email": "login_test@example.com",
            "password": "password123",
            "name": "Login Test"
        }).to_string())
        .dispatch()
        .await;

    // Login
    let login_response = client
        .post("/api/login")
        .header(ContentType::JSON)
        .body(json!({
            "email": "login_test@example.com",
            "password": "password123"
        }).to_string())
        .dispatch()
        .await;

    assert_eq!(login_response.status(), Status::Ok);

    let login_body: Value = login_response.into_json().await.unwrap();
    let token = login_body["token"].as_str().unwrap();

    // Access protected route
    let me_response = client
        .get("/api/users/me")
        .header(Header::new("Authorization", format!("Bearer {}", token)))
        .dispatch()
        .await;

    assert_eq!(me_response.status(), Status::Ok);

    let me_body: Value = me_response.into_json().await.unwrap();
    assert_eq!(me_body["email"], "login_test@example.com");
}
```

### Authorization Failure Tests

```rust
#[rocket::async_test]
async fn test_unauthorized_access() {
    let client = get_client().await;
    let response = client.get("/api/users/me").dispatch().await;
    assert_eq!(response.status(), Status::Unauthorized);
}

#[rocket::async_test]
async fn test_invalid_token() {
    let client = get_client().await;

    let response = client
        .get("/api/users/me")
        .header(Header::new("Authorization", "Bearer invalid_token"))
        .dispatch()
        .await;

    assert_eq!(response.status(), Status::Unauthorized);
}
```

---

## Migration Patterns

### From Actix-web to Rocket

```rust
// Actix-web
#[get("/users/{id}")]
async fn get_user(path: web::Path<Uuid>) -> impl Responder {
    // ...
}

// Rocket equivalent
#[get("/users/<id>")]
async fn get_user(id: &str) -> Result<Json<UserResponse>, AppError> {
    let user_id = Uuid::parse_str(id)?;
    // ...
}
```

### From Axum to Rocket

```rust
// Axum
async fn get_user(
    State(pool): State<PgPool>,
    Path(id): Path<Uuid>,
) -> Result<Json<User>, AppError> {
    // ...
}

// Rocket equivalent
#[get("/users/<id>")]
async fn get_user(
    mut db: DbConn,
    id: &str,
) -> Result<Json<UserResponse>, AppError> {
    let user_id = Uuid::parse_str(id)?;
    // ...
}
```

### Key Differences

| Concept | Actix-web / Axum | Rocket |
|---------|------------------|--------|
| **Routing** | Path macro or Router | Attribute macros (`#[get]`, `#[post]`) |
| **Extractors** | `web::Path`, `State`, `Path` | Function params, request guards |
| **Middleware** | Service / Layer / Tower | Fairings |
| **Auth** | Manual middleware | Request guards (`FromRequest`) |
| **Database** | Manual pool setup | `rocket_db_pools` with `.attach()` |
| **Config** | Manual | `Rocket.toml` with profile support |
| **Error handling** | `IntoResponse` | `Responder` trait |
