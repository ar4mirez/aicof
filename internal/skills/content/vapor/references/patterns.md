# Vapor Patterns Reference

## Contents

- [Fluent Query Patterns](#fluent-query-patterns)
- [Relationships and Eager Loading](#relationships-and-eager-loading)
- [Service Layer Pattern](#service-layer-pattern)
- [WebSocket Integration](#websocket-integration)
- [Leaf Templates](#leaf-templates)
- [Background Queues](#background-queues)
- [Testing](#testing)
- [Docker Deployment](#docker-deployment)
- [Package.swift Template](#packageswift-template)

## Fluent Query Patterns

### Filtering and Sorting

```swift
// Basic query with filters
let activeUsers = try await User.query(on: req.db)
    .filter(\.$isActive == true)
    .filter(\.$role == .admin)
    .sort(\.$createdAt, .descending)
    .all()

// Group filters with OR logic
let results = try await Post.query(on: req.db)
    .group(.or) { group in
        group.filter(\.$status == .published)
        group.filter(\.$status == .draft)
    }
    .all()

// Range filter
let recentPosts = try await Post.query(on: req.db)
    .filter(\.$createdAt >= Date().addingTimeInterval(-86400 * 7))
    .all()

// LIKE / CONTAINS
let matches = try await User.query(on: req.db)
    .filter(\.$name, .contains(inverse: false, .anywhere), "john")
    .all()
```

### Pagination

```swift
struct PageRequest: Content {
    var page: Int
    var per: Int

    init(page: Int = 1, per: Int = 20) {
        self.page = max(1, page)
        self.per = min(100, max(1, per))
    }
}

struct PaginatedResponse<T: Content>: Content {
    let items: [T]
    let metadata: PageMetadata
}

struct PageMetadata: Content {
    let page: Int
    let perPage: Int
    let total: Int
    let totalPages: Int
}

// Usage in controller
@Sendable
func index(req: Request) async throws -> PaginatedResponse<UserResponse> {
    let page = try req.query.decode(PageRequest.self)
    let result = try await User.query(on: req.db)
        .filter(\.$isActive == true)
        .sort(\.$createdAt, .descending)
        .paginate(PageRequest(page: page.page, per: page.per))

    return PaginatedResponse(
        items: try result.items.map { try UserResponse(user: $0) },
        metadata: PageMetadata(
            page: page.page,
            perPage: page.per,
            total: result.metadata.total,
            totalPages: result.metadata.pageCount
        )
    )
}
```

### Aggregation

```swift
// Count
let totalUsers = try await User.query(on: req.db)
    .filter(\.$isActive == true)
    .count()

// Sum, average, min, max
let totalAmount = try await Order.query(on: req.db)
    .sum(\.$amount)

// First / unique result
let admin = try await User.query(on: req.db)
    .filter(\.$email == "admin@example.com")
    .first()
```

### Transactions

```swift
let (user, token) = try await req.db.transaction { database in
    let user = User(email: email, passwordHash: hash, name: name)
    try await user.save(on: database)

    let token = try user.generateToken()
    try await token.save(on: database)

    return (user, token)
}
```

## Relationships and Eager Loading

### Parent-Child Relationship

```swift
// Post model with parent reference
final class Post: Model, Content, @unchecked Sendable {
    static let schema = "posts"

    @ID(key: .id) var id: UUID?
    @Field(key: "title") var title: String
    @Field(key: "content") var content: String
    @Parent(key: "user_id") var user: User
    @Timestamp(key: "created_at", on: .create) var createdAt: Date?

    init() {}

    init(id: UUID? = nil, title: String, content: String, userID: User.IDValue) {
        self.id = id
        self.title = title
        self.content = content
        self.$user.id = userID
    }
}

// Migration with foreign key
struct CreatePost: AsyncMigration {
    func prepare(on database: Database) async throws {
        try await database.schema("posts")
            .id()
            .field("title", .string, .required)
            .field("content", .string, .required)
            .field("user_id", .uuid, .required,
                   .references("users", "id", onDelete: .cascade))
            .field("created_at", .datetime)
            .create()
    }

    func revert(on database: Database) async throws {
        try await database.schema("posts").delete()
    }
}
```

### Siblings (Many-to-Many)

```swift
// Tag model
final class Tag: Model, Content, @unchecked Sendable {
    static let schema = "tags"

    @ID(key: .id) var id: UUID?
    @Field(key: "name") var name: String
    @Siblings(through: PostTag.self, from: \.$tag, to: \.$post) var posts: [Post]

    init() {}
    init(id: UUID? = nil, name: String) { self.id = id; self.name = name }
}

// Pivot model
final class PostTag: Model, @unchecked Sendable {
    static let schema = "post_tags"

    @ID(key: .id) var id: UUID?
    @Parent(key: "post_id") var post: Post
    @Parent(key: "tag_id") var tag: Tag

    init() {}
    init(id: UUID? = nil, postID: Post.IDValue, tagID: Tag.IDValue) {
        self.id = id
        self.$post.id = postID
        self.$tag.id = tagID
    }
}

// Attach/detach siblings
try await post.$tags.attach(tag, on: req.db)
try await post.$tags.detach(tag, on: req.db)
```

### Eager Loading

```swift
// Load children
let users = try await User.query(on: req.db)
    .with(\.$posts)
    .all()

// Nested eager loading
let users = try await User.query(on: req.db)
    .with(\.$posts) { post in
        post.with(\.$tags)
    }
    .all()

// Load parent
let posts = try await Post.query(on: req.db)
    .with(\.$user)
    .all()

// Load siblings through pivot
let posts = try await Post.query(on: req.db)
    .with(\.$tags)
    .all()
```

**Eager loading rules:**
- Always use `.with()` instead of lazy-loading in request handlers (prevents N+1)
- Nest `.with()` for multi-level relationships
- Only load relationships you actually need in the response

## Service Layer Pattern

```swift
// Protocol for testability
protocol UserServiceProtocol: Sendable {
    func findByID(_ id: UUID, on db: Database) async throws -> User?
    func findByEmail(_ email: String, on db: Database) async throws -> User?
    func create(_ request: CreateUserRequest, on db: Database) async throws -> User
    func update(_ user: User, with request: UpdateUserRequest, on db: Database) async throws -> User
    func softDelete(_ user: User, on db: Database) async throws
}

struct UserService: UserServiceProtocol {
    func findByID(_ id: UUID, on db: Database) async throws -> User? {
        try await User.find(id, on: db)
    }

    func findByEmail(_ email: String, on db: Database) async throws -> User? {
        try await User.query(on: db)
            .filter(\.$email == email)
            .first()
    }

    func create(_ request: CreateUserRequest, on db: Database) async throws -> User {
        let hash = try Bcrypt.hash(request.password)
        let user = User(email: request.email, passwordHash: hash, name: request.name)
        try await user.save(on: db)
        return user
    }

    func update(_ user: User, with request: UpdateUserRequest, on db: Database) async throws -> User {
        if let name = request.name { user.name = name }
        if let email = request.email { user.email = email }
        try await user.save(on: db)
        return user
    }

    func softDelete(_ user: User, on db: Database) async throws {
        user.isActive = false
        try await user.save(on: db)
    }
}

// Register on Application and Request for dependency injection
extension Application {
    var userService: UserServiceProtocol { UserService() }
}

extension Request {
    var userService: UserServiceProtocol { application.userService }
}
```

## WebSocket Integration

```swift
// routes.swift
func routes(_ app: Application) throws {
    app.webSocket("ws", "chat", ":roomID") { req, ws in
        let roomID = req.parameters.get("roomID") ?? "default"
        req.logger.info("WebSocket connected to room: \(roomID)")

        // Handle incoming messages
        ws.onText { ws, text in
            req.logger.info("Received: \(text)")
            // Broadcast to all connected clients (implement via actor)
            await ChatManager.shared.broadcast(text, in: roomID, from: ws)
        }

        ws.onClose.whenComplete { _ in
            req.logger.info("WebSocket disconnected from room: \(roomID)")
            Task {
                await ChatManager.shared.remove(ws, from: roomID)
            }
        }

        // Register this connection
        await ChatManager.shared.add(ws, to: roomID)
    }
}

// ChatManager actor for thread-safe connection tracking
actor ChatManager {
    static let shared = ChatManager()
    private var rooms: [String: [WebSocket]] = [:]

    func add(_ ws: WebSocket, to room: String) {
        rooms[room, default: []].append(ws)
    }

    func remove(_ ws: WebSocket, from room: String) {
        rooms[room]?.removeAll { $0 === ws }
    }

    func broadcast(_ message: String, in room: String, from sender: WebSocket) {
        guard let clients = rooms[room] else { return }
        for client in clients where client !== sender {
            client.send(message)
        }
    }
}
```

## Leaf Templates

### Setup

```swift
// In configure.swift
import Leaf

app.views.use(.leaf)
```

### Template Rendering

```swift
// Controller
@Sendable
func index(req: Request) async throws -> View {
    let posts = try await Post.query(on: req.db)
        .with(\.$user)
        .sort(\.$createdAt, .descending)
        .limit(10)
        .all()

    return try await req.view.render("posts/index", [
        "title": "Recent Posts",
        "posts": posts,
    ])
}
```

### Leaf Template Syntax

```html
<!-- Resources/Views/posts/index.leaf -->
<!DOCTYPE html>
<html>
<head><title>#(title)</title></head>
<body>
    <h1>#(title)</h1>

    #for(post in posts):
        <article>
            <h2>#(post.title)</h2>
            <p>By #(post.user.name)</p>
            <p>#(post.content)</p>

            #if(post.status == "published"):
                <span class="badge">Published</span>
            #else:
                <span class="badge">Draft</span>
            #endif
        </article>
    #endfor

    #if(count(posts) == 0):
        <p>No posts yet.</p>
    #endif
</body>
</html>
```

### Layout Inheritance

```html
<!-- Resources/Views/layout.leaf -->
<!DOCTYPE html>
<html>
<head>
    <title>#(title) - My App</title>
    <link rel="stylesheet" href="/css/style.css">
</head>
<body>
    <nav><!-- navigation --></nav>
    <main>#import("content")</main>
    <footer><!-- footer --></footer>
</body>
</html>

<!-- Resources/Views/posts/index.leaf -->
#extend("layout"):
    #export("content"):
        <h1>#(title)</h1>
        <!-- page content here -->
    #endexport
#endextend
```

**Leaf conventions:**
- Place templates in `Resources/Views/`
- Use `#extend`/`#import`/`#export` for layout inheritance
- Use `#for`, `#if`/`#else`/`#endif` for control flow
- Access variables with `#(variableName)`

## Background Queues

### Using Vapor Queues (Redis-backed)

```swift
// Package.swift -- add dependency
.package(url: "https://github.com/vapor/queues-redis-driver.git", from: "1.0.0")

// configure.swift
import QueuesRedisDriver

try app.queues.use(.redis(url: Environment.get("REDIS_URL") ?? "redis://localhost:6379"))
app.queues.add(SendEmailJob())
try app.queues.startInProcessJobs()
```

### Job Definition

```swift
import Queues
import Vapor

struct EmailPayload: Codable {
    let to: String
    let subject: String
    let body: String
}

struct SendEmailJob: AsyncJob {
    typealias Payload = EmailPayload

    func dequeue(_ context: QueueContext, _ payload: EmailPayload) async throws {
        context.logger.info("Sending email to \(payload.to)")
        // Send email via SMTP client or API
        try await EmailClient.send(
            to: payload.to,
            subject: payload.subject,
            body: payload.body
        )
    }

    func error(_ context: QueueContext, _ error: Error, _ payload: EmailPayload) async throws {
        context.logger.error("Failed to send email to \(payload.to): \(error)")
    }
}
```

### Dispatching Jobs

```swift
// In a controller or service
@Sendable
func register(req: Request) async throws -> UserResponse {
    let user = try await createUser(req)

    // Dispatch email job asynchronously
    try await req.queue.dispatch(SendEmailJob.self, EmailPayload(
        to: user.email,
        subject: "Welcome!",
        body: "Thanks for signing up."
    ))

    return try UserResponse(user: user)
}
```

## Testing

### Test Setup with XCTVapor

```swift
import XCTVapor
@testable import App

final class UserControllerTests: XCTestCase {
    var app: Application!

    override func setUp() async throws {
        app = try await Application.make(.testing)
        try await configure(app)
        try await app.autoMigrate()
    }

    override func tearDown() async throws {
        try await app.autoRevert()
        try await app.asyncShutdown()
        app = nil
    }
```

### Testing Endpoints

```swift
    func testRegisterUser() async throws {
        let body = CreateUserRequest(
            email: "test@example.com", password: "password123", name: "Test User"
        )

        try await app.test(.POST, "api/v1/auth/register") { req in
            try req.content.encode(body)
        } afterResponse: { res in
            XCTAssertEqual(res.status, .ok)
            let user = try res.content.decode(UserResponse.self)
            XCTAssertEqual(user.email, "test@example.com")
            XCTAssertEqual(user.name, "Test User")
        }
    }

    func testGetUsersRequiresAuth() async throws {
        try await app.test(.GET, "api/v1/users") { res in
            XCTAssertEqual(res.status, .unauthorized)
        }
    }

    func testGetUsersWithAuth() async throws {
        let user = try await User.create(
            email: "auth@example.com", password: "password123",
            name: "Auth User", on: app.db
        )

        let payload = UserPayload(
            subject: .init(value: try user.requireID().uuidString),
            expiration: .init(value: Date().addingTimeInterval(3600))
        )
        let token = try await app.jwt.keys.sign(payload)

        try await app.test(.GET, "api/v1/users") { req in
            req.headers.bearerAuthorization = BearerAuthorization(token: token)
        } afterResponse: { res in
            XCTAssertEqual(res.status, .ok)
            let response = try res.content.decode(PaginatedResponse<UserResponse>.self)
            XCTAssertFalse(response.items.isEmpty)
        }
    }

    func testLoginReturnsToken() async throws {
        _ = try await User.create(
            email: "login@example.com", password: "password123",
            name: "Login User", on: app.db
        )

        let loginBody = LoginRequest(email: "login@example.com", password: "password123")

        try await app.test(.POST, "api/v1/auth/login") { req in
            try req.content.encode(loginBody)
        } afterResponse: { res in
            XCTAssertEqual(res.status, .ok)
            let token = try res.content.decode(TokenResponse.self)
            XCTAssertFalse(token.accessToken.isEmpty)
            XCTAssertEqual(token.tokenType, "Bearer")
        }
    }

    func testRegisterDuplicateEmailFails() async throws {
        _ = try await User.create(
            email: "dupe@example.com", password: "password123",
            name: "First", on: app.db
        )

        let body = CreateUserRequest(
            email: "dupe@example.com", password: "password123", name: "Second"
        )

        try await app.test(.POST, "api/v1/auth/register") { req in
            try req.content.encode(body)
        } afterResponse: { res in
            XCTAssertEqual(res.status, .conflict)
        }
    }
}
```

### Testing Best Practices

- Use `.testing` environment with a separate test database (SQLite in-memory works well)
- Call `autoMigrate` in `setUp` and `autoRevert` in `tearDown` for clean state
- Test both success and error paths for every endpoint
- Test authentication requirements (missing token, expired token, wrong role)
- Test validation (missing fields, invalid formats, boundary values)
- Use `app.test()` instead of raw `XCTVapor` requests for cleaner tests

## Docker Deployment

### Dockerfile

```dockerfile
# Build stage
FROM swift:5.9-jammy as build
WORKDIR /app
COPY Package.swift Package.resolved ./
RUN swift package resolve
COPY . .
RUN swift build -c release --static-swift-stdlib

# Production stage
FROM ubuntu:jammy
RUN apt-get update && apt-get install -y \
    libcurl4 libxml2 tzdata \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=build /app/.build/release/App /app/App
COPY --from=build /app/Public /app/Public
COPY --from=build /app/Resources /app/Resources

ENV ENVIRONMENT=production
EXPOSE 8080
ENTRYPOINT ["./App"]
CMD ["serve", "--env", "production", "--hostname", "0.0.0.0", "--port", "8080"]
```

### docker-compose.yml

```yaml
services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://vapor:vapor@db:5432/vapor
      - REDIS_URL=redis://redis:6379
      - JWT_SECRET=${JWT_SECRET}
    depends_on:
      - db
      - redis

  db:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: vapor
      POSTGRES_PASSWORD: vapor
      POSTGRES_DB: vapor
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

volumes:
  postgres_data:
```

### Deployment Checklist

- Set `ENVIRONMENT=production` (disables debug logging, autoMigrate)
- Run migrations explicitly: `swift run App migrate` or via CI/CD
- Set all required env vars: `DATABASE_URL`, `JWT_SECRET`, `REDIS_URL`
- Use `--static-swift-stdlib` for smaller, self-contained binaries
- Configure health check endpoint for container orchestration
- Set up log aggregation (Vapor uses swift-log, integrates with any backend)

## Package.swift Template

```swift
// swift-tools-version:5.9
import PackageDescription

let package = Package(
    name: "MyVaporApp",
    platforms: [.macOS(.v13)],
    dependencies: [
        .package(url: "https://github.com/vapor/vapor.git", from: "4.89.0"),
        .package(url: "https://github.com/vapor/fluent.git", from: "4.9.0"),
        .package(url: "https://github.com/vapor/fluent-postgres-driver.git", from: "2.8.0"),
        .package(url: "https://github.com/vapor/redis.git", from: "4.10.0"),
        .package(url: "https://github.com/vapor/jwt.git", from: "4.2.0"),
        .package(url: "https://github.com/vapor/leaf.git", from: "4.3.0"),
    ],
    targets: [
        .executableTarget(
            name: "App",
            dependencies: [
                .product(name: "Vapor", package: "vapor"),
                .product(name: "Fluent", package: "fluent"),
                .product(name: "FluentPostgresDriver", package: "fluent-postgres-driver"),
                .product(name: "Redis", package: "redis"),
                .product(name: "JWT", package: "jwt"),
                .product(name: "Leaf", package: "leaf"),
            ]
        ),
        .testTarget(
            name: "AppTests",
            dependencies: [
                .target(name: "App"),
                .product(name: "XCTVapor", package: "vapor"),
            ]
        )
    ]
)
```
