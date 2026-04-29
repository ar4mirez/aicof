# Python Patterns Reference

> Copy-paste-ready patterns complementing SKILL.md.

## Async Patterns

### Bounded concurrent gather

```python
import asyncio, aiohttp
from collections.abc import Sequence

async def fetch_all(urls: Sequence[str]) -> list[dict[str, object]]:
    sem = asyncio.Semaphore(10)
    async def _one(url: str) -> dict[str, object]:
        async with sem, aiohttp.ClientSession() as s:
            async with s.get(url, timeout=aiohttp.ClientTimeout(total=30)) as r:
                r.raise_for_status()
                return await r.json()
    return await asyncio.gather(*(_one(u) for u in urls))
```

### Async generator (stream rows)

```python
from collections.abc import AsyncIterator

async def stream_rows(query: str) -> AsyncIterator[dict[str, object]]:
    async with pool.acquire() as conn:
        async for row in conn.cursor(query):
            yield dict(row)
```

## Dataclass and Pydantic Models

### Immutable dataclass

```python
from dataclasses import dataclass, field
from uuid import UUID, uuid4

@dataclass(frozen=True, slots=True)
class Order:
    customer_id: UUID
    items: tuple[str, ...]
    total_cents: int
    id: UUID = field(default_factory=uuid4)
    def __post_init__(self) -> None:
        if self.total_cents < 0:
            raise ValueError("total_cents must be non-negative")
```

### Pydantic v2 validated model

```python
from pydantic import BaseModel, Field, field_validator

class CreateUserRequest(BaseModel):
    model_config = {"str_strip_whitespace": True, "frozen": True}
    name: str = Field(min_length=1, max_length=200)
    email: str = Field(pattern=r"^[\w.+-]+@[\w-]+\.[\w.]+$")
    age: int = Field(ge=13, le=150)
    @field_validator("name")
    @classmethod
    def no_blank(cls, v: str) -> str:
        if not v.strip(): raise ValueError("must contain non-whitespace")
        return v
```

## Context Manager Patterns

### Temporary state (revert on exit)

```python
from contextlib import contextmanager
from collections.abc import Generator
import os

@contextmanager
def override_env(key: str, value: str) -> Generator[None, None, None]:
    prev = os.environ.get(key)
    os.environ[key] = value
    try:
        yield
    finally:
        if prev is None: os.environ.pop(key, None)
        else: os.environ[key] = prev
```

## Decorator Patterns

### Retry with exponential backoff

```python
import asyncio, functools
from collections.abc import Awaitable, Callable
from typing import ParamSpec, TypeVar

P, T = ParamSpec("P"), TypeVar("T")

def retry(
    attempts: int = 3, delay: float = 1.0,
    on: tuple[type[Exception], ...] = (Exception,),
) -> Callable[[Callable[P, Awaitable[T]]], Callable[P, Awaitable[T]]]:
    def dec(fn: Callable[P, Awaitable[T]]) -> Callable[P, Awaitable[T]]:
        @functools.wraps(fn)
        async def wrapper(*a: P.args, **kw: P.kwargs) -> T:
            last: Exception | None = None
            for i in range(1, attempts + 1):
                try: return await fn(*a, **kw)
                except on as e:
                    last = e
                    if i < attempts: await asyncio.sleep(delay * 2 ** (i - 1))
            raise last  # type: ignore[misc]
        return wrapper
    return dec
```

## Type Hint Patterns

### Generic class (TypeVar + Generic)

```python
from typing import Generic, TypeVar
from collections.abc import Sequence
from uuid import UUID

E = TypeVar("E")

class Repository(Generic[E]):
    def get(self, id: UUID) -> E | None: ...
    def list(self, *, limit: int = 50, offset: int = 0) -> Sequence[E]: ...
    def save(self, entity: E) -> E: ...
    def delete(self, id: UUID) -> None: ...
```

### Protocol (structural subtyping)

```python
from typing import Protocol, runtime_checkable

@runtime_checkable
class Renderable(Protocol):
    def render_html(self) -> str: ...

def render_page(items: list[Renderable]) -> str:
    return "\n".join(i.render_html() for i in items)
```

### TypeVar with bound

```python
from typing import Protocol, TypeVar
from collections.abc import Sequence

class SupportsLT(Protocol):
    def __lt__(self, other: object) -> bool: ...

Cmp = TypeVar("Cmp", bound=SupportsLT)

def min_value(items: Sequence[Cmp]) -> Cmp:
    if not items: raise ValueError("non-empty sequence required")
    return min(items)
```
