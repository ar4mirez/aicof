# Swift Patterns Reference

## Contents

- [Actor with Published State](#actor-with-published-state)
- [Actor-Based Service Layer](#actor-based-service-layer)
- [Protocol Composition](#protocol-composition)
- [AsyncSequence Patterns](#asyncsequence-patterns)
- [Dependency Injection via Protocols](#dependency-injection-via-protocols)
- [Retry with Exponential Backoff](#retry-with-exponential-backoff)

## Actor with Published State

```swift
import Observation

@Observable
final class SessionManager {
    private let auth: AuthActor
    var currentUser: User?
    var isAuthenticated: Bool { currentUser != nil }

    init(auth: AuthActor) { self.auth = auth }

    func login(email: String, password: String) async throws {
        let token = try await auth.authenticate(email: email, password: password)
        currentUser = try await auth.fetchProfile(token: token)
    }

    func logout() async {
        await auth.invalidateSession()
        currentUser = nil
    }
}

actor AuthActor {
    private var activeToken: String?

    func authenticate(email: String, password: String) async throws -> String {
        let token = try await APIClient.shared.post("/auth/login", body: [
            "email": email, "password": password,
        ])
        activeToken = token
        return token
    }

    func fetchProfile(token: String) async throws -> User {
        try await APIClient.shared.get("/users/me", bearerToken: token)
    }

    func invalidateSession() { activeToken = nil }
}
```

## Actor-Based Service Layer

```swift
actor OrderService {
    private let repository: any OrderRepository
    private let payments: any PaymentGateway
    private var processing: Set<String> = []

    init(repository: any OrderRepository, payments: any PaymentGateway) {
        self.repository = repository
        self.payments = payments
    }

    func placeOrder(_ draft: OrderDraft) async throws -> Order {
        var order = Order(from: draft)
        guard !processing.contains(order.id) else {
            throw OrderError.alreadyProcessing(order.id)
        }
        processing.insert(order.id)
        defer { processing.remove(order.id) }

        let charge = try await payments.charge(amount: order.total, currency: order.currency)
        order.paymentID = charge.id
        order.status = .confirmed
        try await repository.save(order)
        return order
    }
}
```

## Protocol Composition

```swift
// Small, focused protocols
protocol Fetchable {
    associatedtype ID: Hashable & Sendable
    static func fetch(by id: ID) async throws -> Self
}

protocol Persistable {
    func save() async throws
    func delete() async throws
}

protocol Validatable {
    func validate() throws
}

// Compose for specific needs
typealias CRUDModel = Fetchable & Persistable & Validatable

// Generic repository using primary associated type
protocol Repository<Model> {
    associatedtype Model: Identifiable & Sendable
    func findByID(_ id: Model.ID) async throws -> Model?
    func save(_ model: Model) async throws
    func delete(_ id: Model.ID) async throws
}
```

## AsyncSequence Patterns

### AsyncStream for Bridging Callbacks

```swift
func locationUpdates() -> AsyncStream<CLLocation> {
    AsyncStream { continuation in
        let delegate = LocationDelegate(
            onUpdate: { continuation.yield($0) },
            onComplete: { continuation.finish() }
        )
        continuation.onTermination = { _ in delegate.stopUpdating() }
        delegate.startUpdating()
    }
}
```

### Chaining Async Sequences

```swift
func processEvents() async throws {
    let significant = eventStream()
        .filter { $0.priority >= .high }
        .map { EnrichedEvent(from: $0) }
        .prefix(100)

    for try await event in significant {
        try await handleEvent(event)
    }
}
```

## Dependency Injection via Protocols

```swift
protocol Clock: Sendable {
    func now() -> Date
}

struct SystemClock: Clock { func now() -> Date { Date() } }
struct FixedClock: Clock { let fixedDate: Date; func now() -> Date { fixedDate } }

struct TokenValidator {
    private let clock: any Clock
    init(clock: any Clock = SystemClock()) { self.clock = clock }

    func isValid(_ token: Token) -> Bool {
        token.expiresAt > clock.now()
    }
}

// In tests
func test_expiredToken_isInvalid() {
    let clock = FixedClock(fixedDate: Date(timeIntervalSince1970: 1_000_000))
    let validator = TokenValidator(clock: clock)
    let expired = Token(expiresAt: Date(timeIntervalSince1970: 999_999))
    XCTAssertFalse(validator.isValid(expired))
}
```

## Retry with Exponential Backoff

```swift
func withRetry<T: Sendable>(
    maxAttempts: Int = 3,
    initialDelay: Duration = .milliseconds(200),
    maxDelay: Duration = .seconds(10),
    operation: @Sendable () async throws -> T
) async throws -> T {
    var delay = initialDelay
    for attempt in 1...maxAttempts {
        do {
            return try await operation()
        } catch {
            if attempt == maxAttempts { throw error }
            try Task.checkCancellation()
            try await Task.sleep(for: delay)
            delay = min(delay * 2, maxDelay)
        }
    }
    fatalError("Unreachable")
}

// Usage
let data = try await withRetry { try await networkClient.fetch(url) }
```
