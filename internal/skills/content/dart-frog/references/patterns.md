# Dart Frog Patterns Reference

## Contents

- [Route Patterns](#route-patterns)
- [WebSocket Integration](#websocket-integration)
- [Database Integration](#database-integration)
- [Authentication](#authentication)
- [Configuration](#configuration)
- [Testing Patterns](#testing-patterns)
- [Deployment](#deployment)

## Route Patterns

### CRUD Resource Handler

Complete handler for a collection endpoint supporting GET (list) and POST (create):

```dart
// routes/api/v1/users/index.dart
import 'dart:io';

import 'package:dart_frog/dart_frog.dart';

import 'package:myapp/myapp.dart';

Future<Response> onRequest(RequestContext context) async {
  return switch (context.request.method) {
    HttpMethod.get => _getUsers(context),
    HttpMethod.post => _createUser(context),
    _ => Future.value(Response(statusCode: HttpStatus.methodNotAllowed)),
  };
}

Future<Response> _getUsers(RequestContext context) async {
  final userService = context.read<UserService>();

  final page = int.tryParse(
    context.request.uri.queryParameters['page'] ?? '1',
  ) ?? 1;
  final limit = int.tryParse(
    context.request.uri.queryParameters['limit'] ?? '20',
  ) ?? 20;

  final users = await userService.getAll(page: page, limit: limit);

  return Response.json({
    'data': users.map((u) => u.toJson()).toList(),
    'page': page,
    'limit': limit,
  });
}

Future<Response> _createUser(RequestContext context) async {
  final userService = context.read<UserService>();
  final body = await context.request.json() as Map<String, dynamic>;

  try {
    final user = await userService.create(
      email: body['email'] as String,
      name: body['name'] as String,
      password: body['password'] as String,
    );

    return Response.json(user.toJson(), statusCode: HttpStatus.created);
  } on ValidationException catch (e) {
    return Response.json(
      {'error': e.message, 'errors': e.errors},
      statusCode: HttpStatus.unprocessableEntity,
    );
  }
}
```

### Single Resource Handler with Authorization

Handler for an individual resource supporting GET, PUT, and DELETE with ownership checks:

```dart
// routes/api/v1/users/[id].dart
import 'dart:io';

import 'package:dart_frog/dart_frog.dart';

import 'package:myapp/myapp.dart';

Future<Response> onRequest(RequestContext context, String id) async {
  return switch (context.request.method) {
    HttpMethod.get => _getUser(context, id),
    HttpMethod.put => _updateUser(context, id),
    HttpMethod.delete => _deleteUser(context, id),
    _ => Future.value(Response(statusCode: HttpStatus.methodNotAllowed)),
  };
}

Future<Response> _getUser(RequestContext context, String id) async {
  final userService = context.read<UserService>();
  final user = await userService.getById(id);

  if (user == null) {
    return Response.json(
      {'error': 'User not found'},
      statusCode: HttpStatus.notFound,
    );
  }

  return Response.json(user.toJson());
}

Future<Response> _updateUser(RequestContext context, String id) async {
  final userService = context.read<UserService>();
  final currentUser = context.read<User?>();

  if (currentUser == null || (currentUser.id != id && !currentUser.isAdmin)) {
    return Response.json(
      {'error': 'Forbidden'},
      statusCode: HttpStatus.forbidden,
    );
  }

  final body = await context.request.json() as Map<String, dynamic>;

  try {
    final user = await userService.update(id, body);
    return Response.json(user.toJson());
  } on NotFoundException {
    return Response.json(
      {'error': 'User not found'},
      statusCode: HttpStatus.notFound,
    );
  }
}

Future<Response> _deleteUser(RequestContext context, String id) async {
  final userService = context.read<UserService>();
  final currentUser = context.read<User?>();

  if (currentUser == null || (currentUser.id != id && !currentUser.isAdmin)) {
    return Response.json(
      {'error': 'Forbidden'},
      statusCode: HttpStatus.forbidden,
    );
  }

  try {
    await userService.delete(id);
    return Response(statusCode: HttpStatus.noContent);
  } on NotFoundException {
    return Response.json(
      {'error': 'User not found'},
      statusCode: HttpStatus.notFound,
    );
  }
}
```

### Nested Dynamic Routes

When resources are scoped under a parent (e.g., posts belonging to a user):

```dart
// routes/api/v1/users/[userId]/posts/[postId].dart
import 'package:dart_frog/dart_frog.dart';

Future<Response> onRequest(
  RequestContext context,
  String userId,
  String postId,
) async {
  final postService = context.read<PostService>();
  final post = await postService.getByUserAndId(userId, postId);

  if (post == null) {
    return Response.json(
      {'error': 'Post not found'},
      statusCode: 404,
    );
  }

  return Response.json(post.toJson());
}
```

## WebSocket Integration

Add the `dart_frog_web_socket` package and create a WebSocket route:

```dart
// routes/ws.dart
import 'dart:convert';

import 'package:dart_frog/dart_frog.dart';
import 'package:dart_frog_web_socket/dart_frog_web_socket.dart';

Future<Response> onRequest(RequestContext context) async {
  final handler = webSocketHandler((channel, protocol) {
    print('Client connected');

    channel.stream.listen(
      (message) {
        try {
          final data = jsonDecode(message as String) as Map<String, dynamic>;
          _handleMessage(channel, data);
        } catch (e) {
          channel.sink.add(jsonEncode({
            'type': 'error',
            'message': 'Invalid message format',
          }));
        }
      },
      onDone: () => print('Client disconnected'),
      onError: (error) => print('WebSocket error: $error'),
    );
  });

  return handler(context);
}

void _handleMessage(WebSocketChannel channel, Map<String, dynamic> data) {
  final type = data['type'] as String?;

  switch (type) {
    case 'ping':
      channel.sink.add(jsonEncode({'type': 'pong'}));
    case 'echo':
      channel.sink.add(jsonEncode({
        'type': 'echo',
        'data': data['data'],
      }));
    default:
      channel.sink.add(jsonEncode({
        'type': 'error',
        'message': 'Unknown message type: $type',
      }));
  }
}
```

### WebSocket with Broadcast (Chat Room)

```dart
// lib/src/services/chat_service.dart
import 'dart:async';
import 'dart:convert';

import 'package:web_socket_channel/web_socket_channel.dart';

class ChatService {
  final _clients = <WebSocketChannel>{};

  void addClient(WebSocketChannel channel) {
    _clients.add(channel);
    _broadcast({'type': 'system', 'message': 'User joined'});
  }

  void removeClient(WebSocketChannel channel) {
    _clients.remove(channel);
    _broadcast({'type': 'system', 'message': 'User left'});
  }

  void handleMessage(WebSocketChannel sender, Map<String, dynamic> data) {
    _broadcast({
      'type': 'message',
      'data': data['data'],
      'timestamp': DateTime.now().toIso8601String(),
    });
  }

  void _broadcast(Map<String, dynamic> message) {
    final encoded = jsonEncode(message);
    for (final client in _clients) {
      client.sink.add(encoded);
    }
  }
}
```

## Database Integration

### Repository Pattern

```dart
// lib/src/repositories/user_repository.dart
abstract class UserRepository {
  Future<List<User>> getAll({int page = 1, int limit = 20});
  Future<User?> getById(String id);
  Future<User?> getByEmail(String email);
  Future<User> create(User user);
  Future<User> update(String id, Map<String, dynamic> fields);
  Future<void> delete(String id);
}
```

### PostgreSQL Implementation (using `postgres` package)

```dart
// lib/src/repositories/user_repository_impl.dart
import 'package:postgres/postgres.dart';

class UserRepositoryImpl implements UserRepository {
  final Connection _db;

  UserRepositoryImpl(this._db);

  @override
  Future<List<User>> getAll({int page = 1, int limit = 20}) async {
    final offset = (page - 1) * limit;

    final result = await _db.execute(
      Sql.named(
        'SELECT id, email, name, is_admin, created_at, updated_at '
        'FROM users ORDER BY created_at DESC '
        'LIMIT @limit OFFSET @offset',
      ),
      parameters: {'limit': limit, 'offset': offset},
    );

    return result.map((row) => User(
      id: row[0] as String,
      email: row[1] as String,
      name: row[2] as String,
      isAdmin: row[3] as bool,
      createdAt: row[4] as DateTime,
      updatedAt: row[5] as DateTime?,
    )).toList();
  }

  @override
  Future<User?> getById(String id) async {
    final result = await _db.execute(
      Sql.named(
        'SELECT id, email, name, is_admin, created_at, updated_at '
        'FROM users WHERE id = @id',
      ),
      parameters: {'id': id},
    );

    if (result.isEmpty) return null;

    final row = result.first;
    return User(
      id: row[0] as String,
      email: row[1] as String,
      name: row[2] as String,
      isAdmin: row[3] as bool,
      createdAt: row[4] as DateTime,
      updatedAt: row[5] as DateTime?,
    );
  }

  @override
  Future<User> create(User user) async {
    final result = await _db.execute(
      Sql.named(
        'INSERT INTO users (email, name, password_hash, created_at) '
        'VALUES (@email, @name, @passwordHash, @createdAt) '
        'RETURNING id',
      ),
      parameters: {
        'email': user.email,
        'name': user.name,
        'passwordHash': user.passwordHash,
        'createdAt': DateTime.now().toUtc(),
      },
    );

    return user.copyWith(id: result.first[0] as String);
  }
}
```

### Provider Registration for Database

```dart
// routes/_middleware.dart
Handler middleware(Handler handler) {
  return handler
      .use(provider<Config>((_) => Config.fromEnvironment()))
      .use(provider<Connection>((context) async {
        final config = context.read<Config>();
        return Connection.open(
          Endpoint(
            host: config.dbHost,
            port: config.dbPort,
            database: config.dbName,
            username: config.dbUser,
            password: config.dbPassword,
          ),
          settings: ConnectionSettings(sslMode: SslMode.prefer),
        );
      }))
      .use(provider<UserRepository>((context) {
        return UserRepositoryImpl(context.read<Connection>());
      }));
}
```

## Authentication

### JWT Authentication Provider

```dart
// lib/src/middleware/auth_provider.dart
import 'package:dart_frog/dart_frog.dart';
import 'package:dart_jsonwebtoken/dart_jsonwebtoken.dart';

Middleware authProvider() {
  return provider<User?>((context) {
    final authHeader = context.request.headers['Authorization'];

    if (authHeader == null || !authHeader.startsWith('Bearer ')) {
      return null;
    }

    final token = authHeader.substring(7);

    try {
      final config = context.read<Config>();
      final jwt = JWT.verify(token, SecretKey(config.jwtSecret));
      final payload = jwt.payload as Map<String, dynamic>;
      return User.fromJson(payload['user'] as Map<String, dynamic>);
    } catch (_) {
      return null;
    }
  });
}
```

### Require Authentication Middleware

```dart
// lib/src/middleware/require_auth.dart
import 'dart:io';

import 'package:dart_frog/dart_frog.dart';

Middleware requireAuth() {
  return (handler) {
    return (context) {
      final user = context.read<User?>();

      if (user == null) {
        return Response.json(
          {'error': 'Unauthorized'},
          statusCode: HttpStatus.unauthorized,
        );
      }

      return handler(context);
    };
  };
}
```

### Login Route

```dart
// routes/api/v1/auth/login.dart
import 'dart:io';

import 'package:dart_frog/dart_frog.dart';

import 'package:myapp/myapp.dart';

Future<Response> onRequest(RequestContext context) async {
  if (context.request.method != HttpMethod.post) {
    return Response(statusCode: HttpStatus.methodNotAllowed);
  }

  final userService = context.read<UserService>();
  final body = await context.request.json() as Map<String, dynamic>;

  try {
    final result = await userService.login(
      email: body['email'] as String,
      password: body['password'] as String,
    );

    return Response.json({
      'token': result.token,
      'user': result.user.toPublicJson(),
    });
  } on UnauthorizedException {
    return Response.json(
      {'error': 'Invalid credentials'},
      statusCode: HttpStatus.unauthorized,
    );
  }
}
```

### Register Route

```dart
// routes/api/v1/auth/register.dart
import 'dart:io';

import 'package:dart_frog/dart_frog.dart';

import 'package:myapp/myapp.dart';

Future<Response> onRequest(RequestContext context) async {
  if (context.request.method != HttpMethod.post) {
    return Response(statusCode: HttpStatus.methodNotAllowed);
  }

  final userService = context.read<UserService>();
  final body = await context.request.json() as Map<String, dynamic>;

  try {
    final user = await userService.register(
      email: body['email'] as String,
      password: body['password'] as String,
      name: body['name'] as String,
    );

    return Response.json(
      user.toPublicJson(),
      statusCode: HttpStatus.created,
    );
  } on ValidationException catch (e) {
    return Response.json(
      {'error': e.message, 'errors': e.errors},
      statusCode: HttpStatus.unprocessableEntity,
    );
  }
}
```

### Current User Route

```dart
// routes/api/v1/auth/me.dart
import 'dart:io';

import 'package:dart_frog/dart_frog.dart';

import 'package:myapp/myapp.dart';

Response onRequest(RequestContext context) {
  if (context.request.method != HttpMethod.get) {
    return Response(statusCode: HttpStatus.methodNotAllowed);
  }

  final user = context.read<User?>();

  if (user == null) {
    return Response.json(
      {'error': 'Unauthorized'},
      statusCode: HttpStatus.unauthorized,
    );
  }

  return Response.json(user.toPublicJson());
}
```

## Configuration

### Environment-Based Config

```dart
// lib/src/config.dart
class Config {
  final int port;
  final String databaseUrl;
  final String dbHost;
  final int dbPort;
  final String dbName;
  final String dbUser;
  final String dbPassword;
  final String jwtSecret;
  final bool isDevelopment;

  const Config({
    required this.port,
    required this.databaseUrl,
    required this.dbHost,
    required this.dbPort,
    required this.dbName,
    required this.dbUser,
    required this.dbPassword,
    required this.jwtSecret,
    required this.isDevelopment,
  });

  factory Config.fromEnvironment() {
    final env = Platform.environment;

    return Config(
      port: int.tryParse(env['PORT'] ?? '8080') ?? 8080,
      databaseUrl: env['DATABASE_URL'] ?? '',
      dbHost: env['DB_HOST'] ?? 'localhost',
      dbPort: int.tryParse(env['DB_PORT'] ?? '5432') ?? 5432,
      dbName: env['DB_NAME'] ?? 'myapp',
      dbUser: env['DB_USER'] ?? 'postgres',
      dbPassword: env['DB_PASSWORD'] ?? '',
      jwtSecret: env['JWT_SECRET'] ?? 'change-me-in-production',
      isDevelopment: env['DART_ENV'] != 'production',
    );
  }
}
```

### Middleware Patterns: Request Logger

```dart
// lib/src/middleware/request_logger.dart
Middleware requestLogger() {
  return (handler) {
    return (context) async {
      final stopwatch = Stopwatch()..start();
      final request = context.request;

      print('[${DateTime.now()}] ${request.method.value} ${request.uri}');

      final response = await handler(context);

      stopwatch.stop();
      print(
        '[${DateTime.now()}] ${request.method.value} ${request.uri} '
        '${response.statusCode} ${stopwatch.elapsedMilliseconds}ms',
      );

      return response;
    };
  };
}
```

### Middleware Patterns: CORS

```dart
// lib/src/middleware/cors_middleware.dart
Middleware corsMiddleware({
  List<String> allowedOrigins = const ['*'],
  List<String> allowedMethods = const [
    'GET', 'POST', 'PUT', 'DELETE', 'OPTIONS',
  ],
  List<String> allowedHeaders = const ['Content-Type', 'Authorization'],
}) {
  final corsHeaders = {
    'Access-Control-Allow-Origin': allowedOrigins.join(', '),
    'Access-Control-Allow-Methods': allowedMethods.join(', '),
    'Access-Control-Allow-Headers': allowedHeaders.join(', '),
    'Access-Control-Max-Age': '86400',
  };

  return (handler) {
    return (context) async {
      if (context.request.method == HttpMethod.options) {
        return Response(headers: corsHeaders);
      }

      final response = await handler(context);
      return response.copyWith(
        headers: {...response.headers, ...corsHeaders},
      );
    };
  };
}
```

## Testing Patterns

### Route Handler Test with Mocks

```dart
// test/routes/api/v1/users/index_test.dart
import 'dart:io';

import 'package:dart_frog/dart_frog.dart';
import 'package:mocktail/mocktail.dart';
import 'package:test/test.dart';

import 'package:myapp/myapp.dart';
import '../../../../../routes/api/v1/users/index.dart' as route;

class MockRequestContext extends Mock implements RequestContext {}
class MockUserService extends Mock implements UserService {}

void main() {
  late MockRequestContext context;
  late MockUserService mockUserService;

  setUp(() {
    context = MockRequestContext();
    mockUserService = MockUserService();
    when(() => context.read<UserService>()).thenReturn(mockUserService);
  });

  group('GET /api/v1/users', () {
    test('returns list of users', () async {
      final users = [
        User(
          id: '1', email: 'a@test.com', name: 'A',
          createdAt: DateTime.now(),
        ),
        User(
          id: '2', email: 'b@test.com', name: 'B',
          createdAt: DateTime.now(),
        ),
      ];

      when(() => mockUserService.getAll(
        page: any(named: 'page'),
        limit: any(named: 'limit'),
      )).thenAnswer((_) async => users);

      when(() => context.request).thenReturn(
        Request.get(
          Uri.parse('http://localhost/api/v1/users?page=1&limit=20'),
        ),
      );

      final response = await route.onRequest(context);

      expect(response.statusCode, equals(HttpStatus.ok));

      final body = await response.json() as Map<String, dynamic>;
      expect(body['data'], hasLength(2));
    });
  });

  group('POST /api/v1/users', () {
    test('creates user with valid data', () async {
      final user = User(
        id: '1',
        email: 'test@example.com',
        name: 'Test',
        createdAt: DateTime.now(),
      );

      when(() => mockUserService.create(
        email: any(named: 'email'),
        name: any(named: 'name'),
        password: any(named: 'password'),
      )).thenAnswer((_) async => user);

      when(() => context.request).thenReturn(
        Request.post(
          Uri.parse('http://localhost/api/v1/users'),
          body: '{"email":"test@example.com","name":"Test",'
              '"password":"password123"}',
        ),
      );

      final response = await route.onRequest(context);
      expect(response.statusCode, equals(HttpStatus.created));
    });

    test('returns 422 with validation errors', () async {
      when(() => mockUserService.create(
        email: any(named: 'email'),
        name: any(named: 'name'),
        password: any(named: 'password'),
      )).thenThrow(
        ValidationException('Validation failed', {
          'email': ['Invalid email format'],
        }),
      );

      when(() => context.request).thenReturn(
        Request.post(
          Uri.parse('http://localhost/api/v1/users'),
          body: '{"email":"invalid","name":"Test","password":"password123"}',
        ),
      );

      final response = await route.onRequest(context);
      expect(response.statusCode, equals(HttpStatus.unprocessableEntity));
    });
  });
}
```

### Middleware Test

```dart
// test/middleware/auth_provider_test.dart
import 'package:dart_frog/dart_frog.dart';
import 'package:dart_jsonwebtoken/dart_jsonwebtoken.dart';
import 'package:mocktail/mocktail.dart';
import 'package:test/test.dart';

import 'package:myapp/myapp.dart';

class MockRequestContext extends Mock implements RequestContext {}

void main() {
  group('authProvider', () {
    test('returns null when no Authorization header', () {
      final context = MockRequestContext();
      final request = Request.get(Uri.parse('http://localhost/'));

      when(() => context.request).thenReturn(request);

      // The provider function returns null because header is missing
    });

    test('returns user when valid Bearer token provided', () {
      final context = MockRequestContext();
      final config = Config(
        port: 8080,
        databaseUrl: '',
        dbHost: 'localhost',
        dbPort: 5432,
        dbName: 'test',
        dbUser: 'test',
        dbPassword: '',
        jwtSecret: 'test-secret',
        isDevelopment: true,
      );

      final user = User(
        id: '1',
        email: 'test@test.com',
        name: 'Test',
        createdAt: DateTime.now(),
      );

      final jwt = JWT({'user': user.toJson()});
      final token = jwt.sign(SecretKey(config.jwtSecret));

      final request = Request.get(
        Uri.parse('http://localhost/'),
        headers: {'Authorization': 'Bearer $token'},
      );

      when(() => context.request).thenReturn(request);
      when(() => context.read<Config>()).thenReturn(config);

      // Verify the provider correctly parses the token
    });
  });
}
```

## Deployment

### Dockerfile (Multi-Stage)

```dockerfile
# Build stage
FROM dart:stable AS build

WORKDIR /app

# Install dart_frog CLI
RUN dart pub global activate dart_frog_cli

# Copy dependencies first for layer caching
COPY pubspec.* ./
RUN dart pub get

# Copy source
COPY . .

# Generate code (freezed, json_serializable)
RUN dart run build_runner build --delete-conflicting-outputs

# Build production binary
RUN dart pub global run dart_frog_cli:dart_frog build

# Runtime stage
FROM scratch

COPY --from=build /runtime/ /
COPY --from=build /app/build/bin/server /app/bin/server

EXPOSE 8080

CMD ["/app/bin/server"]
```

### docker-compose.yaml

```yaml
version: '3.8'

services:
  api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - DATABASE_URL=postgres://postgres:password@db:5432/myapp
      - JWT_SECRET=${JWT_SECRET}
      - DART_ENV=production
    depends_on:
      - db

  db:
    image: postgres:15
    environment:
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=myapp
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

volumes:
  postgres_data:
```

### Google Cloud Run

```bash
# Build and push container image
gcloud builds submit --tag gcr.io/PROJECT_ID/myapp

# Deploy to Cloud Run
gcloud run deploy myapp \
  --image gcr.io/PROJECT_ID/myapp \
  --platform managed \
  --allow-unauthenticated \
  --set-env-vars "JWT_SECRET=your-secret"
```

### Fly.io

```toml
# fly.toml
app = "myapp"
primary_region = "iad"

[build]
  dockerfile = "Dockerfile"

[env]
  PORT = "8080"

[http_service]
  internal_port = 8080
  force_https = true
  auto_stop_machines = true
  auto_start_machines = true
  min_machines_running = 0
```

```bash
fly launch
fly deploy
fly secrets set JWT_SECRET=your-secret
```

### pubspec.yaml Template

```yaml
name: myapp
description: Dart Frog backend application
version: 1.0.0
publish_to: none

environment:
  sdk: '>=3.0.0 <4.0.0'

dependencies:
  dart_frog: ^1.0.0
  dart_frog_web_socket: ^1.0.0

  # Data
  freezed_annotation: ^2.4.0
  json_annotation: ^4.8.0

  # Auth
  dart_jsonwebtoken: ^2.12.0
  bcrypt: ^1.1.0

  # Database
  postgres: ^3.0.0

dev_dependencies:
  dart_frog_test: ^1.0.0
  test: ^1.24.0
  mocktail: ^1.0.0
  very_good_analysis: ^5.0.0

  # Code generation
  build_runner: ^2.4.0
  freezed: ^2.4.0
  json_serializable: ^6.7.0
```

### analysis_options.yaml

```yaml
include: package:very_good_analysis/analysis_options.yaml

analyzer:
  exclude:
    - "**/*.g.dart"
    - "**/*.freezed.dart"
  errors:
    invalid_annotation_target: ignore

linter:
  rules:
    public_member_api_docs: false
```
