# Flutter Patterns Reference

## Contents

- [Widget Patterns](#widget-patterns)
- [Animations](#animations)
- [Networking](#networking)
- [Local Storage](#local-storage)
- [Testing](#testing)
- [Platform-Specific Code](#platform-specific-code)
- [Performance](#performance)

## Widget Patterns

### Stateful Widget with Form (LoginForm)

```dart
class LoginForm extends ConsumerStatefulWidget {
  const LoginForm({super.key});

  @override
  ConsumerState<LoginForm> createState() => _LoginFormState();
}

class _LoginFormState extends ConsumerState<LoginForm> {
  final _formKey = GlobalKey<FormState>();
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  bool _obscurePassword = true;

  @override
  void dispose() {
    _emailController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    await ref.read(loginProvider.notifier).login(
      _emailController.text.trim(),
      _passwordController.text,
    );
  }

  @override
  Widget build(BuildContext context) {
    final loginState = ref.watch(loginProvider);

    ref.listen(loginProvider, (previous, next) {
      next.whenOrNull(
        error: (error, _) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(error.toString())),
          );
        },
      );
    });

    return Form(
      key: _formKey,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          TextFormField(
            controller: _emailController,
            decoration: const InputDecoration(
              labelText: 'Email',
              prefixIcon: Icon(Icons.email_outlined),
            ),
            keyboardType: TextInputType.emailAddress,
            textInputAction: TextInputAction.next,
            validator: Validators.email,
            enabled: !loginState.isLoading,
          ),
          const SizedBox(height: 16),
          TextFormField(
            controller: _passwordController,
            decoration: InputDecoration(
              labelText: 'Password',
              prefixIcon: const Icon(Icons.lock_outline),
              suffixIcon: IconButton(
                icon: Icon(
                  _obscurePassword
                      ? Icons.visibility_outlined
                      : Icons.visibility_off_outlined,
                ),
                onPressed: () {
                  setState(() => _obscurePassword = !_obscurePassword);
                },
              ),
            ),
            obscureText: _obscurePassword,
            textInputAction: TextInputAction.done,
            validator: Validators.password,
            enabled: !loginState.isLoading,
            onFieldSubmitted: (_) => _submit(),
          ),
          const SizedBox(height: 24),
          ElevatedButton(
            onPressed: loginState.isLoading ? null : _submit,
            child: loginState.isLoading
                ? const SizedBox(
                    height: 20,
                    width: 20,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Text('Sign In'),
          ),
        ],
      ),
    );
  }
}
```

### Reusable List Tile Widget

```dart
class UserListTile extends StatelessWidget {
  final User user;
  const UserListTile({super.key, required this.user});

  @override
  Widget build(BuildContext context) {
    return ListTile(
      leading: CircleAvatar(
        backgroundImage: user.avatarUrl != null
            ? NetworkImage(user.avatarUrl!)
            : null,
        child: user.avatarUrl == null
            ? Text(user.name[0].toUpperCase())
            : null,
      ),
      title: Text(user.name),
      subtitle: Text(user.email),
      trailing: const Icon(Icons.chevron_right),
      onTap: () => context.pushNamed(
        'user-detail',
        pathParameters: {'id': user.id},
      ),
    );
  }
}
```

### Bottom Navigation Scaffold (ShellRoute)

```dart
class ScaffoldWithNavBar extends StatelessWidget {
  final Widget child;
  const ScaffoldWithNavBar({super.key, required this.child});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: child,
      bottomNavigationBar: NavigationBar(
        selectedIndex: _calculateSelectedIndex(context),
        onDestinationSelected: (index) => _onItemTapped(index, context),
        destinations: const [
          NavigationDestination(
            icon: Icon(Icons.home_outlined),
            selectedIcon: Icon(Icons.home),
            label: 'Home',
          ),
          NavigationDestination(
            icon: Icon(Icons.person_outline),
            selectedIcon: Icon(Icons.person),
            label: 'Profile',
          ),
        ],
      ),
    );
  }

  int _calculateSelectedIndex(BuildContext context) {
    final location = GoRouterState.of(context).matchedLocation;
    if (location.startsWith('/profile')) return 1;
    return 0;
  }

  void _onItemTapped(int index, BuildContext context) {
    switch (index) {
      case 0: context.goNamed('home');
      case 1: context.goNamed('profile');
    }
  }
}
```

### Responsive Layout Builder

```dart
class ResponsiveLayout extends StatelessWidget {
  final Widget mobile;
  final Widget? tablet;
  final Widget? desktop;

  const ResponsiveLayout({
    super.key,
    required this.mobile,
    this.tablet,
    this.desktop,
  });

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        if (constraints.maxWidth >= 1200 && desktop != null) {
          return desktop!;
        }
        if (constraints.maxWidth >= 600 && tablet != null) {
          return tablet!;
        }
        return mobile;
      },
    );
  }
}
```

### Sliver-Based Scrollable Screen

```dart
class ProfileScreen extends ConsumerWidget {
  const ProfileScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final user = ref.watch(currentUserProvider);

    return Scaffold(
      body: CustomScrollView(
        slivers: [
          SliverAppBar.large(
            title: Text(user?.name ?? 'Profile'),
          ),
          SliverToBoxAdapter(
            child: ProfileHeader(user: user),
          ),
          SliverPadding(
            padding: const EdgeInsets.all(16),
            sliver: SliverList.builder(
              itemCount: menuItems.length,
              itemBuilder: (context, index) => MenuTile(item: menuItems[index]),
            ),
          ),
        ],
      ),
    );
  }
}
```

## Animations

### Implicit Animations

Use implicit animations for simple property changes:

```dart
// AnimatedContainer: animate size, color, padding, etc.
AnimatedContainer(
  duration: const Duration(milliseconds: 300),
  curve: Curves.easeInOut,
  width: isExpanded ? 200 : 100,
  height: isExpanded ? 200 : 100,
  decoration: BoxDecoration(
    color: isSelected ? Colors.blue : Colors.grey,
    borderRadius: BorderRadius.circular(isExpanded ? 16 : 8),
  ),
  child: child,
)

// AnimatedSwitcher: animate widget transitions
AnimatedSwitcher(
  duration: const Duration(milliseconds: 300),
  transitionBuilder: (child, animation) => FadeTransition(
    opacity: animation,
    child: child,
  ),
  child: isLoading
      ? const CircularProgressIndicator(key: ValueKey('loading'))
      : const Icon(Icons.check, key: ValueKey('done')),
)
```

### Explicit Animations with AnimationController

```dart
class PulseWidget extends StatefulWidget {
  final Widget child;
  const PulseWidget({super.key, required this.child});

  @override
  State<PulseWidget> createState() => _PulseWidgetState();
}

class _PulseWidgetState extends State<PulseWidget>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;
  late final Animation<double> _animation;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 1000),
    )..repeat(reverse: true);
    _animation = Tween<double>(begin: 0.8, end: 1.0).animate(
      CurvedAnimation(parent: _controller, curve: Curves.easeInOut),
    );
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return ScaleTransition(
      scale: _animation,
      child: widget.child,
    );
  }
}
```

### Hero Transitions

```dart
// Source screen
Hero(
  tag: 'user-avatar-${user.id}',
  child: CircleAvatar(backgroundImage: NetworkImage(user.avatarUrl!)),
)

// Destination screen
Hero(
  tag: 'user-avatar-${user.id}',
  child: CircleAvatar(
    radius: 60,
    backgroundImage: NetworkImage(user.avatarUrl!),
  ),
)
```

### Staggered List Animation

```dart
class StaggeredList extends StatefulWidget {
  final List<Widget> children;
  const StaggeredList({super.key, required this.children});

  @override
  State<StaggeredList> createState() => _StaggeredListState();
}

class _StaggeredListState extends State<StaggeredList>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: Duration(milliseconds: 200 * widget.children.length),
    )..forward();
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      children: List.generate(widget.children.length, (index) {
        final start = index / widget.children.length;
        final end = (index + 1) / widget.children.length;
        final animation = Tween<Offset>(
          begin: const Offset(0, 0.3),
          end: Offset.zero,
        ).animate(CurvedAnimation(
          parent: _controller,
          curve: Interval(start, end, curve: Curves.easeOut),
        ));

        return SlideTransition(
          position: animation,
          child: FadeTransition(
            opacity: CurvedAnimation(
              parent: _controller,
              curve: Interval(start, end),
            ),
            child: widget.children[index],
          ),
        );
      }),
    );
  }
}
```

## Networking

### Repository Implementation with Error Handling

```dart
class AuthRepositoryImpl implements AuthRepository {
  final AuthRemoteDataSource remoteDataSource;
  final AuthLocalDataSource localDataSource;

  AuthRepositoryImpl({
    required this.remoteDataSource,
    required this.localDataSource,
  });

  @override
  Stream<User?> get authStateChanges {
    return remoteDataSource.authStateChanges
        .map((model) => model?.toEntity());
  }

  @override
  Future<Either<Failure, User>> signInWithEmailAndPassword(
    String email,
    String password,
  ) async {
    try {
      final userModel = await remoteDataSource
          .signInWithEmailAndPassword(email, password);
      await localDataSource.cacheUser(userModel);
      return Right(userModel.toEntity());
    } on AuthException catch (e) {
      return Left(AuthFailure(e.message));
    } catch (e) {
      return Left(ServerFailure(e.toString()));
    }
  }

  @override
  Future<Either<Failure, void>> signOut() async {
    try {
      await remoteDataSource.signOut();
      await localDataSource.clearCache();
      return const Right(null);
    } catch (e) {
      return Left(ServerFailure(e.toString()));
    }
  }
}
```

### API Client with Interceptors

```dart
class AuthInterceptor extends Interceptor {
  final Ref ref;
  AuthInterceptor(this.ref);

  @override
  void onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    final token = await ref.read(tokenProvider.future);
    if (token != null) {
      options.headers['Authorization'] = 'Bearer $token';
    }
    handler.next(options);
  }

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    if (err.response?.statusCode == 401) {
      ref.read(authProvider.notifier).logout();
    }
    handler.next(err);
  }
}

class ApiClient {
  final Dio _dio;
  ApiClient(this._dio);

  Future<T> get<T>(
    String path, {
    Map<String, dynamic>? queryParameters,
    required T Function(dynamic) parser,
  }) async {
    try {
      final response = await _dio.get(path, queryParameters: queryParameters);
      return parser(response.data);
    } on DioException catch (e) {
      throw _handleError(e);
    }
  }

  Future<T> post<T>(
    String path, {
    dynamic data,
    required T Function(dynamic) parser,
  }) async {
    try {
      final response = await _dio.post(path, data: data);
      return parser(response.data);
    } on DioException catch (e) {
      throw _handleError(e);
    }
  }

  AppException _handleError(DioException error) {
    switch (error.type) {
      case DioExceptionType.connectionTimeout:
      case DioExceptionType.receiveTimeout:
      case DioExceptionType.sendTimeout:
        return NetworkException('Connection timeout');
      case DioExceptionType.badResponse:
        final statusCode = error.response?.statusCode;
        final message = error.response?.data?['message'] ?? 'Unknown error';
        return switch (statusCode) {
          401 => UnauthorizedException(message),
          404 => NotFoundException(message),
          422 => ValidationException(message),
          _   => ServerException(message),
        };
      default:
        return NetworkException('Network error');
    }
  }
}
```

### Pagination Provider

```dart
final paginatedUsersProvider = AsyncNotifierProvider<
    PaginatedUsersNotifier, PaginatedData<User>>(
  PaginatedUsersNotifier.new,
);

class PaginatedUsersNotifier extends AsyncNotifier<PaginatedData<User>> {
  static const _pageSize = 20;

  @override
  Future<PaginatedData<User>> build() async {
    return _fetchPage(0);
  }

  Future<PaginatedData<User>> _fetchPage(int page) async {
    final repo = ref.read(userRepositoryProvider);
    final users = await repo.getUsers(page: page, limit: _pageSize);
    return PaginatedData(
      items: users,
      currentPage: page,
      hasMore: users.length == _pageSize,
    );
  }

  Future<void> loadNextPage() async {
    final current = state.valueOrNull;
    if (current == null || !current.hasMore) return;

    final nextPage = await _fetchPage(current.currentPage + 1);
    state = AsyncData(PaginatedData(
      items: [...current.items, ...nextPage.items],
      currentPage: nextPage.currentPage,
      hasMore: nextPage.hasMore,
    ));
  }
}

class PaginatedData<T> {
  final List<T> items;
  final int currentPage;
  final bool hasMore;
  const PaginatedData({
    required this.items,
    required this.currentPage,
    required this.hasMore,
  });
}
```

## Local Storage

### Shared Preferences Wrapper

```dart
final sharedPreferencesProvider = Provider<SharedPreferences>((ref) {
  throw UnimplementedError('Override in main with actual instance');
});

// Initialize in main.dart:
// final prefs = await SharedPreferences.getInstance();
// ProviderScope(overrides: [sharedPreferencesProvider.overrideWithValue(prefs)])

final themeProvider = StateNotifierProvider<ThemeNotifier, ThemeMode>((ref) {
  final prefs = ref.watch(sharedPreferencesProvider);
  return ThemeNotifier(prefs);
});

class ThemeNotifier extends StateNotifier<ThemeMode> {
  final SharedPreferences _prefs;

  ThemeNotifier(this._prefs) : super(_loadTheme(_prefs));

  static ThemeMode _loadTheme(SharedPreferences prefs) {
    final value = prefs.getString('theme_mode');
    return ThemeMode.values.firstWhere(
      (e) => e.name == value,
      orElse: () => ThemeMode.system,
    );
  }

  Future<void> setTheme(ThemeMode mode) async {
    await _prefs.setString('theme_mode', mode.name);
    state = mode;
  }
}
```

### Secure Storage for Credentials

```dart
final secureStorageProvider = Provider<FlutterSecureStorage>((ref) {
  return const FlutterSecureStorage(
    aOptions: AndroidOptions(encryptedSharedPreferences: true),
  );
});

final tokenProvider = FutureProvider<String?>((ref) async {
  final storage = ref.watch(secureStorageProvider);
  return storage.read(key: 'auth_token');
});

class AuthLocalDataSource {
  final FlutterSecureStorage _storage;
  AuthLocalDataSource(this._storage);

  Future<void> saveToken(String token) async {
    await _storage.write(key: 'auth_token', value: token);
  }

  Future<String?> getToken() async {
    return _storage.read(key: 'auth_token');
  }

  Future<void> clearCredentials() async {
    await _storage.deleteAll();
  }
}
```

## Testing

### Full Widget Test with Mocking

```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

class MockAuthRepository extends Mock implements AuthRepository {}

void main() {
  late MockAuthRepository mockRepository;

  setUp(() {
    mockRepository = MockAuthRepository();
  });

  Widget createWidget() {
    return ProviderScope(
      overrides: [
        authRepositoryProvider.overrideWithValue(mockRepository),
      ],
      child: const MaterialApp(
        home: Scaffold(body: LoginForm()),
      ),
    );
  }

  group('LoginForm', () {
    testWidgets('renders email and password fields', (tester) async {
      await tester.pumpWidget(createWidget());
      expect(find.byType(TextFormField), findsNWidgets(2));
      expect(find.text('Email'), findsOneWidget);
      expect(find.text('Password'), findsOneWidget);
    });

    testWidgets('shows validation errors for empty fields', (tester) async {
      await tester.pumpWidget(createWidget());
      await tester.tap(find.text('Sign In'));
      await tester.pump();
      expect(find.text('Email is required'), findsOneWidget);
      expect(find.text('Password is required'), findsOneWidget);
    });

    testWidgets('calls login with valid credentials', (tester) async {
      when(() => mockRepository.signInWithEmailAndPassword(any(), any()))
          .thenAnswer((_) async => Right(testUser));

      await tester.pumpWidget(createWidget());
      await tester.enterText(
        find.widgetWithText(TextFormField, 'Email'),
        'test@example.com',
      );
      await tester.enterText(
        find.widgetWithText(TextFormField, 'Password'),
        'password123',
      );
      await tester.tap(find.text('Sign In'));
      await tester.pump();

      verify(() => mockRepository.signInWithEmailAndPassword(
            'test@example.com',
            'password123',
          )).called(1);
    });

    testWidgets('shows loading indicator during login', (tester) async {
      when(() => mockRepository.signInWithEmailAndPassword(any(), any()))
          .thenAnswer((_) async {
        await Future.delayed(const Duration(seconds: 1));
        return Right(testUser);
      });

      await tester.pumpWidget(createWidget());
      await tester.enterText(
        find.widgetWithText(TextFormField, 'Email'),
        'test@example.com',
      );
      await tester.enterText(
        find.widgetWithText(TextFormField, 'Password'),
        'password123',
      );
      await tester.tap(find.text('Sign In'));
      await tester.pump();

      expect(find.byType(CircularProgressIndicator), findsOneWidget);
    });
  });
}
```

### Provider Unit Test

```dart
import 'package:dartz/dartz.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

class MockAuthRepository extends Mock implements AuthRepository {}

void main() {
  late MockAuthRepository mockRepository;
  late ProviderContainer container;

  final testUser = User(
    id: '1',
    email: 'test@example.com',
    name: 'Test User',
    createdAt: DateTime.now(),
  );

  setUp(() {
    mockRepository = MockAuthRepository();
    container = ProviderContainer(
      overrides: [
        authRepositoryProvider.overrideWithValue(mockRepository),
      ],
    );
  });

  tearDown(() {
    container.dispose();
  });

  group('LoginNotifier', () {
    test('login success updates state', () async {
      when(() => mockRepository.signInWithEmailAndPassword(any(), any()))
          .thenAnswer((_) async => Right(testUser));

      final notifier = container.read(loginProvider.notifier);
      await notifier.login('test@example.com', 'password');

      final state = container.read(loginProvider);
      expect(state.hasError, false);
      expect(state.isLoading, false);
    });

    test('login failure sets error state', () async {
      when(() => mockRepository.signInWithEmailAndPassword(any(), any()))
          .thenAnswer(
            (_) async => Left(AuthFailure('Invalid credentials')),
          );

      final notifier = container.read(loginProvider.notifier);
      await notifier.login('test@example.com', 'wrong');

      final state = container.read(loginProvider);
      expect(state.hasError, true);
    });
  });
}
```

### Integration Test

```dart
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:myapp/main.dart' as app;

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  group('end-to-end test', () {
    testWidgets('login flow', (tester) async {
      app.main();
      await tester.pumpAndSettle();

      // Should be on login screen
      expect(find.text('Sign In'), findsOneWidget);

      // Enter credentials
      await tester.enterText(
        find.byKey(const Key('email_field')),
        'test@example.com',
      );
      await tester.enterText(
        find.byKey(const Key('password_field')),
        'password123',
      );

      // Submit
      await tester.tap(find.text('Sign In'));
      await tester.pumpAndSettle();

      // Should be on home screen
      expect(find.text('Home'), findsOneWidget);
    });
  });
}
```

### Golden Tests

```dart
testWidgets('UserCard matches golden', (tester) async {
  await tester.pumpWidget(
    MaterialApp(
      theme: AppTheme.light,
      home: Scaffold(
        body: UserCard(user: testUser),
      ),
    ),
  );

  await expectLater(
    find.byType(UserCard),
    matchesGoldenFile('goldens/user_card.png'),
  );
});

// Update goldens: flutter test --update-goldens
```

## Platform-Specific Code

### Conditional Imports

```dart
// platform_utils.dart (barrel file)
export 'platform_utils_stub.dart'
    if (dart.library.io) 'platform_utils_io.dart'
    if (dart.library.html) 'platform_utils_web.dart';

// platform_utils_stub.dart
String getPlatformName() => 'unknown';

// platform_utils_io.dart
import 'dart:io';
String getPlatformName() => Platform.operatingSystem;

// platform_utils_web.dart
String getPlatformName() => 'web';
```

### Platform-Aware Widgets

```dart
class AdaptiveButton extends StatelessWidget {
  final String label;
  final VoidCallback onPressed;

  const AdaptiveButton({
    super.key,
    required this.label,
    required this.onPressed,
  });

  @override
  Widget build(BuildContext context) {
    final platform = Theme.of(context).platform;

    if (platform == TargetPlatform.iOS ||
        platform == TargetPlatform.macOS) {
      return CupertinoButton(
        onPressed: onPressed,
        child: Text(label),
      );
    }

    return ElevatedButton(
      onPressed: onPressed,
      child: Text(label),
    );
  }
}
```

### Platform Channel Error Handling

```dart
Future<T?> invokePlatformMethod<T>(
  MethodChannel channel,
  String method, [
  dynamic arguments,
]) async {
  try {
    return await channel.invokeMethod<T>(method, arguments);
  } on PlatformException catch (e) {
    debugPrint('Platform error [$method]: ${e.message}');
    return null;
  } on MissingPluginException {
    debugPrint('Plugin not available for $method on this platform');
    return null;
  }
}
```

## Performance

### Const Widgets

```dart
// GOOD: compile-time constant, never rebuilt
const SizedBox(height: 16)
const Text('Static label')
const Icon(Icons.home)
const EdgeInsets.all(16)

// BAD: new instance every build
SizedBox(height: 16)
```

### ListView Optimization

```dart
// BAD: builds all items eagerly
ListView(
  children: items.map((item) => ItemWidget(item: item)).toList(),
)

// GOOD: builds items lazily as they scroll into view
ListView.builder(
  itemCount: items.length,
  itemBuilder: (context, index) => ItemWidget(item: items[index]),
)

// GOOD: for fixed-size items, even better performance
ListView.builder(
  itemCount: items.length,
  itemExtent: 72, // fixed height enables optimized scrolling
  itemBuilder: (context, index) => ItemWidget(item: items[index]),
)
```

### Image Caching and Optimization

```dart
// Use CachedNetworkImage for network images
CachedNetworkImage(
  imageUrl: user.avatarUrl,
  placeholder: (_, __) => const CircularProgressIndicator(),
  errorWidget: (_, __, ___) => const Icon(Icons.error),
  memCacheWidth: 200, // decode at required size, not full resolution
)

// Precache images that will be shown immediately
@override
void didChangeDependencies() {
  super.didChangeDependencies();
  precacheImage(const AssetImage('assets/logo.png'), context);
}
```

### Avoiding Unnecessary Rebuilds

```dart
// BAD: entire widget tree rebuilds when count changes
class ParentWidget extends ConsumerWidget {
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final count = ref.watch(counterProvider);
    return Column(
      children: [
        Text('Count: $count'),
        const ExpensiveWidget(), // rebuilds unnecessarily
      ],
    );
  }
}

// GOOD: only CountDisplay rebuilds
class ParentWidget extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        const CountDisplay(),     // reads provider internally
        const ExpensiveWidget(),  // never rebuilds from count changes
      ],
    );
  }
}

class CountDisplay extends ConsumerWidget {
  const CountDisplay({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final count = ref.watch(counterProvider);
    return Text('Count: $count');
  }
}
```

### RepaintBoundary for Complex Widgets

```dart
// Isolate expensive painting operations
RepaintBoundary(
  child: CustomPaint(
    painter: ChartPainter(data: chartData),
    size: const Size(300, 200),
  ),
)
```

### Compute-Heavy Operations in Isolates

```dart
// Move JSON parsing off the main isolate for large payloads
final usersProvider = FutureProvider<List<User>>((ref) async {
  final response = await ref.read(dioProvider).get('/users');
  // Parse in a separate isolate to avoid janking the UI
  return Isolate.run(() {
    final list = response.data as List;
    return list.map((json) => User.fromJson(json)).toList();
  });
});
```

### Analysis Options for Performance

```yaml
# analysis_options.yaml
include: package:flutter_lints/flutter.yaml

analyzer:
  language:
    strict-casts: true
    strict-inference: true
    strict-raw-types: true
  exclude:
    - "**/*.g.dart"
    - "**/*.freezed.dart"

linter:
  rules:
    - always_declare_return_types
    - avoid_dynamic_calls
    - avoid_print
    - prefer_const_constructors
    - prefer_const_declarations
    - prefer_final_fields
    - prefer_final_locals
    - require_trailing_commas
    - unawaited_futures
    - use_key_in_widget_constructors
```
