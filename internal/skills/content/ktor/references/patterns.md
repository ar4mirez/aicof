# Ktor Patterns Reference

## Contents

- [Dependencies (build.gradle.kts)](#dependencies-buildgradlekts)
- [Database Integration (Exposed)](#database-integration-exposed)
- [Models and DTOs](#models-and-dtos)
- [Repository Pattern](#repository-pattern)
- [Service Layer](#service-layer)
- [JWT Utilities](#jwt-utilities)
- [WebSocket Patterns](#websocket-patterns)
- [Ktor HTTP Client](#ktor-http-client)
- [Testing Strategies](#testing-strategies)
- [Deployment](#deployment)
- [CORS and Call Logging](#cors-and-call-logging)

## Dependencies (build.gradle.kts)

```kotlin
plugins {
    kotlin("jvm") version "1.9.22"
    kotlin("plugin.serialization") version "1.9.22"
    id("io.ktor.plugin") version "2.3.7"
}

group = "com.example"
version = "0.0.1"

application {
    mainClass.set("com.example.ApplicationKt")
    val isDevelopment: Boolean = project.ext.has("development")
    applicationDefaultJvmArgs = listOf("-Dio.ktor.development=$isDevelopment")
}

repositories { mavenCentral() }

dependencies {
    // Ktor Server Core
    implementation("io.ktor:ktor-server-core-jvm")
    implementation("io.ktor:ktor-server-netty-jvm")
    implementation("io.ktor:ktor-server-host-common-jvm")

    // Content Negotiation & Serialization
    implementation("io.ktor:ktor-server-content-negotiation-jvm")
    implementation("io.ktor:ktor-serialization-kotlinx-json-jvm")

    // Authentication
    implementation("io.ktor:ktor-server-auth-jvm")
    implementation("io.ktor:ktor-server-auth-jwt-jvm")

    // Validation
    implementation("io.ktor:ktor-server-request-validation")

    // Status Pages (Error Handling)
    implementation("io.ktor:ktor-server-status-pages-jvm")

    // CORS
    implementation("io.ktor:ktor-server-cors-jvm")

    // Call Logging
    implementation("io.ktor:ktor-server-call-logging-jvm")

    // WebSockets
    implementation("io.ktor:ktor-server-websockets-jvm")

    // HTTP Client (for service-to-service calls)
    implementation("io.ktor:ktor-client-core-jvm")
    implementation("io.ktor:ktor-client-cio-jvm")
    implementation("io.ktor:ktor-client-content-negotiation-jvm")

    // Database (Exposed + HikariCP + PostgreSQL)
    implementation("org.jetbrains.exposed:exposed-core:0.45.0")
    implementation("org.jetbrains.exposed:exposed-dao:0.45.0")
    implementation("org.jetbrains.exposed:exposed-jdbc:0.45.0")
    implementation("org.jetbrains.exposed:exposed-java-time:0.45.0")
    implementation("com.zaxxer:HikariCP:5.1.0")
    implementation("org.postgresql:postgresql:42.7.1")

    // Password Hashing
    implementation("at.favre.lib:bcrypt:0.10.2")

    // Logging
    implementation("ch.qos.logback:logback-classic:1.4.14")

    // Testing
    testImplementation("io.ktor:ktor-server-tests-jvm")
    testImplementation("io.ktor:ktor-client-content-negotiation")
    testImplementation("org.jetbrains.kotlin:kotlin-test-junit:1.9.22")
    testImplementation("io.mockk:mockk:1.13.9")
    testImplementation("com.h2database:h2:2.2.224")
}

ktor {
    fatJar { archiveFileName.set("app.jar") }
}
```

## Database Integration (Exposed)

### Connection Pool Configuration

```kotlin
// src/main/kotlin/com/example/plugins/Databases.kt
fun Application.configureDatabases() {
    val config = environment.config

    val hikariConfig = HikariConfig().apply {
        jdbcUrl = config.property("database.url").getString()
        driverClassName = config.property("database.driver").getString()
        username = config.property("database.user").getString()
        password = config.property("database.password").getString()
        maximumPoolSize = config.property("database.maxPoolSize").getString().toInt()
        isAutoCommit = false
        transactionIsolation = "TRANSACTION_REPEATABLE_READ"
        validate()
    }

    val dataSource = HikariDataSource(hikariConfig)
    Database.connect(dataSource)

    // Create tables (use migrations in production)
    transaction {
        SchemaUtils.create(Users)
    }
}
```

### Table Definitions

```kotlin
// Exposed table definition with typed columns
object Users : LongIdTable("users") {
    val email = varchar("email", 255).uniqueIndex()
    val passwordHash = varchar("password_hash", 255)
    val name = varchar("name", 255)
    val role = varchar("role", 50).default("user")
    val createdAt = datetime("created_at").default(LocalDateTime.now())
    val updatedAt = datetime("updated_at").default(LocalDateTime.now())
}
```

### Coroutine-Safe Transactions

Always use `newSuspendedTransaction` instead of `transaction` for coroutine contexts:

```kotlin
// BAD: blocks the coroutine dispatcher
transaction {
    Users.selectAll().map(User::fromRow)
}

// GOOD: coroutine-aware, does not block
newSuspendedTransaction {
    Users.selectAll().map(User::fromRow)
}

// GOOD: extracted into a reusable helper
private suspend fun <T> dbQuery(block: suspend () -> T): T =
    newSuspendedTransaction { block() }
```

## Models and DTOs

### Domain Model with Row Mapping

```kotlin
// Domain model (not serializable -- internal use only)
data class User(
    val id: Long,
    val email: String,
    val passwordHash: String,
    val name: String,
    val role: String,
    val createdAt: LocalDateTime,
    val updatedAt: LocalDateTime,
) {
    companion object {
        fun fromRow(row: ResultRow): User = User(
            id = row[Users.id].value,
            email = row[Users.email],
            passwordHash = row[Users.passwordHash],
            name = row[Users.name],
            role = row[Users.role],
            createdAt = row[Users.createdAt],
            updatedAt = row[Users.updatedAt],
        )
    }
}
```

### Response DTO

```kotlin
// API response (serializable, no sensitive fields)
@Serializable
data class UserResponse(
    val id: Long,
    val email: String,
    val name: String,
    val role: String,
    val createdAt: String,
) {
    companion object {
        fun from(user: User): UserResponse = UserResponse(
            id = user.id,
            email = user.email,
            name = user.name,
            role = user.role,
            createdAt = user.createdAt.toString(),
        )
    }
}
```

### Request DTOs

```kotlin
@Serializable
data class CreateUserRequest(
    val email: String,
    val password: String,
    val name: String,
)

@Serializable
data class UpdateUserRequest(
    val name: String? = null,
    val email: String? = null,
)

@Serializable
data class LoginRequest(
    val email: String,
    val password: String,
)

@Serializable
data class LoginResponse(
    val token: String,
    val user: UserResponse,
)
```

## Repository Pattern

### Full CRUD Repository

```kotlin
class UserRepository {

    private suspend fun <T> dbQuery(block: suspend () -> T): T =
        newSuspendedTransaction { block() }

    suspend fun findById(id: Long): User? = dbQuery {
        Users.select { Users.id eq id }
            .map(User::fromRow)
            .singleOrNull()
    }

    suspend fun findByEmail(email: String): User? = dbQuery {
        Users.select { Users.email eq email }
            .map(User::fromRow)
            .singleOrNull()
    }

    suspend fun findAll(limit: Int = 20, offset: Long = 0): List<User> = dbQuery {
        Users.selectAll()
            .orderBy(Users.createdAt, SortOrder.DESC)
            .limit(limit, offset)
            .map(User::fromRow)
    }

    suspend fun create(
        email: String,
        passwordHash: String,
        name: String,
        role: String = "user",
    ): User = dbQuery {
        val id = Users.insertAndGetId {
            it[Users.email] = email
            it[Users.passwordHash] = passwordHash
            it[Users.name] = name
            it[Users.role] = role
            it[createdAt] = LocalDateTime.now()
            it[updatedAt] = LocalDateTime.now()
        }
        Users.select { Users.id eq id }.map(User::fromRow).single()
    }

    suspend fun update(id: Long, name: String?, email: String?): User? = dbQuery {
        val updated = Users.update({ Users.id eq id }) {
            name?.let { n -> it[Users.name] = n }
            email?.let { e -> it[Users.email] = e }
            it[updatedAt] = LocalDateTime.now()
        }
        if (updated > 0) {
            Users.select { Users.id eq id }.map(User::fromRow).singleOrNull()
        } else null
    }

    suspend fun delete(id: Long): Boolean = dbQuery {
        Users.deleteWhere { Users.id eq id } > 0
    }

    suspend fun existsByEmail(email: String): Boolean = dbQuery {
        Users.select { Users.email eq email }.count() > 0
    }
}
```

## Service Layer

### Service with Business Logic

```kotlin
class UserService(
    private val userRepository: UserRepository,
    private val jwtUtils: JwtUtils,
) {
    suspend fun createUser(request: CreateUserRequest): UserResponse {
        if (userRepository.existsByEmail(request.email)) {
            throw ConflictException("Email already registered")
        }
        val passwordHash = BCrypt.withDefaults()
            .hashToString(12, request.password.toCharArray())
        val user = userRepository.create(
            email = request.email,
            passwordHash = passwordHash,
            name = request.name,
        )
        return UserResponse.from(user)
    }

    suspend fun getUserById(id: Long): UserResponse {
        val user = userRepository.findById(id)
            ?: throw NotFoundException("User not found")
        return UserResponse.from(user)
    }

    suspend fun getAllUsers(limit: Int, offset: Long): List<UserResponse> {
        return userRepository.findAll(limit, offset).map(UserResponse::from)
    }

    suspend fun updateUser(id: Long, request: UpdateUserRequest): UserResponse {
        if (request.email != null) {
            val existing = userRepository.findByEmail(request.email)
            if (existing != null && existing.id != id) {
                throw ConflictException("Email already in use")
            }
        }
        val user = userRepository.update(id, request.name, request.email)
            ?: throw NotFoundException("User not found")
        return UserResponse.from(user)
    }

    suspend fun deleteUser(id: Long) {
        if (!userRepository.delete(id)) {
            throw NotFoundException("User not found")
        }
    }

    suspend fun login(request: LoginRequest): LoginResponse {
        val user = userRepository.findByEmail(request.email)
            ?: throw UnauthorizedException("Invalid credentials")
        val passwordValid = BCrypt.verifyer()
            .verify(request.password.toCharArray(), user.passwordHash)
            .verified
        if (!passwordValid) {
            throw UnauthorizedException("Invalid credentials")
        }
        return LoginResponse(
            token = jwtUtils.generateToken(user),
            user = UserResponse.from(user),
        )
    }
}
```

## JWT Utilities

```kotlin
class JwtUtils(
    private val secret: String,
    private val issuer: String,
    private val audience: String,
    private val expirationMs: Long,
) {
    fun generateToken(user: User): String {
        return JWT.create()
            .withAudience(audience)
            .withIssuer(issuer)
            .withClaim("userId", user.id.toString())
            .withClaim("email", user.email)
            .withClaim("role", user.role)
            .withExpiresAt(Date(System.currentTimeMillis() + expirationMs))
            .sign(Algorithm.HMAC256(secret))
    }
}
```

## WebSocket Patterns

### Connection Manager

```kotlin
class ConnectionManager {
    private val connections = ConcurrentHashMap<String, WebSocketSession>()

    fun addConnection(userId: String, session: WebSocketSession) {
        connections[userId] = session
    }

    fun removeConnection(userId: String) {
        connections.remove(userId)
    }

    suspend fun broadcast(message: String) {
        connections.values.forEach { session ->
            session.send(Frame.Text(message))
        }
    }

    suspend fun sendTo(userId: String, message: String) {
        connections[userId]?.send(Frame.Text(message))
    }
}
```

### WebSocket Routes

```kotlin
fun Route.webSocketRoutes(connectionManager: ConnectionManager) {
    webSocket("/ws/{userId}") {
        val userId = call.parameters["userId"] ?: return@webSocket close(
            CloseReason(CloseReason.Codes.VIOLATED_POLICY, "No user ID")
        )

        connectionManager.addConnection(userId, this)

        try {
            incoming.consumeEach { frame ->
                when (frame) {
                    is Frame.Text -> {
                        val text = frame.readText()
                        connectionManager.broadcast("$userId: $text")
                    }
                    else -> {}
                }
            }
        } finally {
            connectionManager.removeConnection(userId)
        }
    }
}
```

### WebSocket Plugin Installation

```kotlin
fun Application.configureWebSockets() {
    install(WebSockets) {
        pingPeriod = Duration.ofSeconds(15)
        timeout = Duration.ofSeconds(15)
        maxFrameSize = Long.MAX_VALUE
        masking = false
    }
}
```

## Ktor HTTP Client

### Service-to-Service Communication

```kotlin
class ExternalApiClient(
    private val baseUrl: String,
    private val apiKey: String,
) {
    private val client = HttpClient(CIO) {
        install(ContentNegotiation) { json() }
        install(HttpTimeout) {
            requestTimeoutMillis = 10_000
            connectTimeoutMillis = 5_000
        }
        defaultRequest {
            header("Authorization", "Bearer $apiKey")
            contentType(ContentType.Application.Json)
        }
    }

    suspend fun getResource(id: String): ExternalResource {
        return client.get("$baseUrl/resources/$id").body()
    }

    suspend fun createResource(request: CreateResourceRequest): ExternalResource {
        return client.post("$baseUrl/resources") {
            setBody(request)
        }.body()
    }

    fun close() { client.close() }
}
```

### Client with Retry

```kotlin
suspend fun <T> HttpClient.retryRequest(
    maxRetries: Int = 3,
    initialDelayMs: Long = 200,
    block: suspend HttpClient.() -> T,
): T {
    var currentDelay = initialDelayMs
    repeat(maxRetries - 1) { attempt ->
        try {
            return block()
        } catch (e: CancellationException) { throw e }
        catch (e: Exception) {
            logger.warn("HTTP request attempt ${attempt + 1} failed: ${e.message}")
        }
        delay(currentDelay)
        currentDelay = (currentDelay * 2).coerceAtMost(5_000)
    }
    return block()
}
```

## Testing Strategies

### Full Integration Test

```kotlin
class ApplicationTest {
    @Test
    fun `create user returns 201`() = testApplication {
        application { module() }
        val client = createClient {
            install(ContentNegotiation) { json() }
        }

        val response = client.post("/api/users") {
            contentType(ContentType.Application.Json)
            setBody(CreateUserRequest(
                email = "test@example.com",
                password = "password123",
                name = "Test User",
            ))
        }

        assertEquals(HttpStatusCode.Created, response.status)
        val user = response.body<UserResponse>()
        assertEquals("test@example.com", user.email)
        assertEquals("Test User", user.name)
    }

    @Test
    fun `login returns token`() = testApplication {
        application { module() }
        val client = createClient {
            install(ContentNegotiation) { json() }
        }

        // Create user first
        client.post("/api/users") {
            contentType(ContentType.Application.Json)
            setBody(CreateUserRequest("login@example.com", "password123", "Login User"))
        }

        // Then login
        val response = client.post("/api/auth/login") {
            contentType(ContentType.Application.Json)
            setBody(LoginRequest("login@example.com", "password123"))
        }

        assertEquals(HttpStatusCode.OK, response.status)
        val loginResponse = response.body<LoginResponse>()
        assertNotNull(loginResponse.token)
    }

    @Test
    fun `get users requires authentication`() = testApplication {
        application { module() }
        val response = client.get("/api/users")
        assertEquals(HttpStatusCode.Unauthorized, response.status)
    }

    @Test
    fun `authenticated request returns users`() = testApplication {
        application { module() }
        val client = createClient {
            install(ContentNegotiation) { json() }
        }

        // Create and login to get token
        client.post("/api/users") {
            contentType(ContentType.Application.Json)
            setBody(CreateUserRequest("auth@example.com", "password123", "Auth User"))
        }
        val loginResponse = client.post("/api/auth/login") {
            contentType(ContentType.Application.Json)
            setBody(LoginRequest("auth@example.com", "password123"))
        }.body<LoginResponse>()

        // Use token for authenticated request
        val response = client.get("/api/users") {
            header(HttpHeaders.Authorization, "Bearer ${loginResponse.token}")
        }

        assertEquals(HttpStatusCode.OK, response.status)
        val users = response.body<List<UserResponse>>()
        assertTrue(users.isNotEmpty())
    }
}
```

### Service Unit Tests with MockK

```kotlin
class UserServiceTest {
    private lateinit var userRepository: UserRepository
    private lateinit var jwtUtils: JwtUtils
    private lateinit var userService: UserService

    @BeforeTest
    fun setup() {
        userRepository = mockk()
        jwtUtils = mockk()
        userService = UserService(userRepository, jwtUtils)
    }

    @AfterTest
    fun teardown() { clearAllMocks() }

    @Test
    fun `createUser with valid data returns user response`() = runBlocking {
        val request = CreateUserRequest("test@example.com", "password123", "Test User")
        val user = User(
            id = 1, email = "test@example.com", passwordHash = "hashed",
            name = "Test User", role = "user",
            createdAt = LocalDateTime.now(), updatedAt = LocalDateTime.now(),
        )

        coEvery { userRepository.existsByEmail(any()) } returns false
        coEvery { userRepository.create(any(), any(), any(), any()) } returns user

        val result = userService.createUser(request)

        assertEquals("test@example.com", result.email)
        assertEquals("Test User", result.name)
        coVerify { userRepository.create(any(), any(), any(), any()) }
    }

    @Test
    fun `createUser with existing email throws ConflictException`() = runBlocking {
        coEvery { userRepository.existsByEmail(any()) } returns true
        assertFailsWith<ConflictException> {
            userService.createUser(CreateUserRequest("dup@test.com", "pass1234", "Dup"))
        }
    }

    @Test
    fun `getUserById with missing id throws NotFoundException`() = runBlocking {
        coEvery { userRepository.findById(any()) } returns null
        assertFailsWith<NotFoundException> {
            userService.getUserById(999)
        }
    }
}
```

### Test Configuration (H2 in-memory)

```hocon
# src/test/resources/application-test.conf
ktor {
    deployment { port = 0 }
    application { modules = [ com.example.ApplicationKt.module ] }
}
database {
    url = "jdbc:h2:mem:test;DB_CLOSE_DELAY=-1"
    driver = "org.h2.Driver"
    user = "sa"
    password = ""
    maxPoolSize = 5
}
jwt {
    secret = "test-secret-key-min-256-bits-long-for-hmac256"
    issuer = "test"
    audience = "test"
    realm = "test"
    expirationMs = 3600000
}
```

## Deployment

### Docker

```dockerfile
# Multi-stage build
FROM gradle:8-jdk17 AS build
WORKDIR /app
COPY build.gradle.kts settings.gradle.kts ./
COPY gradle ./gradle
RUN gradle dependencies --no-daemon
COPY src ./src
RUN gradle buildFatJar --no-daemon

FROM eclipse-temurin:17-jre-alpine
WORKDIR /app
COPY --from=build /app/build/libs/app.jar ./app.jar
EXPOSE 8080
ENTRYPOINT ["java", "-jar", "app.jar"]
```

### Docker Compose

```yaml
version: "3.8"
services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      PORT: 8080
      DATABASE_URL: jdbc:postgresql://db:5432/myapp
      DATABASE_USER: postgres
      DATABASE_PASSWORD: postgres
      JWT_SECRET: ${JWT_SECRET}
    depends_on:
      db:
        condition: service_healthy

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: myapp
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  pgdata:
```

### Production Checklist

- Build fat JAR: `./gradlew buildFatJar`
- Set `prettyPrint = false` in JSON configuration
- Configure connection pool size based on expected load
- Set proper CORS origins (never `anyHost()` in production)
- Use environment variables for all secrets
- Enable structured JSON logging (logback + logstash-encoder)
- Set JVM flags: `-Xmx512m -XX:+UseG1GC` (tune per workload)
- Health check endpoint at `/health` returning 200

## CORS and Call Logging

### CORS Configuration

```kotlin
install(CORS) {
    // Development: allow all
    anyHost()

    // Production: restrict to specific origins
    // allowHost("example.com", schemes = listOf("https"))
    // allowHost("app.example.com", schemes = listOf("https"))

    allowHeader(HttpHeaders.ContentType)
    allowHeader(HttpHeaders.Authorization)
    allowMethod(HttpMethod.Options)
    allowMethod(HttpMethod.Put)
    allowMethod(HttpMethod.Delete)
}
```

### Call Logging

```kotlin
install(CallLogging) {
    level = Level.INFO
    filter { call -> call.request.path().startsWith("/api") }
    format { call ->
        val status = call.response.status()
        val method = call.request.httpMethod.value
        val path = call.request.path()
        val duration = call.processingTimeMillis()
        "$method $path - $status (${duration}ms)"
    }
}
```

### Dependency Wiring in configureRouting

```kotlin
fun Application.configureRouting() {
    val config = environment.config
    val userRepository = UserRepository()
    val jwtUtils = JwtUtils(
        secret = config.property("jwt.secret").getString(),
        issuer = config.property("jwt.issuer").getString(),
        audience = config.property("jwt.audience").getString(),
        expirationMs = config.property("jwt.expirationMs").getString().toLong(),
    )
    val userService = UserService(userRepository, jwtUtils)

    routing {
        route("/api") {
            authRoutes(userService)
            userRoutes(userService)
        }
    }
}
```
