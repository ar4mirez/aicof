# FastAPI Patterns Reference

## Contents

- [Database Setup](#database-setup)
- [SQLAlchemy Models](#sqlalchemy-models)
- [Repository Pattern](#repository-pattern)
- [Service Layer](#service-layer)
- [Auth Endpoints](#auth-endpoints)
- [Security (JWT & Passwords)](#security-jwt--passwords)
- [Advanced Dependencies](#advanced-dependencies)
- [Middleware](#middleware)
- [Background Tasks](#background-tasks)
- [WebSocket](#websocket)
- [Testing](#testing)
- [Dependencies (pyproject.toml)](#dependencies-pyprojecttoml)

## Database Setup

### Async SQLAlchemy Engine

```python
# app/database.py
from sqlalchemy.ext.asyncio import (
    AsyncSession,
    async_sessionmaker,
    create_async_engine,
)
from app.config import settings

engine = create_async_engine(
    settings.DATABASE_URL,
    echo=settings.DEBUG,
    pool_pre_ping=True,
    pool_size=5,
    max_overflow=10,
)

AsyncSessionLocal = async_sessionmaker(
    engine,
    class_=AsyncSession,
    expire_on_commit=False,
    autocommit=False,
    autoflush=False,
)


async def get_db() -> AsyncSession:
    """Dependency for getting async database session."""
    async with AsyncSessionLocal() as session:
        try:
            yield session
            await session.commit()
        except Exception:
            await session.rollback()
            raise
        finally:
            await session.close()
```

### Base Model with Mixins

```python
# app/models/base.py
from datetime import datetime
from uuid import uuid4

from sqlalchemy import DateTime, func
from sqlalchemy.dialects.postgresql import UUID
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column


class Base(DeclarativeBase):
    pass


class TimestampMixin:
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True),
        server_default=func.now(),
        nullable=False,
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True),
        server_default=func.now(),
        onupdate=func.now(),
        nullable=False,
    )


class UUIDMixin:
    id: Mapped[str] = mapped_column(
        UUID(as_uuid=True),
        primary_key=True,
        default=uuid4,
    )
```

## SQLAlchemy Models

```python
# app/models/user.py
from sqlalchemy import Boolean, String, Text, ForeignKey
from sqlalchemy.orm import Mapped, mapped_column, relationship

from app.models.base import Base, TimestampMixin, UUIDMixin


class User(Base, UUIDMixin, TimestampMixin):
    __tablename__ = "users"

    email: Mapped[str] = mapped_column(String(255), unique=True, index=True)
    hashed_password: Mapped[str] = mapped_column(String(255))
    first_name: Mapped[str | None] = mapped_column(String(100))
    last_name: Mapped[str | None] = mapped_column(String(100))
    is_active: Mapped[bool] = mapped_column(Boolean, default=True)
    is_superuser: Mapped[bool] = mapped_column(Boolean, default=False)

    profile: Mapped["Profile"] = relationship(back_populates="user", uselist=False)
    posts: Mapped[list["Post"]] = relationship(back_populates="author")

    @property
    def full_name(self) -> str:
        if self.first_name and self.last_name:
            return f"{self.first_name} {self.last_name}"
        return self.email


class Profile(Base, UUIDMixin, TimestampMixin):
    __tablename__ = "profiles"

    user_id: Mapped[str] = mapped_column(
        ForeignKey("users.id", ondelete="CASCADE"),
        unique=True,
    )
    bio: Mapped[str | None] = mapped_column(Text)
    avatar_url: Mapped[str | None] = mapped_column(String(500))

    user: Mapped["User"] = relationship(back_populates="profile")


class Post(Base, UUIDMixin, TimestampMixin):
    __tablename__ = "posts"

    title: Mapped[str] = mapped_column(String(255))
    content: Mapped[str] = mapped_column(Text)
    is_published: Mapped[bool] = mapped_column(Boolean, default=False)
    author_id: Mapped[str] = mapped_column(
        ForeignKey("users.id", ondelete="CASCADE"),
    )

    author: Mapped["User"] = relationship(back_populates="posts")
```

## Repository Pattern

### Generic Base Repository

```python
# app/repositories/base.py
from typing import Any, Generic, TypeVar
from uuid import UUID

from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.base import Base

ModelType = TypeVar("ModelType", bound=Base)


class BaseRepository(Generic[ModelType]):
    def __init__(self, model: type[ModelType], session: AsyncSession):
        self.model = model
        self.session = session

    async def get(self, id: UUID) -> ModelType | None:
        result = await self.session.execute(
            select(self.model).where(self.model.id == id)
        )
        return result.scalar_one_or_none()

    async def get_multi(
        self, *, skip: int = 0, limit: int = 100,
    ) -> list[ModelType]:
        result = await self.session.execute(
            select(self.model).offset(skip).limit(limit)
        )
        return list(result.scalars().all())

    async def create(self, obj_in: dict[str, Any]) -> ModelType:
        db_obj = self.model(**obj_in)
        self.session.add(db_obj)
        await self.session.flush()
        await self.session.refresh(db_obj)
        return db_obj

    async def update(
        self, db_obj: ModelType, obj_in: dict[str, Any],
    ) -> ModelType:
        for field, value in obj_in.items():
            if value is not None:
                setattr(db_obj, field, value)
        await self.session.flush()
        await self.session.refresh(db_obj)
        return db_obj

    async def delete(self, id: UUID) -> bool:
        obj = await self.get(id)
        if obj:
            await self.session.delete(obj)
            await self.session.flush()
            return True
        return False

    async def count(self) -> int:
        result = await self.session.execute(
            select(func.count()).select_from(self.model)
        )
        return result.scalar_one()
```

### Specialized Repository

```python
# app/repositories/user.py
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from app.models.user import User
from app.repositories.base import BaseRepository


class UserRepository(BaseRepository[User]):
    def __init__(self, session: AsyncSession):
        super().__init__(User, session)

    async def get_by_email(self, email: str) -> User | None:
        result = await self.session.execute(
            select(User).where(User.email == email)
        )
        return result.scalar_one_or_none()

    async def get_with_profile(self, id: str) -> User | None:
        result = await self.session.execute(
            select(User)
            .options(selectinload(User.profile))
            .where(User.id == id)
        )
        return result.scalar_one_or_none()

    async def get_active_users(
        self, *, skip: int = 0, limit: int = 100,
    ) -> list[User]:
        result = await self.session.execute(
            select(User)
            .where(User.is_active == True)
            .offset(skip)
            .limit(limit)
        )
        return list(result.scalars().all())
```

## Service Layer

```python
# app/services/user.py
from uuid import UUID

from sqlalchemy.ext.asyncio import AsyncSession

from app.core.security import get_password_hash, verify_password
from app.core.exceptions import (
    NotFoundException, ConflictException, UnauthorizedException,
)
from app.models.user import User
from app.repositories.user import UserRepository
from app.schemas.user import UserCreate, UserUpdate


class UserService:
    def __init__(self, session: AsyncSession):
        self.session = session
        self.repository = UserRepository(session)

    async def get(self, user_id: UUID) -> User:
        user = await self.repository.get(user_id)
        if not user:
            raise NotFoundException(f"User {user_id} not found")
        return user

    async def get_by_email(self, email: str) -> User | None:
        return await self.repository.get_by_email(email)

    async def get_multi(
        self, *, skip: int = 0, limit: int = 100,
    ) -> tuple[list[User], int]:
        users = await self.repository.get_multi(skip=skip, limit=limit)
        total = await self.repository.count()
        return users, total

    async def create(self, user_in: UserCreate) -> User:
        existing = await self.repository.get_by_email(user_in.email)
        if existing:
            raise ConflictException("Email already registered")

        user_data = user_in.model_dump(exclude={"password"})
        user_data["hashed_password"] = get_password_hash(user_in.password)
        return await self.repository.create(user_data)

    async def update(self, user_id: UUID, user_in: UserUpdate) -> User:
        user = await self.get(user_id)
        update_data = user_in.model_dump(exclude_unset=True, exclude={"password"})
        if user_in.password:
            update_data["hashed_password"] = get_password_hash(user_in.password)
        return await self.repository.update(user, update_data)

    async def delete(self, user_id: UUID) -> bool:
        await self.get(user_id)  # Verify exists
        return await self.repository.delete(user_id)

    async def authenticate(self, email: str, password: str) -> User:
        user = await self.repository.get_by_email(email)
        if not user:
            raise UnauthorizedException("Invalid credentials")
        if not verify_password(password, user.hashed_password):
            raise UnauthorizedException("Invalid credentials")
        if not user.is_active:
            raise UnauthorizedException("User is inactive")
        return user
```

## Auth Endpoints

```python
# app/api/v1/endpoints/auth.py
from fastapi import APIRouter, Depends
from fastapi.security import OAuth2PasswordRequestForm
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.security import create_access_token, create_refresh_token
from app.database import get_db
from app.schemas.user import Token, UserCreate, UserResponse
from app.services.user import UserService

router = APIRouter()


@router.post("/login", response_model=Token)
async def login(
    form_data: OAuth2PasswordRequestForm = Depends(),
    db: AsyncSession = Depends(get_db),
):
    """OAuth2 compatible token login."""
    service = UserService(db)
    user = await service.authenticate(form_data.username, form_data.password)
    return Token(
        access_token=create_access_token(str(user.id)),
        refresh_token=create_refresh_token(str(user.id)),
    )


@router.post("/register", response_model=UserResponse)
async def register(
    user_in: UserCreate,
    db: AsyncSession = Depends(get_db),
):
    """Register new user."""
    service = UserService(db)
    user = await service.create(user_in)
    return UserResponse.model_validate(user)


@router.post("/refresh", response_model=Token)
async def refresh_token(
    refresh_token: str,
    db: AsyncSession = Depends(get_db),
):
    """Refresh access token."""
    from app.core.security import verify_token

    payload = verify_token(refresh_token, token_type="refresh")
    return Token(
        access_token=create_access_token(payload.sub),
        refresh_token=create_refresh_token(payload.sub),
    )
```

## Security (JWT & Passwords)

```python
# app/core/security.py
from datetime import datetime, timedelta

from jose import JWTError, jwt
from passlib.context import CryptContext

from app.config import settings
from app.core.exceptions import UnauthorizedException
from app.schemas.user import TokenPayload

pwd_context = CryptContext(schemes=["bcrypt"], deprecated="auto")
ALGORITHM = "HS256"


def verify_password(plain_password: str, hashed_password: str) -> bool:
    return pwd_context.verify(plain_password, hashed_password)


def get_password_hash(password: str) -> str:
    return pwd_context.hash(password)


def create_access_token(subject: str) -> str:
    expire = datetime.utcnow() + timedelta(
        minutes=settings.ACCESS_TOKEN_EXPIRE_MINUTES
    )
    to_encode = {"sub": subject, "exp": expire, "type": "access"}
    return jwt.encode(to_encode, settings.SECRET_KEY, algorithm=ALGORITHM)


def create_refresh_token(subject: str) -> str:
    expire = datetime.utcnow() + timedelta(days=settings.REFRESH_TOKEN_EXPIRE_DAYS)
    to_encode = {"sub": subject, "exp": expire, "type": "refresh"}
    return jwt.encode(to_encode, settings.SECRET_KEY, algorithm=ALGORITHM)


def verify_token(token: str, token_type: str = "access") -> TokenPayload:
    try:
        payload = jwt.decode(token, settings.SECRET_KEY, algorithms=[ALGORITHM])
        token_data = TokenPayload(**payload)
        if token_data.type != token_type:
            raise UnauthorizedException("Invalid token type")
        return token_data
    except JWTError:
        raise UnauthorizedException("Invalid token")
```

### Token Schemas

```python
# In app/schemas/user.py
class Token(BaseModel):
    access_token: str
    refresh_token: str
    token_type: str = "bearer"


class TokenPayload(BaseModel):
    sub: str
    exp: datetime
    type: str
```

## Advanced Dependencies

### Optional Authentication

```python
# app/api/deps.py
async def get_optional_user(
    token: str | None = Depends(oauth2_scheme),
    db: AsyncSession = Depends(get_db),
) -> User | None:
    """Get current user if authenticated, None otherwise."""
    if not token:
        return None
    try:
        return await get_current_user(token, db)
    except HTTPException:
        return None
```

### Parameterized Dependencies

```python
from functools import partial


def require_permission(permission: str):
    """Create a dependency that checks for a specific permission."""
    async def check_permission(
        current_user: User = Depends(get_current_user),
    ) -> User:
        if permission not in current_user.permissions:
            raise HTTPException(status_code=403, detail="Insufficient permissions")
        return current_user
    return check_permission


# Usage in endpoint
@router.delete("/{item_id}")
async def delete_item(
    item_id: UUID,
    user: User = Depends(require_permission("items:delete")),
):
    ...
```

### Service Factory Dependency

```python
def get_user_service(db: AsyncSession = Depends(get_db)) -> UserService:
    """Dependency that provides a UserService instance."""
    return UserService(db)


@router.get("/{user_id}", response_model=UserResponse)
async def get_user(
    user_id: UUID,
    service: UserService = Depends(get_user_service),
    current_user: User = Depends(get_current_user),
):
    user = await service.get(user_id)
    return UserResponse.model_validate(user)
```

## Middleware

### Custom Middleware

```python
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.requests import Request
from starlette.responses import Response
import time
import logging

logger = logging.getLogger(__name__)


class RequestTimingMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request: Request, call_next) -> Response:
        start_time = time.perf_counter()
        response = await call_next(request)
        duration = time.perf_counter() - start_time
        response.headers["X-Process-Time"] = str(duration)
        logger.info(
            f"{request.method} {request.url.path} "
            f"completed in {duration:.3f}s "
            f"status={response.status_code}"
        )
        return response


# Register in main.py
app.add_middleware(RequestTimingMiddleware)
```

### Rate Limiting Middleware

```python
from collections import defaultdict
from datetime import datetime


class RateLimitMiddleware(BaseHTTPMiddleware):
    def __init__(self, app, max_requests: int = 100, window_seconds: int = 60):
        super().__init__(app)
        self.max_requests = max_requests
        self.window_seconds = window_seconds
        self.requests: dict[str, list[datetime]] = defaultdict(list)

    async def dispatch(self, request: Request, call_next) -> Response:
        client_ip = request.client.host
        now = datetime.utcnow()

        # Clean old requests
        self.requests[client_ip] = [
            t for t in self.requests[client_ip]
            if (now - t).total_seconds() < self.window_seconds
        ]

        if len(self.requests[client_ip]) >= self.max_requests:
            return Response(
                content="Rate limit exceeded",
                status_code=429,
            )

        self.requests[client_ip].append(now)
        return await call_next(request)
```

## Background Tasks

```python
# app/tasks/email.py
from fastapi import BackgroundTasks
from app.core.email import send_email


def send_welcome_email(
    background_tasks: BackgroundTasks,
    email: str,
    name: str,
):
    """Add welcome email to background tasks."""
    background_tasks.add_task(
        send_email,
        to=email,
        subject="Welcome!",
        template="welcome.html",
        context={"name": name},
    )


# Using in endpoint
@router.post("/register")
async def register(
    user_in: UserCreate,
    background_tasks: BackgroundTasks,
    db: AsyncSession = Depends(get_db),
):
    service = UserService(db)
    user = await service.create(user_in)

    send_welcome_email(
        background_tasks,
        email=user.email,
        name=user.full_name,
    )

    return UserResponse.model_validate(user)
```

**When to use BackgroundTasks vs Celery/ARQ:**
- `BackgroundTasks`: Simple, in-process tasks (emails, logging, cache warming)
- Celery/ARQ: Long-running tasks, need retries, need task queue visibility

## WebSocket

```python
from fastapi import WebSocket, WebSocketDisconnect


class ConnectionManager:
    def __init__(self):
        self.active_connections: list[WebSocket] = []

    async def connect(self, websocket: WebSocket):
        await websocket.accept()
        self.active_connections.append(websocket)

    def disconnect(self, websocket: WebSocket):
        self.active_connections.remove(websocket)

    async def broadcast(self, message: str):
        for connection in self.active_connections:
            await connection.send_text(message)


manager = ConnectionManager()


@app.websocket("/ws/{client_id}")
async def websocket_endpoint(websocket: WebSocket, client_id: str):
    await manager.connect(websocket)
    try:
        while True:
            data = await websocket.receive_text()
            await manager.broadcast(f"Client {client_id}: {data}")
    except WebSocketDisconnect:
        manager.disconnect(websocket)
        await manager.broadcast(f"Client {client_id} left")
```

## Testing

### Fixtures (conftest.py)

```python
# tests/conftest.py
import asyncio
from typing import AsyncGenerator

import pytest
import pytest_asyncio
from httpx import AsyncClient
from sqlalchemy.ext.asyncio import AsyncSession, create_async_engine
from sqlalchemy.orm import sessionmaker

from app.config import settings
from app.database import get_db
from app.main import app
from app.models.base import Base

TEST_DATABASE_URL = settings.DATABASE_URL.replace("mydb", "mydb_test")
engine = create_async_engine(TEST_DATABASE_URL, echo=False)
TestSessionLocal = sessionmaker(engine, class_=AsyncSession, expire_on_commit=False)


@pytest.fixture(scope="session")
def event_loop():
    loop = asyncio.get_event_loop_policy().new_event_loop()
    yield loop
    loop.close()


@pytest_asyncio.fixture(scope="function")
async def db_session() -> AsyncGenerator[AsyncSession, None]:
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    async with TestSessionLocal() as session:
        yield session
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.drop_all)


@pytest_asyncio.fixture(scope="function")
async def client(db_session: AsyncSession) -> AsyncGenerator[AsyncClient, None]:
    async def override_get_db():
        yield db_session

    app.dependency_overrides[get_db] = override_get_db
    async with AsyncClient(app=app, base_url="http://test") as ac:
        yield ac
    app.dependency_overrides.clear()


@pytest_asyncio.fixture
async def authenticated_client(
    client: AsyncClient,
    db_session: AsyncSession,
) -> AsyncClient:
    from app.services.user import UserService
    from app.schemas.user import UserCreate
    from app.core.security import create_access_token

    service = UserService(db_session)
    user = await service.create(
        UserCreate(email="test@example.com", password="testpassword123")
    )
    await db_session.commit()

    token = create_access_token(str(user.id))
    client.headers["Authorization"] = f"Bearer {token}"
    return client
```

### Test Examples

```python
# tests/test_users.py
import pytest
from httpx import AsyncClient


@pytest.mark.asyncio
async def test_register_user(client: AsyncClient):
    response = await client.post(
        "/api/v1/auth/register",
        json={"email": "newuser@example.com", "password": "password123"},
    )
    assert response.status_code == 200
    data = response.json()
    assert data["email"] == "newuser@example.com"
    assert "id" in data


@pytest.mark.asyncio
async def test_login(client: AsyncClient, db_session):
    from app.services.user import UserService
    from app.schemas.user import UserCreate

    service = UserService(db_session)
    await service.create(
        UserCreate(email="login@example.com", password="password123")
    )
    await db_session.commit()

    response = await client.post(
        "/api/v1/auth/login",
        data={"username": "login@example.com", "password": "password123"},
    )
    assert response.status_code == 200
    data = response.json()
    assert "access_token" in data
    assert "refresh_token" in data


@pytest.mark.asyncio
async def test_get_current_user(authenticated_client: AsyncClient):
    response = await authenticated_client.get("/api/v1/users/me")
    assert response.status_code == 200
    data = response.json()
    assert data["email"] == "test@example.com"


@pytest.mark.asyncio
async def test_unauthorized_access(client: AsyncClient):
    response = await client.get("/api/v1/users/me")
    assert response.status_code == 401
```

### Testing Best Practices

- Use `httpx.AsyncClient` (not `TestClient` for async)
- Override dependencies with `app.dependency_overrides`
- Use a separate test database (never test against production)
- Create/drop tables per test function for isolation
- Use factory functions or `pytest-factoryboy` for test data
- Test both happy paths and error cases
- Test authentication and authorization separately

## Dependencies (pyproject.toml)

```toml
[project]
name = "myproject"
version = "1.0.0"
requires-python = ">=3.10"
dependencies = [
    "fastapi>=0.109.0",
    "uvicorn[standard]>=0.27.0",
    "pydantic>=2.5.0",
    "pydantic-settings>=2.1.0",
    "sqlalchemy[asyncio]>=2.0.0",
    "asyncpg>=0.29.0",
    "alembic>=1.13.0",
    "python-jose[cryptography]>=3.3.0",
    "passlib[bcrypt]>=1.7.4",
    "python-multipart>=0.0.6",
    "httpx>=0.26.0",
    "redis>=5.0.0",
]

[project.optional-dependencies]
dev = [
    "pytest>=7.4.0",
    "pytest-asyncio>=0.23.0",
    "pytest-cov>=4.1.0",
    "ruff>=0.1.0",
    "mypy>=1.8.0",
    "pre-commit>=3.6.0",
]
```
