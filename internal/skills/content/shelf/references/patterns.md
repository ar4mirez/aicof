# Shelf Patterns Reference

## Contents

- [CORS Middleware](#cors-middleware)
- [Authentication Middleware (JWT)](#authentication-middleware-jwt)
- [Optional Authentication](#optional-authentication)
- [Rate Limiting Middleware](#rate-limiting-middleware)
- [CRUD Handler Pattern](#crud-handler-pattern)
- [Repository Pattern](#repository-pattern)
- [Service Layer Pattern](#service-layer-pattern)
- [Model Pattern (Freezed)](#model-pattern-freezed)
- [WebSocket Handler](#websocket-handler)
- [Static File Serving](#static-file-serving)
- [Health Check Handler](#health-check-handler)
- [Integration Testing](#integration-testing)
- [Configuration Pattern](#configuration-pattern)
- [Dockerfile](#dockerfile)

## CORS Middleware

```dart
// lib/src/middleware/cors_middleware.dart
import 'package:shelf/shelf.dart';

Middleware corsMiddleware({
  List<String> allowedOrigins = const ['*'],
  List<String> allowedMethods = const ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS'],
  List<String> allowedHeaders = const ['Content-Type', 'Authorization'],
}) {
  return (Handler innerHandler) {
    return (Request request) async {
      // Handle preflight requests
      if (request.method == 'OPTIONS') {
        return Response.ok(
          '',
          headers: _corsHeaders(allowedOrigins, allowedMethods, allowedHeaders),
        );
      }

      final response = await innerHandler(request);

      return response.change(
        headers: {
          ...response.headers,
          ..._corsHeaders(allowedOrigins, allowedMethods, allowedHeaders),
        },
      );
    };
  };
}

Map<String, String> _corsHeaders(
  List<String> origins,
  List<String> methods,
  List<String> headers,
) {
  return {
    'Access-Control-Allow-Origin': origins.join(', '),
    'Access-Control-Allow-Methods': methods.join(', '),
    'Access-Control-Allow-Headers': headers.join(', '),
    'Access-Control-Max-Age': '86400',
  };
}
```

**Rules:**
- Never use `*` for allowed origins in production; whitelist specific domains
- Always handle `OPTIONS` preflight requests and return early
- Set `Access-Control-Max-Age` to reduce preflight request frequency
- Apply CORS middleware early in the pipeline (before auth)

## Authentication Middleware (JWT)

```dart
// lib/src/middleware/auth_middleware.dart
import 'dart:convert';
import 'package:shelf/shelf.dart';
import 'package:dart_jsonwebtoken/dart_jsonwebtoken.dart';
import '../models/user.dart';
import '../config/config.dart';

Middleware authMiddleware() {
  return (Handler innerHandler) {
    return (Request request) async {
      final authHeader = request.headers['Authorization'];

      if (authHeader == null || !authHeader.startsWith('Bearer ')) {
        throw UnauthorizedException();
      }

      final token = authHeader.substring(7);

      try {
        final config = request.context['config'] as Config;
        final jwt = JWT.verify(token, SecretKey(config.jwtSecret));

        final payload = jwt.payload as Map<String, dynamic>;
        final user = User.fromJson(payload['user'] as Map<String, dynamic>);

        // Add user to request context for downstream handlers
        final updatedRequest = request.change(
          context: {...request.context, 'user': user},
        );

        return await innerHandler(updatedRequest);
      } on JWTExpiredException {
        throw UnauthorizedException('Token expired');
      } on JWTException {
        throw UnauthorizedException('Invalid token');
      }
    };
  };
}
```

**Rules:**
- Always check for `Bearer ` prefix before extracting the token
- Pass the authenticated user via `request.change(context:)`, not global state
- Catch specific JWT exceptions (`JWTExpiredException`, `JWTException`) separately
- Throw domain exceptions (`UnauthorizedException`), not HTTP responses -- let error middleware handle status codes

## Optional Authentication

Use when a route works for both authenticated and anonymous users (e.g., public feed with personalization).

```dart
Middleware optionalAuthMiddleware() {
  return (Handler innerHandler) {
    return (Request request) async {
      final authHeader = request.headers['Authorization'];

      if (authHeader != null && authHeader.startsWith('Bearer ')) {
        final token = authHeader.substring(7);

        try {
          final config = request.context['config'] as Config;
          final jwt = JWT.verify(token, SecretKey(config.jwtSecret));

          final payload = jwt.payload as Map<String, dynamic>;
          final user = User.fromJson(payload['user'] as Map<String, dynamic>);

          return await innerHandler(
            request.change(context: {...request.context, 'user': user}),
          );
        } catch (_) {
          // Invalid token -- proceed as anonymous
        }
      }

      return await innerHandler(request);
    };
  };
}
```

**Rules:**
- Never fail the request on invalid tokens; silently proceed as anonymous
- Handlers must check `request.context['user']` for null before assuming authentication
- Use for endpoints like "list items" where auth adds personalization but is not required

## Rate Limiting Middleware

```dart
// lib/src/middleware/rate_limit_middleware.dart
import 'dart:convert';
import 'package:shelf/shelf.dart';

class RateLimiter {
  final int maxRequests;
  final Duration window;
  final Map<String, List<DateTime>> _requests = {};

  RateLimiter({
    this.maxRequests = 100,
    this.window = const Duration(minutes: 1),
  });

  bool isAllowed(String key) {
    final now = DateTime.now();
    final windowStart = now.subtract(window);

    _requests[key] = (_requests[key] ?? [])
        .where((time) => time.isAfter(windowStart))
        .toList();

    if (_requests[key]!.length >= maxRequests) {
      return false;
    }

    _requests[key]!.add(now);
    return true;
  }

  int remaining(String key) {
    final now = DateTime.now();
    final windowStart = now.subtract(window);

    final count = (_requests[key] ?? [])
        .where((time) => time.isAfter(windowStart))
        .length;

    return (maxRequests - count).clamp(0, maxRequests);
  }
}

Middleware rateLimitMiddleware({
  int maxRequests = 100,
  Duration window = const Duration(minutes: 1),
}) {
  final limiter = RateLimiter(maxRequests: maxRequests, window: window);

  return (Handler innerHandler) {
    return (Request request) async {
      final clientIp = request.headers['X-Forwarded-For'] ??
          request.headers['X-Real-IP'] ??
          'unknown';

      if (!limiter.isAllowed(clientIp)) {
        return Response(
          429,
          body: jsonEncode({'error': 'Too many requests'}),
          headers: {
            'Content-Type': 'application/json',
            'Retry-After': '60',
            'X-RateLimit-Remaining': '0',
          },
        );
      }

      final response = await innerHandler(request);

      return response.change(
        headers: {
          ...response.headers,
          'X-RateLimit-Remaining': '${limiter.remaining(clientIp)}',
          'X-RateLimit-Limit': '$maxRequests',
        },
      );
    };
  };
}
```

**Rules:**
- Use stricter limits on auth endpoints (e.g., 10 per minute for login)
- Always include `Retry-After` and `X-RateLimit-*` headers in responses
- In production, use Redis or similar shared store instead of in-memory map (per-instance limiting is insufficient behind a load balancer)
- Extract client IP from `X-Forwarded-For` when behind a reverse proxy

## CRUD Handler Pattern

Full handler with public and protected routes, pagination, and authorization checks.

```dart
// lib/src/handlers/users_handler.dart
import 'dart:convert';
import 'package:shelf/shelf.dart';
import 'package:shelf_router/shelf_router.dart';
import '../models/user.dart';
import '../services/user_service.dart';
import '../middleware/auth_middleware.dart';

class UsersHandler {
  final UserService _userService;

  UsersHandler(this._userService);

  Router get router {
    final router = Router();

    router.post('/register', _register);
    router.post('/login', _login);
    router.get('/', _withAuth(_getAll));
    router.get('/<id>', _withAuth(_getById));
    router.put('/<id>', _withAuth(_update));
    router.delete('/<id>', _withAuth(_delete));

    return router;
  }

  Handler _withAuth(Handler handler) {
    return const Pipeline()
        .addMiddleware(authMiddleware())
        .addHandler(handler);
  }

  Future<Response> _register(Request request) async {
    final body = await request.readAsString();
    final json = jsonDecode(body) as Map<String, dynamic>;

    final user = await _userService.register(
      email: json['email'] as String,
      password: json['password'] as String,
      name: json['name'] as String,
    );

    return Response(
      201,
      body: jsonEncode(user.toJson()),
      headers: {'Content-Type': 'application/json'},
    );
  }

  Future<Response> _login(Request request) async {
    final body = await request.readAsString();
    final json = jsonDecode(body) as Map<String, dynamic>;

    final result = await _userService.login(
      email: json['email'] as String,
      password: json['password'] as String,
    );

    return Response.ok(
      jsonEncode({
        'token': result.token,
        'user': result.user.toJson(),
      }),
      headers: {'Content-Type': 'application/json'},
    );
  }

  Future<Response> _getAll(Request request) async {
    final page = int.tryParse(
            request.url.queryParameters['page'] ?? '1') ?? 1;
    final limit = int.tryParse(
            request.url.queryParameters['limit'] ?? '20') ?? 20;

    final users = await _userService.getAll(page: page, limit: limit);

    return Response.ok(
      jsonEncode({
        'data': users.map((u) => u.toJson()).toList(),
        'page': page,
        'limit': limit,
      }),
      headers: {'Content-Type': 'application/json'},
    );
  }

  Future<Response> _getById(Request request, String id) async {
    final user = await _userService.getById(id);

    if (user == null) {
      throw NotFoundException('User not found');
    }

    return Response.ok(
      jsonEncode(user.toJson()),
      headers: {'Content-Type': 'application/json'},
    );
  }

  Future<Response> _update(Request request, String id) async {
    final currentUser = request.context['user'] as User;

    if (currentUser.id != id && !currentUser.isAdmin) {
      throw UnauthorizedException();
    }

    final body = await request.readAsString();
    final json = jsonDecode(body) as Map<String, dynamic>;
    final user = await _userService.update(id, json);

    return Response.ok(
      jsonEncode(user.toJson()),
      headers: {'Content-Type': 'application/json'},
    );
  }

  Future<Response> _delete(Request request, String id) async {
    final currentUser = request.context['user'] as User;

    if (currentUser.id != id && !currentUser.isAdmin) {
      throw UnauthorizedException();
    }

    await _userService.delete(id);
    return Response(204);
  }
}
```

## Repository Pattern

Define an abstract interface and implement with concrete storage.

```dart
// lib/src/repositories/user_repository.dart
import '../models/user.dart';

abstract class UserRepository {
  Future<User> create({
    required String email,
    required String passwordHash,
    required String name,
  });
  Future<User?> findById(String id);
  Future<User?> findByEmail(String email);
  Future<List<User>> findAll({int offset = 0, int limit = 20});
  Future<User> update(String id, {String? name, String? email});
  Future<void> delete(String id);
  Future<void> close();
}

// In-memory implementation (swap with PostgreSQL, etc.)
class InMemoryUserRepository implements UserRepository {
  final Map<String, User> _users = {};
  int _idCounter = 0;

  @override
  Future<User> create({
    required String email,
    required String passwordHash,
    required String name,
  }) async {
    final id = (++_idCounter).toString();
    final user = User(
      id: id,
      email: email,
      name: name,
      passwordHash: passwordHash,
      createdAt: DateTime.now(),
    );
    _users[id] = user;
    return user;
  }

  @override
  Future<User?> findById(String id) async => _users[id];

  @override
  Future<User?> findByEmail(String email) async =>
      _users.values.where((u) => u.email == email).firstOrNull;

  @override
  Future<List<User>> findAll({int offset = 0, int limit = 20}) async =>
      _users.values.skip(offset).take(limit).toList();

  @override
  Future<User> update(String id, {String? name, String? email}) async {
    final user = _users[id];
    if (user == null) throw NotFoundException('User not found');

    final updated = user.copyWith(
      name: name ?? user.name,
      email: email ?? user.email,
      updatedAt: DateTime.now(),
    );
    _users[id] = updated;
    return updated;
  }

  @override
  Future<void> delete(String id) async => _users.remove(id);

  @override
  Future<void> close() async {}
}
```

**Rules:**
- Always define an abstract class for the repository interface
- Keep repository methods focused on data access -- no business logic
- Return domain models, not raw database types
- Include a `close()` method for cleaning up database connections
- Use `firstOrNull` instead of `firstWhere` with `orElse` for nullable lookups

## Service Layer Pattern

Services contain business logic and coordinate between repositories.

```dart
// lib/src/services/user_service.dart
import 'package:dart_jsonwebtoken/dart_jsonwebtoken.dart';
import 'package:bcrypt/bcrypt.dart';
import '../config/config.dart';
import '../models/user.dart';
import '../repositories/user_repository.dart';
import '../middleware/auth_middleware.dart';

class UserService {
  final UserRepository _repository;
  final Config _config;

  UserService(this._repository, this._config);

  Future<User> register({
    required String email,
    required String password,
    required String name,
  }) async {
    _validateEmail(email);
    _validatePassword(password);

    final existing = await _repository.findByEmail(email);
    if (existing != null) {
      throw ValidationException(
        'Email already exists',
        {'email': ['Email is already registered']},
      );
    }

    final passwordHash = BCrypt.hashpw(password, BCrypt.gensalt());

    return _repository.create(
      email: email.toLowerCase().trim(),
      passwordHash: passwordHash,
      name: name.trim(),
    );
  }

  Future<LoginResult> login({
    required String email,
    required String password,
  }) async {
    final user = await _repository.findByEmail(email.toLowerCase().trim());

    if (user == null || user.passwordHash == null) {
      throw UnauthorizedException('Invalid credentials');
    }

    if (!BCrypt.checkpw(password, user.passwordHash!)) {
      throw UnauthorizedException('Invalid credentials');
    }

    final jwt = JWT(
      {'user': user.toPublicJson(), 'iat': DateTime.now().millisecondsSinceEpoch ~/ 1000},
      issuer: 'myapp',
      subject: user.id,
    );

    final token = jwt.sign(
      SecretKey(_config.jwtSecret),
      expiresIn: const Duration(days: 7),
    );

    return LoginResult(token: token, user: user);
  }

  Future<List<User>> getAll({int page = 1, int limit = 20}) async {
    return _repository.findAll(offset: (page - 1) * limit, limit: limit);
  }

  Future<User?> getById(String id) async => _repository.findById(id);

  Future<User> update(String id, Map<String, dynamic> data) async {
    final user = await _repository.findById(id);
    if (user == null) throw NotFoundException('User not found');

    return _repository.update(
      id,
      name: data['name'] as String?,
      email: data['email'] as String?,
    );
  }

  Future<void> delete(String id) async {
    final user = await _repository.findById(id);
    if (user == null) throw NotFoundException('User not found');
    await _repository.delete(id);
  }

  void _validateEmail(String email) {
    final emailRegex = RegExp(r'^[\w-\.]+@([\w-]+\.)+[\w-]{2,4}$');
    if (!emailRegex.hasMatch(email)) {
      throw ValidationException('Invalid email', {'email': ['Please enter a valid email']});
    }
  }

  void _validatePassword(String password) {
    if (password.length < 8) {
      throw ValidationException(
        'Invalid password',
        {'password': ['Password must be at least 8 characters']},
      );
    }
  }
}
```

**Rules:**
- Services accept repositories and config via constructor injection
- Never reference `Request` or `Response` in services -- they are framework-agnostic
- Use the same typed exceptions as handlers (`ValidationException`, `NotFoundException`)
- Always validate and sanitize inputs before passing to the repository
- Never return password hashes to callers; use `toPublicJson()` for user-facing data

## Model Pattern (Freezed)

```dart
// lib/src/models/user.dart
import 'package:freezed_annotation/freezed_annotation.dart';

part 'user.freezed.dart';
part 'user.g.dart';

@freezed
class User with _$User {
  const User._();

  const factory User({
    required String id,
    required String email,
    required String name,
    @Default(false) bool isAdmin,
    @JsonKey(includeToJson: false) String? passwordHash,
    required DateTime createdAt,
    DateTime? updatedAt,
  }) = _User;

  factory User.fromJson(Map<String, dynamic> json) => _$UserFromJson(json);

  Map<String, dynamic> toPublicJson() {
    return {
      'id': id,
      'email': email,
      'name': name,
      'isAdmin': isAdmin,
      'createdAt': createdAt.toIso8601String(),
      if (updatedAt != null) 'updatedAt': updatedAt!.toIso8601String(),
    };
  }
}

@freezed
class LoginResult with _$LoginResult {
  const factory LoginResult({
    required String token,
    required User user,
  }) = _LoginResult;
}
```

**Rules:**
- Use `@JsonKey(includeToJson: false)` to exclude sensitive fields (password hashes)
- Provide a `toPublicJson()` method for user-facing serialization
- Add `const User._()` to enable custom methods on freezed classes
- Run `dart run build_runner build` after model changes
- Use `@Default(false)` for boolean fields with sensible defaults

## WebSocket Handler

```dart
// lib/src/handlers/websocket_handler.dart
import 'dart:convert';
import 'package:shelf/shelf.dart';
import 'package:shelf_web_socket/shelf_web_socket.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

class WebSocketHandler {
  final Set<WebSocketChannel> _clients = {};

  Handler get handler => webSocketHandler(_onConnection);

  void _onConnection(WebSocketChannel webSocket) {
    _clients.add(webSocket);
    print('Client connected. Total: ${_clients.length}');

    webSocket.stream.listen(
      (message) => _handleMessage(webSocket, message),
      onDone: () {
        _clients.remove(webSocket);
        print('Client disconnected. Total: ${_clients.length}');
      },
      onError: (error) {
        print('WebSocket error: $error');
        _clients.remove(webSocket);
      },
    );
  }

  void _handleMessage(WebSocketChannel sender, dynamic message) {
    try {
      final data = jsonDecode(message as String) as Map<String, dynamic>;
      final type = data['type'] as String;

      switch (type) {
        case 'ping':
          sender.sink.add(jsonEncode({'type': 'pong'}));
          break;
        case 'broadcast':
          _broadcast(data['payload']);
          break;
        default:
          sender.sink.add(jsonEncode({
            'type': 'error',
            'message': 'Unknown message type',
          }));
      }
    } catch (e) {
      sender.sink.add(jsonEncode({
        'type': 'error',
        'message': 'Invalid message format',
      }));
    }
  }

  void _broadcast(dynamic payload) {
    final message = jsonEncode({'type': 'message', 'payload': payload});
    for (final client in _clients) {
      client.sink.add(message);
    }
  }
}
```

**Mounting WebSocket handlers:**

```dart
// In app.dart or routes setup
final wsHandler = WebSocketHandler();

final router = Router()
  ..mount('/ws', wsHandler.handler)
  ..mount('/api/v1', apiRouter.call);
```

**Rules:**
- Use `shelf_web_socket` package (`webSocketHandler` function)
- Track connected clients in a `Set<WebSocketChannel>` for broadcasting
- Always handle `onDone` and `onError` to clean up disconnected clients
- Validate incoming message format before processing
- Use JSON encoding for message protocol consistency
- Consider adding heartbeat/ping-pong for connection health monitoring

## Static File Serving

```dart
import 'package:shelf/shelf.dart';
import 'package:shelf_static/shelf_static.dart';

// Serve files from a directory
final staticHandler = createStaticHandler(
  'public',
  defaultDocument: 'index.html',
);

// Combine with API routes using Cascade
final cascade = Cascade()
    .add(apiRouter)
    .add(staticHandler);

final handler = const Pipeline()
    .addMiddleware(loggingMiddleware())
    .addHandler(cascade.handler);
```

**Rules:**
- Use `shelf_static` package for file serving
- Set `defaultDocument` for SPA fallback
- Place API routes before static handler in `Cascade` so API takes priority
- Validate that the static directory exists and is within the expected path
- Never serve directories containing sensitive files (e.g., `.env`, `lib/`)

## Health Check Handler

```dart
// lib/src/handlers/health_handler.dart
import 'dart:convert';
import 'package:shelf/shelf.dart';
import 'package:shelf_router/shelf_router.dart';

class HealthHandler {
  Router get router {
    final router = Router();
    router.get('/', _health);
    router.get('/ready', _ready);
    router.get('/live', _live);
    return router;
  }

  Response _health(Request request) {
    return Response.ok(
      jsonEncode({
        'status': 'healthy',
        'timestamp': DateTime.now().toIso8601String(),
      }),
      headers: {'Content-Type': 'application/json'},
    );
  }

  Response _ready(Request request) {
    // Check database, cache, and external service connectivity
    return Response.ok(
      jsonEncode({'ready': true}),
      headers: {'Content-Type': 'application/json'},
    );
  }

  Response _live(Request request) {
    return Response.ok(
      jsonEncode({'live': true}),
      headers: {'Content-Type': 'application/json'},
    );
  }
}
```

**Rules:**
- `/health` -- general status (for monitoring dashboards)
- `/ready` -- readiness probe (Kubernetes); check database and dependencies
- `/live` -- liveness probe (Kubernetes); respond if process is alive
- Health endpoints should not require authentication
- Mount at the root level, not behind `/api/v1`

## Integration Testing

Test handlers directly using `shelf` without starting an HTTP server.

```dart
// test/handlers/users_handler_test.dart
import 'dart:convert';
import 'package:shelf/shelf.dart';
import 'package:test/test.dart';
import 'package:mocktail/mocktail.dart';
import 'package:myapp/src/handlers/users_handler.dart';
import 'package:myapp/src/services/user_service.dart';
import 'package:myapp/src/models/user.dart';

class MockUserService extends Mock implements UserService {}

void main() {
  late MockUserService mockService;
  late UsersHandler handler;

  setUp(() {
    mockService = MockUserService();
    handler = UsersHandler(mockService);
  });

  group('POST /register', () {
    test('returns 201 with created user', () async {
      final user = User(
        id: '1',
        email: 'test@example.com',
        name: 'Test User',
        createdAt: DateTime.now(),
      );

      when(() => mockService.register(
            email: any(named: 'email'),
            password: any(named: 'password'),
            name: any(named: 'name'),
          )).thenAnswer((_) async => user);

      final request = Request(
        'POST',
        Uri.parse('http://localhost/register'),
        body: jsonEncode({
          'email': 'test@example.com',
          'password': 'password123',
          'name': 'Test User',
        }),
      );

      final response = await handler.router.call(request);

      expect(response.statusCode, equals(201));
      final body = jsonDecode(await response.readAsString());
      expect(body['email'], equals('test@example.com'));
    });
  });

  group('GET /', () {
    test('returns paginated users', () async {
      final users = [
        User(id: '1', email: 'a@test.com', name: 'A', createdAt: DateTime.now()),
        User(id: '2', email: 'b@test.com', name: 'B', createdAt: DateTime.now()),
      ];

      when(() => mockService.getAll(
            page: any(named: 'page'),
            limit: any(named: 'limit'),
          )).thenAnswer((_) async => users);

      final request = Request(
        'GET',
        Uri.parse('http://localhost/?page=1&limit=20'),
      ).change(context: {
        'user': User(
          id: '1',
          email: 'admin@test.com',
          name: 'Admin',
          createdAt: DateTime.now(),
        ),
      });

      final response = await handler.router.call(request);

      expect(response.statusCode, equals(200));
      final body = jsonDecode(await response.readAsString());
      expect(body['data'], hasLength(2));
    });
  });
}
```

**Testing Rules:**
- Call `handler.router.call(request)` directly -- no HTTP server needed
- Use `mocktail` for mocking services (registerFallbackValue for complex types)
- Use `request.change(context:)` to inject authenticated user context in tests
- Test both success and error paths (validation, not found, unauthorized)
- Group tests by HTTP method and route using `group`
- Each test should be independent; use `setUp` for shared setup

## Configuration Pattern

```dart
// lib/src/config/config.dart
import 'dart:io';

class Config {
  final int port;
  final String databaseUrl;
  final String jwtSecret;
  final bool isDevelopment;

  const Config({
    required this.port,
    required this.databaseUrl,
    required this.jwtSecret,
    required this.isDevelopment,
  });

  factory Config.fromEnvironment() {
    return Config(
      port: int.parse(Platform.environment['PORT'] ?? '8080'),
      databaseUrl: Platform.environment['DATABASE_URL'] ??
          'postgres://localhost/myapp',
      jwtSecret: Platform.environment['JWT_SECRET'] ??
          (Platform.environment['DART_ENV'] == 'production'
              ? (throw StateError('JWT_SECRET required in production'))
              : 'development-secret-key'),
      isDevelopment: Platform.environment['DART_ENV'] != 'production',
    );
  }
}
```

**Rules:**
- Read all values from `Platform.environment`
- Provide dev defaults for non-secret values; throw for missing secrets in production
- Make the class immutable with `const` constructor and `final` fields
- Inject `Config` into `Application`, services, and middleware via constructor parameters
- Never expose `Config` globally; pass it through the dependency chain

## Dockerfile

Multi-stage build for minimal production image.

```dockerfile
# Build stage
FROM dart:stable AS build
WORKDIR /app

# Copy dependencies first (layer caching)
COPY pubspec.* ./
RUN dart pub get

# Copy source and build
COPY . .
RUN dart run build_runner build --delete-conflicting-outputs
RUN dart compile exe bin/server.dart -o bin/server

# Runtime stage (minimal image)
FROM scratch
COPY --from=build /runtime/ /
COPY --from=build /app/bin/server /app/bin/server

EXPOSE 8080
CMD ["/app/bin/server"]
```

**Rules:**
- Use multi-stage build to keep image small (final image has no SDK)
- Copy `pubspec.*` first to leverage Docker layer caching for dependency resolution
- Run `build_runner` before `dart compile exe` for generated code
- Use `FROM scratch` for the runtime stage -- AOT-compiled Dart binary is self-contained
- Always set `EXPOSE` to document the expected port
- Set environment variables via Docker Compose or Kubernetes, not in the Dockerfile
