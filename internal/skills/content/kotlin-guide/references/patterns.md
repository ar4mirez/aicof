# Kotlin Patterns Reference

## Contents

- [Coroutine Patterns](#coroutine-patterns)
- [Sealed Class Hierarchies](#sealed-class-hierarchies)
- [DSL Builders](#dsl-builders)
- [Flow Patterns](#flow-patterns)

## Coroutine Patterns

### Retry with Exponential Backoff

```kotlin
suspend fun <T> retryWithBackoff(
    maxRetries: Int = 3,
    initialDelayMs: Long = 100,
    maxDelayMs: Long = 5_000,
    factor: Double = 2.0,
    block: suspend () -> T,
): T {
    var currentDelay = initialDelayMs
    repeat(maxRetries - 1) { attempt ->
        try {
            return block()
        } catch (e: CancellationException) {
            throw e
        } catch (e: Exception) {
            logger.warn("Attempt ${attempt + 1} failed: ${e.message}")
        }
        delay(currentDelay)
        currentDelay = (currentDelay * factor).toLong().coerceAtMost(maxDelayMs)
    }
    return block() // final attempt, let exception propagate
}
```

### Fan-Out / Fan-In with Bounded Concurrency

```kotlin
suspend fun <T, R> List<T>.mapParallel(
    concurrency: Int = 10,
    transform: suspend (T) -> R,
): List<R> = coroutineScope {
    val semaphore = Semaphore(concurrency)
    map { item ->
        async {
            semaphore.withPermit { transform(item) }
        }
    }.awaitAll()
}

// Usage
val results = orders.mapParallel(concurrency = 5) { order ->
    paymentService.process(order)
}
```

### Supervisor Pattern for Partial Failure

```kotlin
suspend fun processAllOrders(orders: List<Order>): List<Result<Receipt>> =
    supervisorScope {
        orders.map { order ->
            async {
                try {
                    Result.Success(paymentService.process(order))
                } catch (e: CancellationException) {
                    throw e // always rethrow cancellation
                } catch (e: Exception) {
                    Result.Failure(AppError.Validation("order", e.message ?: ""))
                }
            }
        }.awaitAll()
    }
```

## Sealed Class Hierarchies

### Domain Event System

```kotlin
sealed interface DomainEvent {
    val occurredAt: Instant
    val aggregateId: String
}

sealed interface UserEvent : DomainEvent {
    val userId: UserId
    override val aggregateId: String get() = userId.value

    data class Created(
        override val userId: UserId,
        val email: Email,
        override val occurredAt: Instant = Instant.now(),
    ) : UserEvent

    data class EmailChanged(
        override val userId: UserId,
        val oldEmail: Email,
        val newEmail: Email,
        override val occurredAt: Instant = Instant.now(),
    ) : UserEvent

    data class Deactivated(
        override val userId: UserId,
        val reason: String,
        override val occurredAt: Instant = Instant.now(),
    ) : UserEvent
}

// Exhaustive handling -- compiler enforces all branches
fun handleUserEvent(event: UserEvent) = when (event) {
    is UserEvent.Created -> sendWelcomeEmail(event.email)
    is UserEvent.EmailChanged -> sendVerificationEmail(event.newEmail)
    is UserEvent.Deactivated -> notifyAdmins(event.userId, event.reason)
}
```

### State Machine

```kotlin
sealed class OrderState {
    abstract val orderId: OrderId

    data class Draft(override val orderId: OrderId, val items: List<LineItem>) : OrderState()
    data class Submitted(override val orderId: OrderId, val submittedAt: Instant) : OrderState()
    data class Paid(override val orderId: OrderId, val paymentId: PaymentId) : OrderState()
    data class Shipped(override val orderId: OrderId, val trackingCode: String) : OrderState()
    data class Cancelled(override val orderId: OrderId, val reason: String) : OrderState()
}

fun OrderState.transition(action: OrderAction): OrderState = when (this) {
    is OrderState.Draft -> when (action) {
        is OrderAction.Submit -> OrderState.Submitted(orderId, Instant.now())
        is OrderAction.Cancel -> OrderState.Cancelled(orderId, action.reason)
        else -> throw IllegalStateException("Cannot $action from Draft")
    }
    is OrderState.Submitted -> when (action) {
        is OrderAction.Pay -> OrderState.Paid(orderId, action.paymentId)
        is OrderAction.Cancel -> OrderState.Cancelled(orderId, action.reason)
        else -> throw IllegalStateException("Cannot $action from Submitted")
    }
    is OrderState.Paid -> when (action) {
        is OrderAction.Ship -> OrderState.Shipped(orderId, action.trackingCode)
        else -> throw IllegalStateException("Cannot $action from Paid")
    }
    is OrderState.Shipped, is OrderState.Cancelled ->
        throw IllegalStateException("Order in terminal state")
}
```

## DSL Builders

### Type-Safe Configuration DSL

```kotlin
@DslMarker
annotation class ConfigDsl

@ConfigDsl
class ServerConfig {
    var host: String = "0.0.0.0"
    var port: Int = 8080
    private var _database: DatabaseConfig? = null

    val database: DatabaseConfig get() = requireNotNull(_database) { "Database not configured" }

    fun database(block: DatabaseConfig.() -> Unit) {
        _database = DatabaseConfig().apply(block)
    }
}

@ConfigDsl
class DatabaseConfig {
    var url: String = ""
    var maxPoolSize: Int = 10
    var connectionTimeoutMs: Long = 5_000

    fun validate() {
        require(url.isNotBlank()) { "Database URL must not be blank" }
        require(maxPoolSize in 1..100) { "Pool size must be 1..100" }
    }
}

fun server(block: ServerConfig.() -> Unit): ServerConfig =
    ServerConfig().apply(block)

// Usage
val config = server {
    host = "0.0.0.0"
    port = 9090
    database {
        url = "jdbc:postgresql://localhost:5432/mydb"
        maxPoolSize = 20
    }
}
```

### Route Builder DSL

```kotlin
@DslMarker
annotation class RouteDsl

@RouteDsl
class Router {
    private val routes = mutableListOf<Route>()

    fun get(path: String, handler: suspend (Request) -> Response) {
        routes.add(Route("GET", path, handler))
    }

    fun post(path: String, handler: suspend (Request) -> Response) {
        routes.add(Route("POST", path, handler))
    }

    fun group(prefix: String, block: Router.() -> Unit) {
        val nested = Router().apply(block)
        routes.addAll(nested.routes.map { it.copy(path = "$prefix${it.path}") })
    }

    fun build(): List<Route> = routes.toList()
}

fun routes(block: Router.() -> Unit): List<Route> = Router().apply(block).build()

// Usage
val apiRoutes = routes {
    get("/health") { Response.ok("healthy") }
    group("/api/v1") {
        get("/users") { req -> userController.list(req) }
        post("/users") { req -> userController.create(req) }
    }
}
```

## Flow Patterns

### StateFlow for UI State

```kotlin
class UserViewModel(
    private val userService: UserService,
) : ViewModel() {
    private val _state = MutableStateFlow<UiState>(UiState.Loading)
    val state: StateFlow<UiState> = _state.asStateFlow()

    sealed interface UiState {
        data object Loading : UiState
        data class Loaded(val users: List<User>) : UiState
        data class Error(val message: String) : UiState
    }

    fun loadUsers() {
        viewModelScope.launch {
            _state.value = UiState.Loading
            when (val result = userService.listUsers()) {
                is Result.Success -> _state.value = UiState.Loaded(result.data)
                is Result.Failure -> _state.value = UiState.Error(result.error.message)
            }
        }
    }
}
```

### Combining Multiple Flows

```kotlin
val dashboardState: StateFlow<DashboardState> = combine(
    userRepository.observeCurrentUser(),
    notificationRepository.observeUnreadCount(),
    settingsRepository.observeTheme(),
) { user, unreadCount, theme ->
    DashboardState(user.name, unreadCount, theme)
}.stateIn(
    scope = viewModelScope,
    started = SharingStarted.WhileSubscribed(5_000),
    initialValue = DashboardState.DEFAULT,
)
```
