# Express.js Patterns Reference

## Contents

- [Controller Pattern](#controller-pattern)
- [Service Layer Pattern](#service-layer-pattern)
- [Authentication Middleware](#authentication-middleware)
- [Role-Based Access Control](#role-based-access-control)
- [Validation Schemas](#validation-schemas)
- [Rate Limiting](#rate-limiting)
- [Database Integration (Prisma)](#database-integration-prisma)
- [Not Found Handler](#not-found-handler)
- [Integration Testing](#integration-testing)
- [Configuration Files](#configuration-files)

## Controller Pattern

Controllers are thin handlers that delegate to services and forward errors to middleware.

```typescript
// src/controllers/user.controller.ts
import { Request, Response, NextFunction } from 'express';
import { UserService } from '../services/user.service';
import { CreateUserDTO, UpdateUserDTO } from '../types';
import { AppError } from '../utils/errors';

export class UserController {
  private userService: UserService;

  constructor() {
    this.userService = new UserService();
  }

  getAll = async (req: Request, res: Response, next: NextFunction) => {
    try {
      const { page = 1, limit = 10, sort = 'createdAt' } = req.query;

      const result = await this.userService.findAll({
        page: Number(page),
        limit: Number(limit),
        sort: String(sort),
      });

      res.json({
        data: result.users,
        meta: {
          page: result.page,
          limit: result.limit,
          total: result.total,
          totalPages: result.totalPages,
        },
      });
    } catch (error) {
      next(error);
    }
  };

  getById = async (req: Request, res: Response, next: NextFunction) => {
    try {
      const { id } = req.params;
      const user = await this.userService.findById(id);

      if (!user) {
        throw new AppError('User not found', 404);
      }

      res.json({ data: user });
    } catch (error) {
      next(error);
    }
  };

  create = async (req: Request, res: Response, next: NextFunction) => {
    try {
      const data: CreateUserDTO = req.body;
      const user = await this.userService.create(data);

      res.status(201).json({ data: user });
    } catch (error) {
      next(error);
    }
  };

  update = async (req: Request, res: Response, next: NextFunction) => {
    try {
      const { id } = req.params;
      const data: UpdateUserDTO = req.body;
      const user = await this.userService.update(id, data);

      if (!user) {
        throw new AppError('User not found', 404);
      }

      res.json({ data: user });
    } catch (error) {
      next(error);
    }
  };

  delete = async (req: Request, res: Response, next: NextFunction) => {
    try {
      const { id } = req.params;
      await this.userService.delete(id);

      res.status(204).send();
    } catch (error) {
      next(error);
    }
  };
}
```

### Key Points

- Use arrow functions for methods to preserve `this` context
- Always wrap async handlers in try/catch and call `next(error)`
- Return appropriate HTTP status codes: 200 (OK), 201 (Created), 204 (No Content), 404 (Not Found)
- Use consistent response envelope: `{ data: ... }` and `{ data: ..., meta: ... }`

## Service Layer Pattern

Services contain business logic and orchestrate between repositories.

```typescript
// src/services/user.service.ts
import { UserRepository } from '../repositories/user.repository';
import { CreateUserDTO, UpdateUserDTO, User, PaginationResult } from '../types';
import { hashPassword } from '../utils/crypto';
import { AppError } from '../utils/errors';

interface FindAllOptions {
  page: number;
  limit: number;
  sort: string;
}

export class UserService {
  private repository: UserRepository;

  constructor() {
    this.repository = new UserRepository();
  }

  async findAll(options: FindAllOptions): Promise<PaginationResult<User>> {
    const { page, limit, sort } = options;
    const skip = (page - 1) * limit;

    const [users, total] = await Promise.all([
      this.repository.findAll({ skip, limit, sort }),
      this.repository.count(),
    ]);

    return {
      users,
      page,
      limit,
      total,
      totalPages: Math.ceil(total / limit),
    };
  }

  async findById(id: string): Promise<User | null> {
    return this.repository.findById(id);
  }

  async findByEmail(email: string): Promise<User | null> {
    return this.repository.findByEmail(email);
  }

  async create(data: CreateUserDTO): Promise<User> {
    const existing = await this.findByEmail(data.email);
    if (existing) {
      throw new AppError('Email already in use', 409);
    }

    const hashedPassword = await hashPassword(data.password);

    return this.repository.create({
      ...data,
      password: hashedPassword,
    });
  }

  async update(id: string, data: UpdateUserDTO): Promise<User | null> {
    const user = await this.findById(id);
    if (!user) {
      return null;
    }

    if (data.email && data.email !== user.email) {
      const existing = await this.findByEmail(data.email);
      if (existing) {
        throw new AppError('Email already in use', 409);
      }
    }

    return this.repository.update(id, data);
  }

  async delete(id: string): Promise<void> {
    const user = await this.findById(id);
    if (!user) {
      throw new AppError('User not found', 404);
    }

    await this.repository.delete(id);
  }
}
```

### Key Points

- Services never reference `req`, `res`, or `NextFunction`
- Throw `AppError` for domain-level errors (duplicate email, not found)
- Use `Promise.all` for independent async operations (performance)
- Hash passwords before persisting

## Authentication Middleware

### JWT Bearer Token Authentication

```typescript
// src/middlewares/auth.middleware.ts
import { Request, Response, NextFunction } from 'express';
import jwt from 'jsonwebtoken';
import { AppError } from '../utils/errors';

interface JwtPayload {
  userId: string;
  email: string;
}

declare global {
  namespace Express {
    interface Request {
      user?: JwtPayload;
    }
  }
}

export const authMiddleware = async (
  req: Request,
  _res: Response,
  next: NextFunction
) => {
  try {
    const authHeader = req.headers.authorization;

    if (!authHeader?.startsWith('Bearer ')) {
      throw new AppError('No token provided', 401);
    }

    const token = authHeader.split(' ')[1];
    const decoded = jwt.verify(
      token,
      process.env.JWT_SECRET!
    ) as JwtPayload;

    req.user = decoded;
    next();
  } catch (error) {
    if (error instanceof jwt.JsonWebTokenError) {
      next(new AppError('Invalid token', 401));
    } else {
      next(error);
    }
  }
};
```

### Key Points

- Extend Express `Request` type globally for `req.user`
- Extract token from `Authorization: Bearer <token>` header
- Catch `JsonWebTokenError` specifically for clear 401 responses
- Use `process.env.JWT_SECRET` (never hardcode)

## Role-Based Access Control

```typescript
export const requireRole = (...roles: string[]) => {
  return (req: Request, _res: Response, next: NextFunction) => {
    if (!req.user) {
      return next(new AppError('Authentication required', 401));
    }

    const userRole = (req.user as any).role;
    if (!roles.includes(userRole)) {
      return next(new AppError('Insufficient permissions', 403));
    }

    next();
  };
};

// Usage in routes:
// router.delete('/:id', authMiddleware, requireRole('admin'), controller.delete);
```

## Validation Schemas

### Zod Schema Examples

```typescript
// src/schemas/user.schema.ts
import { z } from 'zod';

export const createUserSchema = z.object({
  body: z.object({
    email: z.string().email('Invalid email format'),
    password: z.string().min(8, 'Password must be at least 8 characters'),
    name: z.string().min(2, 'Name must be at least 2 characters'),
  }),
});

export const updateUserSchema = z.object({
  body: z.object({
    email: z.string().email().optional(),
    name: z.string().min(2).optional(),
  }),
  params: z.object({
    id: z.string().uuid('Invalid user ID'),
  }),
});
```

### Key Points

- Validate `body`, `query`, and `params` together in one schema
- Use descriptive error messages for each field
- Mark optional fields explicitly with `.optional()`
- Use `.uuid()`, `.email()`, `.min()`, `.max()` for common constraints

## Rate Limiting

### Basic Rate Limiter

```typescript
// src/middlewares/rateLimit.middleware.ts
import rateLimit from 'express-rate-limit';

// General API limiter
export const apiLimiter = rateLimit({
  windowMs: 15 * 60 * 1000, // 15 minutes
  max: 100,                  // 100 requests per window per IP
  message: {
    status: 'error',
    message: 'Too many requests, please try again later',
  },
  standardHeaders: true,
  legacyHeaders: false,
});

// Stricter limiter for auth routes
export const authLimiter = rateLimit({
  windowMs: 60 * 60 * 1000, // 1 hour
  max: 5,                    // 5 failed attempts per hour
  message: {
    status: 'error',
    message: 'Too many login attempts, please try again later',
  },
});
```

### Redis-Based Rate Limiter (Distributed Systems)

```typescript
import rateLimit from 'express-rate-limit';
import RedisStore from 'rate-limit-redis';
import { createClient } from 'redis';

export const createRedisLimiter = async () => {
  const client = createClient({ url: process.env.REDIS_URL });
  await client.connect();

  return rateLimit({
    windowMs: 15 * 60 * 1000,
    max: 100,
    store: new RedisStore({
      sendCommand: (...args: string[]) => client.sendCommand(args),
    }),
  });
};
```

### Usage

```typescript
// Apply globally
app.use(apiLimiter);

// Apply to specific routes
router.use('/auth', authLimiter, authRoutes);
```

## Database Integration (Prisma)

### Database Client Setup

```typescript
// src/config/database.ts
import { PrismaClient } from '@prisma/client';

const prisma = new PrismaClient({
  log: process.env.NODE_ENV === 'development'
    ? ['query', 'error', 'warn']
    : ['error'],
});

export { prisma as db };
```

### Repository Pattern

```typescript
// src/repositories/user.repository.ts
import { db } from '../config/database';
import { CreateUserDTO, UpdateUserDTO } from '../types';

export class UserRepository {
  async findAll(options: { skip: number; limit: number; sort: string }) {
    return db.user.findMany({
      skip: options.skip,
      take: options.limit,
      orderBy: { [options.sort]: 'desc' },
      select: {
        id: true,
        email: true,
        name: true,
        createdAt: true,
        updatedAt: true,
      },
    });
  }

  async findById(id: string) {
    return db.user.findUnique({
      where: { id },
      select: {
        id: true,
        email: true,
        name: true,
        createdAt: true,
        updatedAt: true,
      },
    });
  }

  async findByEmail(email: string) {
    return db.user.findUnique({ where: { email } });
  }

  async create(data: CreateUserDTO) {
    return db.user.create({
      data,
      select: { id: true, email: true, name: true, createdAt: true },
    });
  }

  async update(id: string, data: UpdateUserDTO) {
    return db.user.update({
      where: { id },
      data,
      select: { id: true, email: true, name: true, updatedAt: true },
    });
  }

  async delete(id: string) {
    return db.user.delete({ where: { id } });
  }

  async count() {
    return db.user.count();
  }
}
```

### Key Points

- Use `select` to limit returned fields (never return passwords)
- Use `findUnique` for single-record lookups (indexed fields)
- Enable query logging in development only
- Use transactions for multi-step writes: `db.$transaction([...])`

## Not Found Handler

```typescript
// src/middlewares/notFound.middleware.ts
import { Request, Response } from 'express';

export const notFoundHandler = (req: Request, res: Response) => {
  res.status(404).json({
    status: 'error',
    message: `Route ${req.originalUrl} not found`,
  });
};
```

Place this middleware **after** all routes but **before** the error handler in `app.ts`.

## Integration Testing

### Full Test Suite Example

```typescript
// tests/user.test.ts
import request from 'supertest';
import app from '../src/app';
import { db } from '../src/config/database';

describe('User API', () => {
  beforeAll(async () => {
    await db.$connect();
  });

  afterAll(async () => {
    await db.$disconnect();
  });

  beforeEach(async () => {
    await db.user.deleteMany();
  });

  describe('GET /api/v1/users', () => {
    it('should return empty array when no users', async () => {
      const response = await request(app)
        .get('/api/v1/users')
        .expect('Content-Type', /json/)
        .expect(200);

      expect(response.body.data).toEqual([]);
    });

    it('should return users with pagination', async () => {
      // Seed 15 test users
      await seedTestUsers(15);

      const response = await request(app)
        .get('/api/v1/users?page=1&limit=10')
        .expect(200);

      expect(response.body.data).toHaveLength(10);
      expect(response.body.meta.total).toBe(15);
      expect(response.body.meta.totalPages).toBe(2);
    });
  });

  describe('POST /api/v1/users', () => {
    it('should create a new user', async () => {
      const userData = {
        email: 'test@example.com',
        password: 'password123',
        name: 'Test User',
      };

      const response = await request(app)
        .post('/api/v1/users')
        .send(userData)
        .expect(201);

      expect(response.body.data.email).toBe(userData.email);
      expect(response.body.data.name).toBe(userData.name);
      expect(response.body.data.password).toBeUndefined();
    });

    it('should return 400 for invalid email', async () => {
      const response = await request(app)
        .post('/api/v1/users')
        .send({ email: 'invalid', password: 'password123', name: 'Test' })
        .expect(400);

      expect(response.body.status).toBe('error');
    });

    it('should return 409 for duplicate email', async () => {
      const userData = { email: 'dup@example.com', password: 'pass1234', name: 'Dup' };

      await request(app).post('/api/v1/users').send(userData);

      const response = await request(app)
        .post('/api/v1/users')
        .send(userData)
        .expect(409);

      expect(response.body.message).toContain('already in use');
    });
  });

  describe('GET /api/v1/users/:id', () => {
    it('should return user by id', async () => {
      const user = await createTestUser();

      const response = await request(app)
        .get(`/api/v1/users/${user.id}`)
        .expect(200);

      expect(response.body.data.id).toBe(user.id);
    });

    it('should return 404 for non-existent user', async () => {
      await request(app)
        .get('/api/v1/users/non-existent-id')
        .expect(404);
    });
  });
});
```

### Testing Guidelines

- Test against `app` (not a running `server`) to avoid port conflicts
- Clean the database in `beforeEach` for test isolation
- Connect/disconnect in `beforeAll`/`afterAll` for performance
- Test success paths, validation errors, and domain errors (409, 404)
- Assert response body structure, not just status codes
- Never assert on internal implementation details (query counts, etc.)

## Configuration Files

### TypeScript Configuration

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "commonjs",
    "lib": ["ES2020"],
    "outDir": "./dist",
    "rootDir": "./src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "resolveJsonModule": true,
    "declaration": true,
    "declarationMap": true,
    "sourceMap": true,
    "baseUrl": "./src",
    "paths": {
      "@/*": ["./*"]
    }
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist"]
}
```

### Recommended Dependencies

```json
{
  "dependencies": {
    "express": "^4.18.2",
    "cors": "^2.8.5",
    "helmet": "^7.1.0",
    "morgan": "^1.10.0",
    "dotenv": "^16.3.1",
    "zod": "^3.22.4",
    "jsonwebtoken": "^9.0.2",
    "bcryptjs": "^2.4.3",
    "@prisma/client": "^5.x",
    "express-rate-limit": "^7.x"
  },
  "devDependencies": {
    "typescript": "^5.x",
    "@types/express": "^4.17.21",
    "@types/node": "^20.x",
    "@types/cors": "^2.8.x",
    "@types/morgan": "^1.9.x",
    "ts-node-dev": "^2.0.0",
    "jest": "^29.x",
    "supertest": "^6.x",
    "@types/supertest": "^6.x"
  }
}
```
