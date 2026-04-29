# Advanced TypeScript Patterns

> Complements SKILL.md with advanced type-level patterns, runtime validation,
> async orchestration, module organization, and React + TypeScript idioms.

---

## Discriminated Unions

Use a shared literal property to let TypeScript narrow types automatically.

```typescript
type ApiResponse<T> =
  | { status: "success"; data: T; timestamp: number }
  | { status: "error"; error: string; retryable: boolean }
  | { status: "loading" };

function handleResponse<T>(res: ApiResponse<T>): T | null {
  switch (res.status) {
    case "success":
      return res.data; // TS knows `data` and `timestamp` exist
    case "error":
      if (res.retryable) console.warn("Will retry:", res.error);
      return null;
    case "loading":
      return null;
  }
}
```

Exhaustiveness check -- ensure every variant is handled:

```typescript
function assertNever(value: never): never {
  throw new Error(`Unhandled discriminant: ${JSON.stringify(value)}`);
}

// Add `default: return assertNever(res);` to the switch above.
// If a new variant is added, the compiler will flag the missing case.
```

## Template Literal Types

Build string types from other types at the type level.

```typescript
type EventName = "click" | "focus" | "blur";
type HandlerName = `on${Capitalize<EventName>}`; // "onClick" | "onFocus" | "onBlur"

type CssUnit = "px" | "rem" | "em" | "%";
type CssValue = `${number}${CssUnit}`; // "16px", "1.5rem", etc.

// Route parameter extraction
type ExtractParams<T extends string> =
  T extends `${string}:${infer Param}/${infer Rest}`
    ? Param | ExtractParams<Rest>
    : T extends `${string}:${infer Param}`
      ? Param
      : never;

type Params = ExtractParams<"/users/:userId/posts/:postId">; // "userId" | "postId"
```

## Conditional & Mapped Types

```typescript
// Make selected properties optional
type PartialBy<T, K extends keyof T> = Omit<T, K> & Partial<Pick<T, K>>;

// Deep readonly (recursive)
type DeepReadonly<T> = {
  readonly [K in keyof T]: T[K] extends object ? DeepReadonly<T[K]> : T[K];
};

// Strip null/undefined from all properties
type NonNullableFields<T> = {
  [K in keyof T]: NonNullable<T[K]>;
};

// Extract function return types that are promises
type UnwrapPromise<T> = T extends Promise<infer U> ? U : T;
```

## Zod Advanced Validation

Beyond basic schemas -- transforms, refinements, and discriminated unions at runtime.

```typescript
import { z } from "zod";

// Transform: parse then reshape
const MoneySchema = z
  .string()
  .regex(/^\d+\.\d{2}$/, "Must be formatted as 0.00")
  .transform((val) => Math.round(parseFloat(val) * 100)); // store as cents

type Money = z.output<typeof MoneySchema>; // number (cents)

// Discriminated union at runtime (mirrors the type-level pattern)
const ShapeSchema = z.discriminatedUnion("kind", [
  z.object({ kind: z.literal("circle"), radius: z.number().positive() }),
  z.object({ kind: z.literal("rect"), width: z.number(), height: z.number() }),
]);

// Refinement with custom error paths
const DateRangeSchema = z
  .object({
    start: z.coerce.date(),
    end: z.coerce.date(),
  })
  .refine((data) => data.end > data.start, {
    message: "End date must be after start date",
    path: ["end"],
  });

// Compose schemas for API layers
const PaginationSchema = z.object({
  page: z.coerce.number().int().min(1).default(1),
  limit: z.coerce.number().int().min(1).max(100).default(20),
});

const UserListQuerySchema = PaginationSchema.extend({
  search: z.string().optional(),
  role: z.enum(["admin", "user", "viewer"]).optional(),
});
```

## Async Patterns

### Promise.allSettled with typed results

```typescript
async function fetchMultiple<T>(
  tasks: Array<{ name: string; fn: () => Promise<T> }>
): Promise<{ succeeded: T[]; failed: Array<{ name: string; reason: unknown }> }> {
  const results = await Promise.allSettled(tasks.map((t) => t.fn()));

  const succeeded: T[] = [];
  const failed: Array<{ name: string; reason: unknown }> = [];

  results.forEach((result, i) => {
    if (result.status === "fulfilled") {
      succeeded.push(result.value);
    } else {
      failed.push({ name: tasks[i]!.name, reason: result.reason });
    }
  });

  return { succeeded, failed };
}
```

### AbortController with cascading cancellation

```typescript
function withTimeout<T>(
  fn: (signal: AbortSignal) => Promise<T>,
  ms: number
): Promise<T> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), ms);

  return fn(controller.signal).finally(() => clearTimeout(timeout));
}

// Usage: parent signal propagates to child operations
async function fetchUserWithPosts(
  userId: string,
  parentSignal?: AbortSignal
): Promise<{ user: User; posts: Post[] }> {
  const controller = new AbortController();
  parentSignal?.addEventListener("abort", () => controller.abort());

  const [user, posts] = await Promise.all([
    fetchJson<User>(`/api/users/${userId}`, controller.signal),
    fetchJson<Post[]>(`/api/users/${userId}/posts`, controller.signal),
  ]);

  return { user, posts };
}
```

### Async iterators for streaming data

```typescript
async function* paginateApi<T>(
  baseUrl: string,
  pageSize: number = 50
): AsyncGenerator<T[], void, undefined> {
  let page = 1;
  let hasMore = true;

  while (hasMore) {
    const url = `${baseUrl}?page=${page}&limit=${pageSize}`;
    const data: T[] = await fetchJson(url);

    if (data.length === 0) {
      hasMore = false;
    } else {
      yield data;
      page++;
      hasMore = data.length === pageSize;
    }
  }
}

// Consume: processes one page at a time, constant memory
for await (const batch of paginateApi<User>("/api/users")) {
  await processBatch(batch);
}
```

## Module Patterns

### Path aliases (tsconfig paths)

```json
{
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"],
      "@domain/*": ["src/domain/*"],
      "@services/*": ["src/services/*"]
    }
  }
}
```

```typescript
// Clean imports instead of fragile relative paths
import { UserService } from "@services/user.service";
import { validateEmail } from "@domain/user";
```

### Barrel exports -- safe boundaries only

```typescript
// src/domain/index.ts -- package boundary, OK
export { createUser, validateUser } from "./user";
export { createOrder } from "./order";
export type { User, Order } from "./types";

// AVOID barrel re-exports within feature folders (causes circular deps).
// Import directly: import { helper } from "./utils/helper";
```

### Type-only imports (enforced by verbatimModuleSyntax)

```typescript
import type { User } from "./types";         // erased at runtime
import { UserSchema } from "./schemas";        // kept at runtime
import { type Order, OrderSchema } from "./schemas"; // mixed: Order erased, schema kept
```

## React + TypeScript Patterns

### Generic components

```typescript
interface ListProps<T> {
  items: T[];
  renderItem: (item: T, index: number) => React.ReactNode;
  keyExtractor: (item: T) => string;
}

function List<T>({ items, renderItem, keyExtractor }: ListProps<T>): React.ReactElement {
  return (
    <ul>
      {items.map((item, i) => (
        <li key={keyExtractor(item)}>{renderItem(item, i)}</li>
      ))}
    </ul>
  );
}

// Usage -- T is inferred as User
<List items={users} renderItem={(u) => u.name} keyExtractor={(u) => u.id} />
```

### forwardRef with generics

```typescript
import { forwardRef, type Ref } from "react";

interface InputProps<T extends string | number> {
  value: T;
  onChange: (value: T) => void;
  label: string;
}

// forwardRef loses generics -- use a wrapper pattern
function InputInner<T extends string | number>(
  props: InputProps<T>,
  ref: Ref<HTMLInputElement>
): React.ReactElement {
  return (
    <label>
      {props.label}
      <input
        ref={ref}
        value={props.value}
        onChange={(e) => props.onChange(e.target.value as T)}
      />
    </label>
  );
}

export const Input = forwardRef(InputInner) as <T extends string | number>(
  props: InputProps<T> & { ref?: Ref<HTMLInputElement> }
) => React.ReactElement;
```

### Custom hooks with generics

```typescript
import { useState, useCallback } from "react";

interface UseAsyncState<T> {
  data: T | null;
  error: Error | null;
  loading: boolean;
  execute: (...args: unknown[]) => Promise<T | null>;
}

function useAsync<T>(asyncFn: (...args: unknown[]) => Promise<T>): UseAsyncState<T> {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState(false);

  const execute = useCallback(
    async (...args: unknown[]): Promise<T | null> => {
      setLoading(true);
      setError(null);
      try {
        const result = await asyncFn(...args);
        setData(result);
        return result;
      } catch (err) {
        const wrapped = err instanceof Error ? err : new Error(String(err));
        setError(wrapped);
        return null;
      } finally {
        setLoading(false);
      }
    },
    [asyncFn]
  );

  return { data, error, loading, execute };
}
```

### Polymorphic `as` prop

```typescript
type PolymorphicProps<E extends React.ElementType, P = object> = P &
  Omit<React.ComponentPropsWithoutRef<E>, keyof P> & {
    as?: E;
  };

function Button<E extends React.ElementType = "button">({
  as,
  children,
  ...rest
}: PolymorphicProps<E, { children: React.ReactNode }>): React.ReactElement {
  const Component = as ?? "button";
  return <Component {...rest}>{children}</Component>;
}

// Renders <a> with full anchor props, type-safe
<Button as="a" href="/home">Go Home</Button>
```
