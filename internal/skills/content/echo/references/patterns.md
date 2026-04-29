# Echo Patterns Reference

## Contents

- [Full Application Entry Point](#full-application-entry-point)
- [Configuration](#configuration)
- [Models and DTOs](#models-and-dtos)
- [Repository Layer](#repository-layer)
- [Service Layer](#service-layer)
- [Handler Examples](#handler-examples)
- [Authentication](#authentication)
- [Auth Middleware](#auth-middleware)
- [Paginated Responses](#paginated-responses)
- [Database Connection](#database-connection)
- [Testing](#testing)
- [Dependencies](#dependencies)

## Full Application Entry Point

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "time"

    "myproject/internal/config"
    "myproject/internal/handler"
    "myproject/internal/middleware"
    "myproject/internal/repository"
    "myproject/internal/service"
    "myproject/internal/validator"

    "github.com/labstack/echo/v4"
    echoMw "github.com/labstack/echo/v4/middleware"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    db, err := initDB(cfg.DatabaseURL)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }

    e := echo.New()
    e.HideBanner = true
    e.Validator = validator.NewCustomValidator()

    // Global middleware
    e.Use(echoMw.Recover())
    e.Use(echoMw.Logger())
    e.Use(echoMw.RequestID())
    e.Use(echoMw.CORSWithConfig(echoMw.CORSConfig{
        AllowOrigins: cfg.CORSOrigins,
        AllowMethods: []string{
            http.MethodGet, http.MethodPost,
            http.MethodPut, http.MethodPatch,
            http.MethodDelete,
        },
        AllowHeaders: []string{
            echo.HeaderOrigin, echo.HeaderContentType,
            echo.HeaderAccept, echo.HeaderAuthorization,
        },
    }))

    // Initialize layers
    repos := repository.NewRepositories(db)
    services := service.NewServices(repos, cfg)
    handlers := handler.NewHandlers(services)
    authMiddleware := middleware.NewAuthMiddleware(services.Auth)

    setupRoutes(e, handlers, authMiddleware)

    // Start server with graceful shutdown
    go func() {
        if err := e.Start(":" + cfg.Port); err != nil &&
            err != http.ErrServerClosed {
            e.Logger.Fatal("Shutting down the server")
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt)
    <-quit

    ctx, cancel := context.WithTimeout(
        context.Background(), 10*time.Second,
    )
    defer cancel()

    if err := e.Shutdown(ctx); err != nil {
        e.Logger.Fatal(err)
    }
}

func setupRoutes(
    e *echo.Echo,
    h *handler.Handlers,
    authMw *middleware.AuthMiddleware,
) {
    e.GET("/health", func(c echo.Context) error {
        return c.JSON(http.StatusOK,
            map[string]string{"status": "ok"})
    })

    v1 := e.Group("/api/v1")

    auth := v1.Group("/auth")
    auth.POST("/login", h.Auth.Login)
    auth.POST("/register", h.Auth.Register)
    auth.POST("/refresh", h.Auth.Refresh)

    users := v1.Group("/users")
    users.POST("", h.User.Create)
    users.Use(authMw.Authenticate)
    users.GET("", h.User.GetAll)
    users.GET("/me", h.User.GetCurrent)
    users.GET("/:id", h.User.GetByID)
    users.PUT("/:id", h.User.Update)
    users.DELETE("/:id", h.User.Delete, authMw.RequireAdmin)
}
```

## Configuration

```go
package config

import (
    "os"
    "strings"
    "time"

    "github.com/joho/godotenv"
)

type Config struct {
    Environment        string
    Port               string
    DatabaseURL        string
    JWTSecret          string
    JWTAccessDuration  time.Duration
    JWTRefreshDuration time.Duration
    CORSOrigins        []string
}

func Load() (*Config, error) {
    _ = godotenv.Load()

    return &Config{
        Environment: getEnv("ENVIRONMENT", "development"),
        Port:        getEnv("PORT", "8080"),
        DatabaseURL: getEnv("DATABASE_URL",
            "postgres://postgres:postgres@localhost:5432/myapp?sslmode=disable"),
        JWTSecret:          getEnv("JWT_SECRET", "your-secret-key"),
        JWTAccessDuration:  parseDuration(getEnv("JWT_ACCESS_DURATION", "15m")),
        JWTRefreshDuration: parseDuration(getEnv("JWT_REFRESH_DURATION", "168h")),
        CORSOrigins:        strings.Split(getEnv("CORS_ORIGINS", "*"), ","),
    }, nil
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func parseDuration(s string) time.Duration {
    d, err := time.ParseDuration(s)
    if err != nil {
        return 15 * time.Minute
    }
    return d
}
```

## Models and DTOs

```go
package model

import (
    "time"

    "golang.org/x/crypto/bcrypt"
    "gorm.io/gorm"
)

// Domain model
type User struct {
    ID        uint           `json:"id" gorm:"primaryKey"`
    Email     string         `json:"email" gorm:"uniqueIndex;not null"`
    Password  string         `json:"-" gorm:"not null"`
    FirstName string         `json:"first_name" gorm:"not null"`
    LastName  string         `json:"last_name" gorm:"not null"`
    Role      string         `json:"role" gorm:"default:user"`
    IsActive  bool           `json:"is_active" gorm:"default:true"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (u *User) SetPassword(password string) error {
    hash, err := bcrypt.GenerateFromPassword(
        []byte(password), bcrypt.DefaultCost,
    )
    if err != nil {
        return err
    }
    u.Password = string(hash)
    return nil
}

func (u *User) CheckPassword(password string) bool {
    err := bcrypt.CompareHashAndPassword(
        []byte(u.Password), []byte(password),
    )
    return err == nil
}

// Request DTOs
type CreateUserRequest struct {
    Email     string `json:"email" validate:"required,email"`
    Password  string `json:"password" validate:"required,min=8"`
    FirstName string `json:"first_name" validate:"required,min=1,max=100"`
    LastName  string `json:"last_name" validate:"required,min=1,max=100"`
}

type UpdateUserRequest struct {
    FirstName *string `json:"first_name" validate:"omitempty,min=1,max=100"`
    LastName  *string `json:"last_name" validate:"omitempty,min=1,max=100"`
    IsActive  *bool   `json:"is_active"`
}

type LoginRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required"`
}

// Response DTOs (never expose passwords or internal fields)
type UserResponse struct {
    ID        uint      `json:"id"`
    Email     string    `json:"email"`
    FirstName string    `json:"first_name"`
    LastName  string    `json:"last_name"`
    Role      string    `json:"role"`
    IsActive  bool      `json:"is_active"`
    CreatedAt time.Time `json:"created_at"`
}

func (u *User) ToResponse() *UserResponse {
    return &UserResponse{
        ID:        u.ID,
        Email:     u.Email,
        FirstName: u.FirstName,
        LastName:  u.LastName,
        Role:      u.Role,
        IsActive:  u.IsActive,
        CreatedAt: u.CreatedAt,
    }
}

type TokenResponse struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    TokenType    string `json:"token_type"`
    ExpiresIn    int64  `json:"expires_in"`
}
```

## Repository Layer

```go
package repository

import (
    "context"

    "myproject/internal/model"

    "gorm.io/gorm"
)

type UserRepository interface {
    Create(ctx context.Context, user *model.User) error
    GetByID(ctx context.Context, id uint) (*model.User, error)
    GetByEmail(ctx context.Context, email string) (*model.User, error)
    GetAll(ctx context.Context, page, perPage int) ([]model.User, int64, error)
    Update(ctx context.Context, user *model.User) error
    Delete(ctx context.Context, id uint) error
}

type userRepository struct {
    db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
    return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
    return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) GetByID(ctx context.Context, id uint) (*model.User, error) {
    var user model.User
    if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
        return nil, err
    }
    return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
    var user model.User
    if err := r.db.WithContext(ctx).
        Where("email = ?", email).First(&user).Error; err != nil {
        return nil, err
    }
    return &user, nil
}

func (r *userRepository) GetAll(
    ctx context.Context, page, perPage int,
) ([]model.User, int64, error) {
    var users []model.User
    var total int64

    offset := (page - 1) * perPage

    if err := r.db.WithContext(ctx).
        Model(&model.User{}).Count(&total).Error; err != nil {
        return nil, 0, err
    }

    if err := r.db.WithContext(ctx).
        Offset(offset).Limit(perPage).
        Order("created_at DESC").
        Find(&users).Error; err != nil {
        return nil, 0, err
    }

    return users, total, nil
}

func (r *userRepository) Update(ctx context.Context, user *model.User) error {
    return r.db.WithContext(ctx).Save(user).Error
}

func (r *userRepository) Delete(ctx context.Context, id uint) error {
    return r.db.WithContext(ctx).Delete(&model.User{}, id).Error
}
```

## Service Layer

```go
package service

import (
    "context"
    "errors"

    "myproject/internal/model"
    "myproject/internal/repository"

    "gorm.io/gorm"
)

var (
    ErrUserNotFound       = errors.New("user not found")
    ErrUserAlreadyExists  = errors.New("user already exists")
    ErrInvalidCredentials = errors.New("invalid credentials")
)

type UserService interface {
    Create(ctx context.Context, req *model.CreateUserRequest) (*model.User, error)
    GetByID(ctx context.Context, id uint) (*model.User, error)
    GetAll(ctx context.Context, page, perPage int) ([]model.User, int64, error)
    Update(ctx context.Context, id uint, req *model.UpdateUserRequest) (*model.User, error)
    Delete(ctx context.Context, id uint) error
}

type userService struct {
    userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) UserService {
    return &userService{userRepo: userRepo}
}

func (s *userService) Create(
    ctx context.Context, req *model.CreateUserRequest,
) (*model.User, error) {
    existing, _ := s.userRepo.GetByEmail(ctx, req.Email)
    if existing != nil {
        return nil, ErrUserAlreadyExists
    }

    user := &model.User{
        Email:     req.Email,
        FirstName: req.FirstName,
        LastName:  req.LastName,
        Role:      "user",
    }

    if err := user.SetPassword(req.Password); err != nil {
        return nil, err
    }

    if err := s.userRepo.Create(ctx, user); err != nil {
        return nil, err
    }

    return user, nil
}

func (s *userService) GetByID(ctx context.Context, id uint) (*model.User, error) {
    user, err := s.userRepo.GetByID(ctx, id)
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, ErrUserNotFound
        }
        return nil, err
    }
    return user, nil
}

func (s *userService) GetAll(
    ctx context.Context, page, perPage int,
) ([]model.User, int64, error) {
    if page < 1 {
        page = 1
    }
    if perPage < 1 || perPage > 100 {
        perPage = 20
    }
    return s.userRepo.GetAll(ctx, page, perPage)
}

func (s *userService) Update(
    ctx context.Context, id uint, req *model.UpdateUserRequest,
) (*model.User, error) {
    user, err := s.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }

    if req.FirstName != nil {
        user.FirstName = *req.FirstName
    }
    if req.LastName != nil {
        user.LastName = *req.LastName
    }
    if req.IsActive != nil {
        user.IsActive = *req.IsActive
    }

    if err := s.userRepo.Update(ctx, user); err != nil {
        return nil, err
    }

    return user, nil
}

func (s *userService) Delete(ctx context.Context, id uint) error {
    if _, err := s.GetByID(ctx, id); err != nil {
        return err
    }
    return s.userRepo.Delete(ctx, id)
}
```

## Handler Examples

### User CRUD Handlers

```go
package handler

import (
    "errors"
    "net/http"
    "strconv"

    "myproject/internal/model"
    "myproject/internal/service"

    "github.com/labstack/echo/v4"
)

type UserHandler struct {
    userService service.UserService
}

func NewUserHandler(userService service.UserService) *UserHandler {
    return &UserHandler{userService: userService}
}

func (h *UserHandler) GetAll(c echo.Context) error {
    page, _ := strconv.Atoi(c.QueryParam("page"))
    if page < 1 {
        page = 1
    }
    perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
    if perPage < 1 {
        perPage = 20
    }

    users, total, err := h.userService.GetAll(
        c.Request().Context(), page, perPage,
    )
    if err != nil {
        return c.JSON(http.StatusInternalServerError,
            model.ErrorResponse(err.Error()))
    }

    responses := make([]*model.UserResponse, len(users))
    for i, user := range users {
        responses[i] = user.ToResponse()
    }

    totalPages := int(total) / perPage
    if int(total)%perPage > 0 {
        totalPages++
    }

    return c.JSON(http.StatusOK, &model.PaginatedResponse{
        Success: true,
        Data:    responses,
        Meta: model.PageMeta{
            Page:       page,
            PerPage:    perPage,
            Total:      total,
            TotalPages: totalPages,
        },
    })
}

func (h *UserHandler) GetByID(c echo.Context) error {
    id, err := strconv.ParseUint(c.Param("id"), 10, 32)
    if err != nil {
        return c.JSON(http.StatusBadRequest,
            model.ErrorResponse("invalid user ID"))
    }

    user, err := h.userService.GetByID(
        c.Request().Context(), uint(id),
    )
    if err != nil {
        if errors.Is(err, service.ErrUserNotFound) {
            return c.JSON(http.StatusNotFound,
                model.ErrorResponse(err.Error()))
        }
        return c.JSON(http.StatusInternalServerError,
            model.ErrorResponse(err.Error()))
    }

    return c.JSON(http.StatusOK,
        model.SuccessResponse(user.ToResponse()))
}

func (h *UserHandler) GetCurrent(c echo.Context) error {
    userID, ok := c.Get("user_id").(uint)
    if !ok {
        return c.JSON(http.StatusUnauthorized,
            model.ErrorResponse("unauthorized"))
    }

    user, err := h.userService.GetByID(
        c.Request().Context(), userID,
    )
    if err != nil {
        return c.JSON(http.StatusNotFound,
            model.ErrorResponse(err.Error()))
    }

    return c.JSON(http.StatusOK,
        model.SuccessResponse(user.ToResponse()))
}

func (h *UserHandler) Create(c echo.Context) error {
    var req model.CreateUserRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest,
            model.ErrorResponse(err.Error()))
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    user, err := h.userService.Create(
        c.Request().Context(), &req,
    )
    if err != nil {
        if errors.Is(err, service.ErrUserAlreadyExists) {
            return c.JSON(http.StatusConflict,
                model.ErrorResponse(err.Error()))
        }
        return c.JSON(http.StatusInternalServerError,
            model.ErrorResponse(err.Error()))
    }

    return c.JSON(http.StatusCreated,
        model.SuccessResponse(user.ToResponse()))
}

func (h *UserHandler) Update(c echo.Context) error {
    id, err := strconv.ParseUint(c.Param("id"), 10, 32)
    if err != nil {
        return c.JSON(http.StatusBadRequest,
            model.ErrorResponse("invalid user ID"))
    }

    var req model.UpdateUserRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest,
            model.ErrorResponse(err.Error()))
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    user, err := h.userService.Update(
        c.Request().Context(), uint(id), &req,
    )
    if err != nil {
        if errors.Is(err, service.ErrUserNotFound) {
            return c.JSON(http.StatusNotFound,
                model.ErrorResponse(err.Error()))
        }
        return c.JSON(http.StatusInternalServerError,
            model.ErrorResponse(err.Error()))
    }

    return c.JSON(http.StatusOK,
        model.SuccessResponse(user.ToResponse()))
}

func (h *UserHandler) Delete(c echo.Context) error {
    id, err := strconv.ParseUint(c.Param("id"), 10, 32)
    if err != nil {
        return c.JSON(http.StatusBadRequest,
            model.ErrorResponse("invalid user ID"))
    }

    err = h.userService.Delete(
        c.Request().Context(), uint(id),
    )
    if err != nil {
        if errors.Is(err, service.ErrUserNotFound) {
            return c.JSON(http.StatusNotFound,
                model.ErrorResponse(err.Error()))
        }
        return c.JSON(http.StatusInternalServerError,
            model.ErrorResponse(err.Error()))
    }

    return c.NoContent(http.StatusNoContent)
}
```

### Auth Handlers

```go
package handler

import (
    "net/http"

    "myproject/internal/model"
    "myproject/internal/service"

    "github.com/labstack/echo/v4"
)

type AuthHandler struct {
    authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
    return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Login(c echo.Context) error {
    var req model.LoginRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest,
            model.ErrorResponse(err.Error()))
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    tokens, err := h.authService.Login(
        c.Request().Context(), &req,
    )
    if err != nil {
        return c.JSON(http.StatusUnauthorized,
            model.ErrorResponse(err.Error()))
    }

    return c.JSON(http.StatusOK, tokens)
}

func (h *AuthHandler) Register(c echo.Context) error {
    var req model.CreateUserRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest,
            model.ErrorResponse(err.Error()))
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    user, err := h.authService.Register(
        c.Request().Context(), &req,
    )
    if err != nil {
        if err == service.ErrUserAlreadyExists {
            return c.JSON(http.StatusConflict,
                model.ErrorResponse(err.Error()))
        }
        return c.JSON(http.StatusInternalServerError,
            model.ErrorResponse(err.Error()))
    }

    return c.JSON(http.StatusCreated,
        model.SuccessResponse(user.ToResponse()))
}

func (h *AuthHandler) Refresh(c echo.Context) error {
    var req struct {
        RefreshToken string `json:"refresh_token" validate:"required"`
    }
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest,
            model.ErrorResponse(err.Error()))
    }

    tokens, err := h.authService.RefreshToken(
        c.Request().Context(), req.RefreshToken,
    )
    if err != nil {
        return c.JSON(http.StatusUnauthorized,
            model.ErrorResponse(err.Error()))
    }

    return c.JSON(http.StatusOK, tokens)
}
```

## Authentication

### JWT Auth Service

```go
package service

import (
    "context"
    "errors"
    "time"

    "myproject/internal/config"
    "myproject/internal/model"
    "myproject/internal/repository"

    "github.com/golang-jwt/jwt/v5"
)

type AuthService interface {
    Login(ctx context.Context, req *model.LoginRequest) (*model.TokenResponse, error)
    Register(ctx context.Context, req *model.CreateUserRequest) (*model.User, error)
    RefreshToken(ctx context.Context, refreshToken string) (*model.TokenResponse, error)
    ValidateToken(tokenString string) (*Claims, error)
}

type Claims struct {
    UserID uint   `json:"user_id"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}

type authService struct {
    userRepo repository.UserRepository
    cfg      *config.Config
}

func NewAuthService(
    userRepo repository.UserRepository, cfg *config.Config,
) AuthService {
    return &authService{userRepo: userRepo, cfg: cfg}
}

func (s *authService) Login(
    ctx context.Context, req *model.LoginRequest,
) (*model.TokenResponse, error) {
    user, err := s.userRepo.GetByEmail(ctx, req.Email)
    if err != nil {
        return nil, ErrInvalidCredentials
    }
    if !user.CheckPassword(req.Password) {
        return nil, ErrInvalidCredentials
    }
    if !user.IsActive {
        return nil, errors.New("account is deactivated")
    }
    return s.generateTokens(user)
}

func (s *authService) ValidateToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(
        tokenString, &Claims{},
        func(token *jwt.Token) (interface{}, error) {
            return []byte(s.cfg.JWTSecret), nil
        },
    )
    if err != nil {
        return nil, err
    }
    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, errors.New("invalid token")
    }
    return claims, nil
}

func (s *authService) generateTokens(
    user *model.User,
) (*model.TokenResponse, error) {
    now := time.Now()

    accessClaims := &Claims{
        UserID: user.ID,
        Role:   user.Role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(
                now.Add(s.cfg.JWTAccessDuration)),
            IssuedAt: jwt.NewNumericDate(now),
        },
    }

    accessToken := jwt.NewWithClaims(
        jwt.SigningMethodHS256, accessClaims,
    )
    accessTokenString, err := accessToken.SignedString(
        []byte(s.cfg.JWTSecret),
    )
    if err != nil {
        return nil, err
    }

    refreshClaims := &Claims{
        UserID: user.ID,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(
                now.Add(s.cfg.JWTRefreshDuration)),
            IssuedAt: jwt.NewNumericDate(now),
        },
    }

    refreshToken := jwt.NewWithClaims(
        jwt.SigningMethodHS256, refreshClaims,
    )
    refreshTokenString, err := refreshToken.SignedString(
        []byte(s.cfg.JWTSecret),
    )
    if err != nil {
        return nil, err
    }

    return &model.TokenResponse{
        AccessToken:  accessTokenString,
        RefreshToken: refreshTokenString,
        TokenType:    "Bearer",
        ExpiresIn:    int64(s.cfg.JWTAccessDuration.Seconds()),
    }, nil
}
```

## Auth Middleware

```go
package middleware

import (
    "net/http"
    "strings"

    "myproject/internal/model"
    "myproject/internal/service"

    "github.com/labstack/echo/v4"
)

type AuthMiddleware struct {
    authService service.AuthService
}

func NewAuthMiddleware(authService service.AuthService) *AuthMiddleware {
    return &AuthMiddleware{authService: authService}
}

// Authenticate validates JWT token from Authorization header
func (m *AuthMiddleware) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        header := c.Request().Header.Get("Authorization")
        if header == "" {
            return c.JSON(http.StatusUnauthorized,
                model.ErrorResponse("missing authorization header"))
        }

        parts := strings.Split(header, " ")
        if len(parts) != 2 || parts[0] != "Bearer" {
            return c.JSON(http.StatusUnauthorized,
                model.ErrorResponse("invalid authorization header"))
        }

        claims, err := m.authService.ValidateToken(parts[1])
        if err != nil {
            return c.JSON(http.StatusUnauthorized,
                model.ErrorResponse("invalid token"))
        }

        c.Set("user_id", claims.UserID)
        c.Set("user_role", claims.Role)

        return next(c)
    }
}

// RequireAdmin requires admin role (use after Authenticate)
func (m *AuthMiddleware) RequireAdmin(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        role, ok := c.Get("user_role").(string)
        if !ok || role != "admin" {
            return c.JSON(http.StatusForbidden,
                model.ErrorResponse("admin access required"))
        }
        return next(c)
    }
}
```

## Paginated Responses

```go
package model

type PaginatedResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data"`
    Meta    PageMeta    `json:"meta"`
}

type PageMeta struct {
    Page       int   `json:"page"`
    PerPage    int   `json:"per_page"`
    Total      int64 `json:"total"`
    TotalPages int   `json:"total_pages"`
}

func MessageResponse(message string) *Response {
    return &Response{Success: true, Message: message}
}
```

## Database Connection

```go
func initDB(dsn string) (*gorm.DB, error) {
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        return nil, err
    }

    sqlDB, err := db.DB()
    if err != nil {
        return nil, err
    }

    sqlDB.SetMaxOpenConns(25)
    sqlDB.SetMaxIdleConns(25)
    sqlDB.SetConnMaxLifetime(5 * time.Minute)

    return db, nil
}
```

## Testing

### Handler Tests with Mocks

```go
package handler_test

import (
    "context"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "myproject/internal/handler"
    "myproject/internal/model"
    "myproject/internal/service"
    "myproject/internal/validator"

    "github.com/labstack/echo/v4"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

// Mock service
type MockUserService struct {
    mock.Mock
}

func (m *MockUserService) Create(
    ctx context.Context, req *model.CreateUserRequest,
) (*model.User, error) {
    args := m.Called(ctx, req)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserService) GetByID(
    ctx context.Context, id uint,
) (*model.User, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*model.User), args.Error(1)
}

// Helper
func setupEcho() *echo.Echo {
    e := echo.New()
    e.Validator = validator.NewCustomValidator()
    return e
}

func TestCreateUser(t *testing.T) {
    e := setupEcho()
    mockService := new(MockUserService)
    h := handler.NewUserHandler(mockService)

    t.Run("successful creation", func(t *testing.T) {
        expectedUser := &model.User{
            ID: 1, Email: "test@example.com",
            FirstName: "Test", LastName: "User",
        }

        mockService.On("Create", mock.Anything, mock.Anything).
            Return(expectedUser, nil).Once()

        body := `{"email":"test@example.com","password":"password123","first_name":"Test","last_name":"User"}`
        req := httptest.NewRequest(http.MethodPost, "/users",
            strings.NewReader(body))
        req.Header.Set(echo.HeaderContentType,
            echo.MIMEApplicationJSON)
        rec := httptest.NewRecorder()
        c := e.NewContext(req, rec)

        err := h.Create(c)

        assert.NoError(t, err)
        assert.Equal(t, http.StatusCreated, rec.Code)
        mockService.AssertExpectations(t)
    })

    t.Run("duplicate email", func(t *testing.T) {
        mockService.On("Create", mock.Anything, mock.Anything).
            Return(nil, service.ErrUserAlreadyExists).Once()

        body := `{"email":"existing@example.com","password":"password123","first_name":"Test","last_name":"User"}`
        req := httptest.NewRequest(http.MethodPost, "/users",
            strings.NewReader(body))
        req.Header.Set(echo.HeaderContentType,
            echo.MIMEApplicationJSON)
        rec := httptest.NewRecorder()
        c := e.NewContext(req, rec)

        err := h.Create(c)

        assert.NoError(t, err)
        assert.Equal(t, http.StatusConflict, rec.Code)
        mockService.AssertExpectations(t)
    })
}

func TestGetUser(t *testing.T) {
    e := setupEcho()
    mockService := new(MockUserService)
    h := handler.NewUserHandler(mockService)

    t.Run("user found", func(t *testing.T) {
        expectedUser := &model.User{
            ID: 1, Email: "test@example.com",
            FirstName: "Test", LastName: "User",
        }

        mockService.On("GetByID", mock.Anything, uint(1)).
            Return(expectedUser, nil).Once()

        req := httptest.NewRequest(http.MethodGet, "/users/1", nil)
        rec := httptest.NewRecorder()
        c := e.NewContext(req, rec)
        c.SetParamNames("id")
        c.SetParamValues("1")

        err := h.GetByID(c)

        assert.NoError(t, err)
        assert.Equal(t, http.StatusOK, rec.Code)
        mockService.AssertExpectations(t)
    })

    t.Run("user not found", func(t *testing.T) {
        mockService.On("GetByID", mock.Anything, uint(999)).
            Return(nil, service.ErrUserNotFound).Once()

        req := httptest.NewRequest(http.MethodGet, "/users/999", nil)
        rec := httptest.NewRecorder()
        c := e.NewContext(req, rec)
        c.SetParamNames("id")
        c.SetParamValues("999")

        err := h.GetByID(c)

        assert.NoError(t, err)
        assert.Equal(t, http.StatusNotFound, rec.Code)
        mockService.AssertExpectations(t)
    })
}
```

### Testing Guidelines

- Use `httptest.NewRequest` and `httptest.NewRecorder` for handler tests
- Create Echo context with `e.NewContext(req, rec)`
- Set path params via `c.SetParamNames()` and `c.SetParamValues()`
- Mock services with `testify/mock`; assert expectations after each test
- Always register the custom validator in test setup
- Test both success and error paths (not found, conflict, bad request)

## Dependencies

```go
// go.mod
module myproject

go 1.21

require (
    github.com/labstack/echo/v4 v4.11.3
    github.com/go-playground/validator/v10 v10.16.0
    github.com/golang-jwt/jwt/v5 v5.0.0
    github.com/joho/godotenv v1.5.1
    golang.org/x/crypto v0.14.0
    gorm.io/driver/postgres v1.5.4
    gorm.io/gorm v1.25.5
)

require (
    github.com/stretchr/testify v1.8.4 // testing
)
```

## Custom Validation Rules

```go
func formatValidationErrors(err error) map[string]interface{} {
    errors := make(map[string]string)

    for _, err := range err.(validator.ValidationErrors) {
        field := err.Field()
        switch err.Tag() {
        case "required":
            errors[field] = field + " is required"
        case "email":
            errors[field] = field + " must be a valid email"
        case "min":
            errors[field] = field + " must be at least " +
                err.Param() + " characters"
        case "max":
            errors[field] = field + " must be at most " +
                err.Param() + " characters"
        default:
            errors[field] = field + " is invalid"
        }
    }

    return map[string]interface{}{
        "error":  "Validation failed",
        "fields": errors,
    }
}
```
