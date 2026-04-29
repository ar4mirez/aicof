# Dart Patterns Reference

## Contents

- [Stream Patterns](#stream-patterns)
- [Isolate Communication](#isolate-communication)
- [Sealed Class Hierarchies](#sealed-class-hierarchies)
- [Repository Pattern](#repository-pattern)

## Stream Patterns

### StreamController for Event Bus

```dart
class EventBus {
  final _controller = StreamController<AppEvent>.broadcast();

  Stream<T> on<T extends AppEvent>() =>
      _controller.stream.whereType<T>();

  void emit(AppEvent event) => _controller.add(event);

  void dispose() => _controller.close();
}

// Usage
final bus = EventBus();
bus.on<UserLoggedIn>().listen((event) {
  print('Welcome ${event.user.name}');
});
bus.emit(UserLoggedIn(user));
```

### Stream Transformers

```dart
// Custom debounce transformer
StreamTransformer<T, T> debounce<T>(Duration duration) {
  Timer? timer;
  return StreamTransformer.fromHandlers(
    handleData: (data, sink) {
      timer?.cancel();
      timer = Timer(duration, () => sink.add(data));
    },
    handleDone: (sink) {
      timer?.cancel();
      sink.close();
    },
  );
}

// Combine multiple streams into one
Stream<({String query, List<Filter> filters})> searchParams(
  Stream<String> queryStream,
  Stream<List<Filter>> filterStream,
) async* {
  var query = '';
  var filters = <Filter>[];

  await for (final event in StreamGroup.merge([
    queryStream.map((q) => (type: 'query', value: q)),
    filterStream.map((f) => (type: 'filter', value: f)),
  ])) {
    if (event.type == 'query') query = event.value as String;
    if (event.type == 'filter') filters = event.value as List<Filter>;
    yield (query: query, filters: filters);
  }
}
```

### Paginated Stream

```dart
Stream<List<Item>> paginatedFetch({
  required Future<List<Item>> Function(int page) fetcher,
  int startPage = 0,
}) async* {
  var page = startPage;
  while (true) {
    final items = await fetcher(page);
    if (items.isEmpty) return;
    yield items;
    page++;
  }
}

// Usage
await for (final batch in paginatedFetch(fetcher: api.getItems)) {
  allItems.addAll(batch);
  if (allItems.length >= maxItems) break;
}
```

## Isolate Communication

### Long-Running Isolate with Bidirectional Messages

```dart
class BackgroundProcessor {
  late final Isolate _isolate;
  late final SendPort _sendPort;
  final _receivePort = ReceivePort();

  Future<void> start() async {
    _isolate = await Isolate.spawn(
      _isolateEntry,
      _receivePort.sendPort,
    );
    _sendPort = await _receivePort.first as SendPort;
  }

  Future<ProcessResult> process(ProcessRequest request) async {
    final response = ReceivePort();
    _sendPort.send((request, response.sendPort));
    final result = await response.first as ProcessResult;
    response.close();
    return result;
  }

  void dispose() {
    _isolate.kill();
    _receivePort.close();
  }

  static void _isolateEntry(SendPort mainSendPort) {
    final port = ReceivePort();
    mainSendPort.send(port.sendPort);

    port.listen((message) {
      final (request, replyPort) = message as (ProcessRequest, SendPort);
      final result = _heavyComputation(request);
      replyPort.send(result);
    });
  }

  static ProcessResult _heavyComputation(ProcessRequest req) {
    // CPU-intensive work runs here without blocking main isolate
    return ProcessResult(data: req.input.toUpperCase());
  }
}
```

### Isolate Pool for Parallel Work

```dart
Future<List<R>> parallelMap<T, R>(
  List<T> items,
  R Function(T) transform, {
  int concurrency = 4,
}) async {
  final results = List<R?>.filled(items.length, null);
  final chunks = _partition(items, concurrency);

  await Future.wait(
    chunks.indexed.map((entry) async {
      final (chunkIndex, chunk) = entry;
      final chunkResults = await Isolate.run(
        () => chunk.map(transform).toList(),
      );
      for (var i = 0; i < chunkResults.length; i++) {
        results[chunkIndex * (items.length ~/ concurrency) + i] =
            chunkResults[i];
      }
    }),
  );

  return results.cast<R>();
}

List<List<T>> _partition<T>(List<T> list, int parts) {
  final size = (list.length / parts).ceil();
  return [
    for (var i = 0; i < list.length; i += size)
      list.sublist(i, (i + size).clamp(0, list.length)),
  ];
}
```

## Sealed Class Hierarchies

### Domain Error Modeling

```dart
sealed class AppError {
  const AppError();
  String get userMessage;
}

final class NotFound extends AppError {
  final String resource;
  final String id;
  const NotFound(this.resource, this.id);

  @override
  String get userMessage => '$resource not found';
}

final class ValidationError extends AppError {
  final Map<String, List<String>> fieldErrors;
  const ValidationError(this.fieldErrors);

  @override
  String get userMessage => 'Please fix the highlighted fields';
}

final class NetworkError extends AppError {
  final int? statusCode;
  final String detail;
  const NetworkError(this.detail, [this.statusCode]);

  @override
  String get userMessage => 'Connection problem. Please try again.';
}

// Exhaustive handling guaranteed by compiler
int toHttpStatus(AppError error) => switch (error) {
  NotFound()        => 404,
  ValidationError() => 422,
  NetworkError(:final statusCode) => statusCode ?? 502,
};
```

### State Machine with Sealed Classes

```dart
sealed class AuthState {
  const AuthState();
}

final class Unauthenticated extends AuthState {
  const Unauthenticated();
}

final class Authenticating extends AuthState {
  final String email;
  const Authenticating(this.email);
}

final class Authenticated extends AuthState {
  final User user;
  final DateTime expiresAt;
  const Authenticated(this.user, this.expiresAt);

  bool get isExpired => DateTime.now().isAfter(expiresAt);
}

final class AuthFailed extends AuthState {
  final String reason;
  final int attempts;
  const AuthFailed(this.reason, this.attempts);

  bool get isLocked => attempts >= 5;
}

// State transitions are explicit and type-safe
AuthState handleLogin(AuthState current, LoginEvent event) =>
    switch ((current, event)) {
      (Unauthenticated(), LoginRequested(:final email)) =>
        Authenticating(email),
      (Authenticating(:final email), LoginSucceeded(:final user, :final token)) =>
        Authenticated(user, token.expiresAt),
      (Authenticating(), LoginFailed(:final reason)) =>
        AuthFailed(reason, 1),
      (AuthFailed(:final attempts), LoginFailed(:final reason)) =>
        AuthFailed(reason, attempts + 1),
      (AuthFailed(isLocked: true), LoginRequested()) =>
        current, // remain locked
      (AuthFailed(), LoginRequested(:final email)) =>
        Authenticating(email),
      (Authenticated(), LogoutRequested()) =>
        const Unauthenticated(),
      _ => current,
    };
```

## Repository Pattern

```dart
abstract interface class Repository<T, ID> {
  Future<T?> findById(ID id);
  Future<List<T>> findAll({int offset = 0, int limit = 20});
  Future<T> save(T entity);
  Future<void> delete(ID id);
}

class UserRepository implements Repository<User, UserId> {
  final Database _db;

  UserRepository(this._db);

  @override
  Future<User?> findById(UserId id) async {
    final row = await _db.query(
      'SELECT * FROM users WHERE id = ?',
      [id.value],
    );
    return row.isEmpty ? null : User.fromRow(row.first);
  }

  @override
  Future<List<User>> findAll({int offset = 0, int limit = 20}) async {
    final rows = await _db.query(
      'SELECT * FROM users ORDER BY created_at DESC LIMIT ? OFFSET ?',
      [limit, offset],
    );
    return rows.map(User.fromRow).toList();
  }

  @override
  Future<User> save(User user) async {
    await _db.execute(
      'INSERT INTO users (id, name, email) VALUES (?, ?, ?) '
      'ON CONFLICT (id) DO UPDATE SET name = ?, email = ?',
      [user.id.value, user.name, user.email, user.name, user.email],
    );
    return user;
  }

  @override
  Future<void> delete(UserId id) async {
    await _db.execute('DELETE FROM users WHERE id = ?', [id.value]);
  }
}
```
