# Spring Boot (Kotlin) Patterns Reference

## Contents

- [Dependencies Setup](#dependencies-setup)
- [WebFlux Reactive Patterns](#webflux-reactive-patterns)
- [JPA Advanced Patterns](#jpa-advanced-patterns)
- [Coroutines with Spring](#coroutines-with-spring)
- [Security Advanced Patterns](#security-advanced-patterns)
- [Testing Patterns](#testing-patterns)
- [Configuration Patterns](#configuration-patterns)

## Dependencies Setup

### Full build.gradle.kts

```kotlin
import org.jetbrains.kotlin.gradle.tasks.KotlinCompile

plugins {
    id("org.springframework.boot") version "3.2.0"
    id("io.spring.dependency-management") version "1.1.4"
    kotlin("jvm") version "1.9.21"
    kotlin("plugin.spring") version "1.9.21"
    kotlin("plugin.jpa") version "1.9.21"
}

group = "com.example"
version = "0.0.1-SNAPSHOT"

java {
    sourceCompatibility = JavaVersion.VERSION_21
}

repositories {
    mavenCentral()
}

dependencies {
    // Spring Boot
    implementation("org.springframework.boot:spring-boot-starter-web")
    implementation("org.springframework.boot:spring-boot-starter-validation")
    implementation("org.springframework.boot:spring-boot-starter-security")
    implementation("org.springframework.boot:spring-boot-starter-data-jpa")

    // Kotlin
    implementation("com.fasterxml.jackson.module:jackson-module-kotlin")
    implementation("org.jetbrains.kotlin:kotlin-reflect")

    // Coroutines (optional but recommended)
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-core")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-reactor")

    // Database
    runtimeOnly("org.postgresql:postgresql")

    // JWT
    implementation("io.jsonwebtoken:jjwt-api:0.12.3")
    runtimeOnly("io.jsonwebtoken:jjwt-impl:0.12.3")
    runtimeOnly("io.jsonwebtoken:jjwt-jackson:0.12.3")

    // Development
    developmentOnly("org.springframework.boot:spring-boot-devtools")

    // Testing
    testImplementation("org.springframework.boot:spring-boot-starter-test")
    testImplementation("org.springframework.security:spring-security-test")
    testImplementation("io.mockk:mockk:1.13.8")
    testImplementation("com.ninja-squad:springmockk:4.0.2")
}

tasks.withType<KotlinCompile> {
    kotlinOptions {
        freeCompilerArgs += "-Xjsr305=strict"
        jvmTarget = "21"
    }
}

tasks.withType<Test> {
    useJUnitPlatform()
}
```

### WebFlux Dependencies (Reactive)

Replace `spring-boot-starter-web` with:

```kotlin
implementation("org.springframework.boot:spring-boot-starter-webflux")
implementation("org.jetbrains.kotlinx:kotlinx-coroutines-reactor")
implementation("io.projectreactor.kotlin:reactor-kotlin-extensions")

// R2DBC instead of JPA for non-blocking DB
implementation("org.springframework.boot:spring-boot-starter-data-r2dbc")
runtimeOnly("org.postgresql:r2dbc-postgresql")
```

## WebFlux Reactive Patterns

### Coroutine-Based Controller (WebFlux)

```kotlin
@RestController
@RequestMapping("/api/users")
class UserController(
    private val userService: UserService,
) {
    @GetMapping("/{id}")
    suspend fun getUser(@PathVariable id: UUID): ResponseEntity<UserResponse> {
        val user = userService.findById(id)
            ?: return ResponseEntity.notFound().build()
        return ResponseEntity.ok(UserResponse.from(user))
    }

    @GetMapping
    fun getAllUsers(): Flow<UserResponse> {
        return userService.findAll().map { UserResponse.from(it) }
    }

    @PostMapping
    suspend fun createUser(
        @Valid @RequestBody request: CreateUserRequest,
    ): ResponseEntity<UserResponse> {
        val user = userService.create(request)
        return ResponseEntity.status(HttpStatus.CREATED).body(user)
    }

    @DeleteMapping("/{id}")
    @ResponseStatus(HttpStatus.NO_CONTENT)
    suspend fun deleteUser(@PathVariable id: UUID) {
        userService.delete(id)
    }
}
```

### Reactive Service with Coroutines

```kotlin
@Service
class UserService(
    private val userRepository: UserRepository,
    private val passwordEncoder: PasswordEncoder,
) {
    suspend fun findById(id: UUID): User? {
        return userRepository.findById(id)
    }

    fun findAll(): Flow<User> {
        return userRepository.findAll().asFlow()
    }

    @Transactional
    suspend fun create(request: CreateUserRequest): UserResponse {
        val existing = userRepository.findByEmail(request.email)
        if (existing != null) {
            throw ConflictException("Email already registered")
        }

        val user = User(
            email = request.email,
            passwordHash = passwordEncoder.encode(request.password),
            name = request.name,
        )
        val saved = userRepository.save(user)
        return UserResponse.from(saved)
    }

    @Transactional
    suspend fun delete(id: UUID) {
        if (!userRepository.existsById(id)) {
            throw NotFoundException("User not found: $id")
        }
        userRepository.deleteById(id)
    }
}
```

### R2DBC Coroutine Repository

```kotlin
interface UserRepository : CoroutineCrudRepository<User, UUID> {
    suspend fun findByEmail(email: String): User?
    suspend fun existsByEmail(email: String): Boolean
}
```

### Server-Sent Events with Flow

```kotlin
@GetMapping("/stream", produces = [MediaType.TEXT_EVENT_STREAM_VALUE])
fun streamUpdates(): Flow<ServerSentEvent<NotificationDto>> = flow {
    notificationService.observeUpdates().collect { notification ->
        emit(
            ServerSentEvent.builder<NotificationDto>()
                .id(notification.id.toString())
                .event("notification")
                .data(NotificationDto.from(notification))
                .build()
        )
    }
}
```

### WebClient with Coroutines

```kotlin
@Service
class ExternalApiService(
    private val webClient: WebClient,
) {
    suspend fun fetchExternalData(id: String): ExternalData {
        return webClient.get()
            .uri("/api/data/{id}", id)
            .retrieve()
            .awaitBody<ExternalData>()
    }

    suspend fun fetchWithTimeout(id: String): ExternalData? {
        return withTimeoutOrNull(5_000) {
            webClient.get()
                .uri("/api/data/{id}", id)
                .retrieve()
                .awaitBodyOrNull<ExternalData>()
        }
    }

    fun fetchAllPaginated(pageSize: Int = 50): Flow<ExternalData> = flow {
        var page = 0
        do {
            val response = webClient.get()
                .uri { it.path("/api/data").queryParam("page", page).queryParam("size", pageSize).build() }
                .retrieve()
                .awaitBody<PagedResponse<ExternalData>>()
            response.content.forEach { emit(it) }
            page++
        } while (response.hasNext)
    }
}
```

## JPA Advanced Patterns

### Custom Query Methods

```kotlin
@Repository
interface UserRepository : JpaRepository<User, UUID> {
    fun findByEmail(email: String): User?
    fun existsByEmail(email: String): Boolean

    @Query("SELECT u FROM User u WHERE u.role = :role AND u.createdAt > :since")
    fun findActiveByRole(
        @Param("role") role: String,
        @Param("since") since: Instant,
        pageable: Pageable,
    ): Page<User>

    @Query("SELECT u FROM User u WHERE LOWER(u.name) LIKE LOWER(CONCAT('%', :query, '%'))")
    fun searchByName(@Param("query") query: String): List<User>

    @Modifying
    @Query("UPDATE User u SET u.role = :role WHERE u.id = :id")
    fun updateRole(@Param("id") id: UUID, @Param("role") role: String): Int
}
```

### Specification for Dynamic Queries

```kotlin
object UserSpecifications {
    fun hasRole(role: String): Specification<User> =
        Specification { root, _, cb -> cb.equal(root.get<String>("role"), role) }

    fun nameLike(name: String): Specification<User> =
        Specification { root, _, cb -> cb.like(cb.lower(root.get("name")), "%${name.lowercase()}%") }

    fun createdAfter(date: Instant): Specification<User> =
        Specification { root, _, cb -> cb.greaterThan(root.get("createdAt"), date) }
}

// Usage in service
@Service
class UserService(
    private val userRepository: UserRepository, // extends JpaSpecificationExecutor<User>
) {
    fun searchUsers(role: String?, name: String?, since: Instant?): List<User> {
        var spec = Specification.where<User>(null)
        role?.let { spec = spec.and(UserSpecifications.hasRole(it)) }
        name?.let { spec = spec.and(UserSpecifications.nameLike(it)) }
        since?.let { spec = spec.and(UserSpecifications.createdAfter(it)) }
        return userRepository.findAll(spec)
    }
}
```

### Auditing with Spring Data

```kotlin
@Configuration
@EnableJpaAuditing
class JpaAuditConfig

@MappedSuperclass
@EntityListeners(AuditingEntityListener::class)
abstract class AuditableEntity(
    @CreatedDate
    @Column(updatable = false)
    var createdAt: Instant? = null,

    @LastModifiedDate
    var updatedAt: Instant? = null,

    @CreatedBy
    @Column(updatable = false)
    var createdBy: String? = null,

    @LastModifiedBy
    var updatedBy: String? = null,
)

@Component
class AuditorAwareImpl : AuditorAware<String> {
    override fun getCurrentAuditor(): Optional<String> {
        return Optional.ofNullable(SecurityContextHolder.getContext().authentication?.name)
    }
}
```

### Projection DTOs

```kotlin
// Interface-based projection (Spring Data generates implementation)
interface UserSummary {
    val id: UUID
    val name: String
    val email: String
}

@Repository
interface UserRepository : JpaRepository<User, UUID> {
    fun findAllProjectedBy(pageable: Pageable): Page<UserSummary>

    @Query("SELECT u.id as id, u.name as name, u.email as email FROM User u WHERE u.role = :role")
    fun findSummariesByRole(@Param("role") role: String): List<UserSummary>
}
```

### Entity Relationships

```kotlin
@Entity
@Table(name = "orders")
data class Order(
    @Id
    @GeneratedValue(strategy = GenerationType.UUID)
    val id: UUID? = null,

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "user_id", nullable = false)
    val user: User,

    @OneToMany(mappedBy = "order", cascade = [CascadeType.ALL], orphanRemoval = true)
    val items: MutableList<OrderItem> = mutableListOf(),

    @Enumerated(EnumType.STRING)
    val status: OrderStatus = OrderStatus.DRAFT,
) {
    fun addItem(item: OrderItem) {
        items.add(item)
    }
}

// Always use FetchType.LAZY for collections to prevent N+1
// Use @EntityGraph or JOIN FETCH for eager loading when needed
@Repository
interface OrderRepository : JpaRepository<Order, UUID> {
    @EntityGraph(attributePaths = ["items", "user"])
    fun findWithItemsById(id: UUID): Order?
}
```

## Coroutines with Spring

### Async Service Operations

```kotlin
@Service
class NotificationService(
    private val emailClient: EmailClient,
    private val pushClient: PushClient,
    private val smsClient: SmsClient,
) {
    suspend fun notifyUser(userId: UUID, message: String) = coroutineScope {
        val emailJob = async { emailClient.send(userId, message) }
        val pushJob = async { pushClient.send(userId, message) }
        val smsJob = async { smsClient.send(userId, message) }

        // Wait for all, collect results
        val results = awaitAll(emailJob, pushJob, smsJob)
        results.forEach { result ->
            if (result.isFailure) {
                logger.warn("Notification channel failed: ${result.error}")
            }
        }
    }
}
```

### Scheduled Tasks with Coroutines

```kotlin
@Component
class ScheduledTasks(
    private val userService: UserService,
    private val scope: CoroutineScope,
) {
    @Scheduled(fixedRate = 3600000) // every hour
    fun cleanupExpiredSessions() {
        scope.launch {
            try {
                val count = userService.cleanupExpiredSessions()
                logger.info("Cleaned up $count expired sessions")
            } catch (e: Exception) {
                logger.error("Failed to cleanup sessions", e)
            }
        }
    }
}

@Configuration
class CoroutineConfig {
    @Bean
    fun applicationScope(): CoroutineScope {
        return CoroutineScope(SupervisorJob() + Dispatchers.Default)
    }
}
```

## Security Advanced Patterns

### JWT Service

```kotlin
@Service
class JwtService(
    private val jwtProperties: JwtProperties,
) {
    private val secretKey: SecretKey = Keys.hmacShaKeyFor(jwtProperties.secret.toByteArray())

    fun generateToken(user: User): String {
        val now = Date()
        val expiration = Date(now.time + jwtProperties.expiration)

        return Jwts.builder()
            .subject(user.id.toString())
            .claim("email", user.email)
            .claim("role", user.role)
            .issuedAt(now)
            .expiration(expiration)
            .signWith(secretKey)
            .compact()
    }

    fun validateToken(token: String): Boolean = try {
        val claims = extractAllClaims(token)
        !claims.expiration.before(Date())
    } catch (e: Exception) {
        false
    }

    fun extractUserId(token: String): UUID =
        UUID.fromString(extractAllClaims(token).subject)

    fun extractRole(token: String): String =
        extractAllClaims(token)["role"] as String

    private fun extractAllClaims(token: String): Claims =
        Jwts.parser()
            .verifyWith(secretKey)
            .build()
            .parseSignedClaims(token)
            .payload
}
```

### JWT Authentication Filter

```kotlin
@Component
class JwtAuthenticationFilter(
    private val jwtService: JwtService,
) : OncePerRequestFilter() {

    override fun doFilterInternal(
        request: HttpServletRequest,
        response: HttpServletResponse,
        filterChain: FilterChain,
    ) {
        val authHeader = request.getHeader("Authorization")

        if (authHeader == null || !authHeader.startsWith("Bearer ")) {
            filterChain.doFilter(request, response)
            return
        }

        val token = authHeader.substring(7)

        if (jwtService.validateToken(token)) {
            val userId = jwtService.extractUserId(token)
            val role = jwtService.extractRole(token)
            val authorities = listOf(SimpleGrantedAuthority("ROLE_$role"))
            val authentication = UsernamePasswordAuthenticationToken(userId, null, authorities)
            SecurityContextHolder.getContext().authentication = authentication
        }

        filterChain.doFilter(request, response)
    }
}
```

### Method-Level Security

```kotlin
@RestController
@RequestMapping("/api/admin")
@PreAuthorize("hasRole('ADMIN')")
class AdminController(
    private val adminService: AdminService,
) {
    @GetMapping("/users")
    fun listUsers(pageable: Pageable) = adminService.listUsers(pageable)

    @DeleteMapping("/users/{id}")
    @PreAuthorize("hasRole('SUPER_ADMIN')")
    fun deleteUser(@PathVariable id: UUID) = adminService.deleteUser(id)
}

// Custom security expression
@Component("userSecurity")
class UserSecurity {
    fun isOwner(userId: UUID, authentication: Authentication): Boolean {
        val principal = authentication.principal
        return when (principal) {
            is UUID -> principal == userId
            is String -> principal == userId.toString()
            else -> false
        }
    }
}

// Usage: @PreAuthorize("hasRole('ADMIN') or @userSecurity.isOwner(#id, authentication)")
```

### CORS Configuration

```kotlin
@Configuration
class WebConfig : WebMvcConfigurer {
    override fun addCorsMappings(registry: CorsRegistry) {
        registry.addMapping("/api/**")
            .allowedOrigins("http://localhost:3000")
            .allowedMethods("GET", "POST", "PUT", "DELETE", "PATCH")
            .allowedHeaders("*")
            .allowCredentials(true)
            .maxAge(3600)
    }
}
```

## Testing Patterns

### Unit Test with MockK

```kotlin
@ExtendWith(MockKExtension::class)
class UserServiceTest {
    @MockK
    lateinit var userRepository: UserRepository

    @MockK
    lateinit var passwordEncoder: PasswordEncoder

    @MockK
    lateinit var jwtService: JwtService

    @InjectMockKs
    lateinit var userService: UserService

    private val testUser = User(
        id = UUID.randomUUID(),
        email = "test@example.com",
        passwordHash = "hashedPassword",
        name = "Test User",
        role = "USER",
        createdAt = Instant.now(),
        updatedAt = Instant.now(),
    )

    @BeforeEach
    fun setUp() { clearAllMocks() }

    @Test
    fun `createUser should create user successfully`() {
        val request = CreateUserRequest(
            email = "new@example.com",
            password = "password123",
            name = "New User",
        )

        every { userRepository.existsByEmail(request.email) } returns false
        every { passwordEncoder.encode(request.password) } returns "hashedPassword"
        every { userRepository.save(any()) } returns testUser.copy(
            email = request.email,
            name = request.name,
        )

        val result = userService.createUser(request)

        assertNotNull(result)
        verify { userRepository.save(any()) }
    }

    @Test
    fun `createUser should throw ConflictException when email exists`() {
        val request = CreateUserRequest(
            email = "existing@example.com",
            password = "password123",
            name = "User",
        )

        every { userRepository.existsByEmail(request.email) } returns true

        assertThrows<ConflictException> { userService.createUser(request) }
    }

    @Test
    fun `login should return token for valid credentials`() {
        val request = LoginRequest(email = "test@example.com", password = "password123")

        every { userRepository.findByEmail(request.email) } returns testUser
        every { passwordEncoder.matches(request.password, testUser.passwordHash) } returns true
        every { jwtService.generateToken(testUser) } returns "jwt-token"

        val result = userService.login(request)

        assertEquals("jwt-token", result.token)
        assertNotNull(result.user)
    }

    @Test
    fun `login should throw UnauthorizedException for invalid password`() {
        val request = LoginRequest(email = "test@example.com", password = "wrongpassword")

        every { userRepository.findByEmail(request.email) } returns testUser
        every { passwordEncoder.matches(request.password, testUser.passwordHash) } returns false

        assertThrows<UnauthorizedException> { userService.login(request) }
    }
}
```

### Integration Test with MockMvc

```kotlin
@SpringBootTest
@AutoConfigureMockMvc
@ActiveProfiles("test")
@Transactional
class UserControllerIntegrationTest {
    @Autowired
    lateinit var mockMvc: MockMvc

    @Autowired
    lateinit var objectMapper: ObjectMapper

    @Test
    fun `register should create new user`() {
        val request = CreateUserRequest(
            email = "newuser@example.com",
            password = "password123",
            name = "New User",
        )

        mockMvc.post("/api/register") {
            contentType = MediaType.APPLICATION_JSON
            content = objectMapper.writeValueAsString(request)
        }.andExpect {
            status { isCreated() }
            jsonPath("$.email") { value("newuser@example.com") }
            jsonPath("$.name") { value("New User") }
        }
    }

    @Test
    fun `register should fail with invalid email`() {
        val request = CreateUserRequest(
            email = "invalid-email",
            password = "password123",
            name = "User",
        )

        mockMvc.post("/api/register") {
            contentType = MediaType.APPLICATION_JSON
            content = objectMapper.writeValueAsString(request)
        }.andExpect {
            status { isBadRequest() }
        }
    }

    @Test
    fun `getCurrentUser should require authentication`() {
        mockMvc.get("/api/users/me")
            .andExpect { status { isUnauthorized() } }
    }
}
```

### Testing with Security Context

```kotlin
@Test
@WithMockUser(roles = ["ADMIN"])
fun `admin can list all users`() {
    mockMvc.get("/api/users") {
        param("page", "0")
        param("size", "10")
    }.andExpect {
        status { isOk() }
        jsonPath("$.content") { isArray() }
    }
}

@Test
fun `authenticated user can access own profile`() {
    val userId = UUID.randomUUID()

    mockMvc.get("/api/users/me") {
        with(SecurityMockMvcRequestPostProcessors.user(userId.toString()).roles("USER"))
    }.andExpect {
        status { isOk() }
    }
}
```

### Testcontainers for Database Integration

```kotlin
@SpringBootTest
@Testcontainers
@ActiveProfiles("test")
class DatabaseIntegrationTest {
    companion object {
        @Container
        @JvmStatic
        val postgres = PostgreSQLContainer("postgres:16-alpine")
            .withDatabaseName("testdb")
            .withUsername("test")
            .withPassword("test")

        @DynamicPropertySource
        @JvmStatic
        fun configureProperties(registry: DynamicPropertyRegistry) {
            registry.add("spring.datasource.url", postgres::getJdbcUrl)
            registry.add("spring.datasource.username", postgres::getUsername)
            registry.add("spring.datasource.password", postgres::getPassword)
        }
    }

    @Autowired
    lateinit var userRepository: UserRepository

    @Test
    @Transactional
    fun `should persist and retrieve user`() {
        val user = User(
            email = "test@example.com",
            passwordHash = "hashed",
            name = "Test User",
        )

        val saved = userRepository.save(user)
        val found = userRepository.findByEmail("test@example.com")

        assertNotNull(found)
        assertEquals(saved.id, found?.id)
        assertEquals("Test User", found?.name)
    }
}
```

## Configuration Patterns

### Multi-Profile Configuration

```yaml
# application.yml (shared defaults)
spring:
  application:
    name: myproject
  jpa:
    hibernate:
      ddl-auto: validate
    show-sql: false
server:
  port: 8080

---
# application-dev.yml
spring:
  datasource:
    url: jdbc:postgresql://localhost:5432/mydb_dev
    username: ${DB_USERNAME:postgres}
    password: ${DB_PASSWORD:postgres}
  jpa:
    show-sql: true
logging:
  level:
    com.example: DEBUG

---
# application-test.yml
spring:
  datasource:
    url: jdbc:h2:mem:testdb
    driver-class-name: org.h2.Driver
  jpa:
    hibernate:
      ddl-auto: create-drop

---
# application-prod.yml
spring:
  datasource:
    url: ${DATABASE_URL}
    username: ${DB_USERNAME}
    password: ${DB_PASSWORD}
    hikari:
      maximum-pool-size: 20
      minimum-idle: 5
server:
  port: ${PORT:8080}
```

### Health and Actuator Endpoints

```kotlin
@RestController
@RequestMapping("/health")
class HealthController {
    @GetMapping
    fun health() = mapOf(
        "status" to "ok",
        "timestamp" to System.currentTimeMillis(),
    )
}

// For Spring Boot Actuator (add dependency: spring-boot-starter-actuator)
// application.yml
// management:
//   endpoints:
//     web:
//       exposure:
//         include: health,info,metrics
//   endpoint:
//     health:
//       show-details: when_authorized
```

### Bean Configuration with Kotlin DSL

```kotlin
@Configuration
class AppConfig {
    @Bean
    fun objectMapper(): ObjectMapper = jacksonObjectMapper().apply {
        registerModule(JavaTimeModule())
        disable(SerializationFeature.WRITE_DATES_AS_TIMESTAMPS)
        setSerializationInclusion(JsonInclude.Include.NON_NULL)
    }

    @Bean
    fun webClient(): WebClient = WebClient.builder()
        .baseUrl("https://api.example.com")
        .defaultHeader(HttpHeaders.CONTENT_TYPE, MediaType.APPLICATION_JSON_VALUE)
        .build()
}
```

### Event Publishing

```kotlin
// Domain event
data class UserCreatedEvent(val userId: UUID, val email: String)

// Publisher
@Service
class UserService(
    private val userRepository: UserRepository,
    private val eventPublisher: ApplicationEventPublisher,
) {
    @Transactional
    fun createUser(request: CreateUserRequest): UserResponse {
        val user = userRepository.save(/* ... */)
        eventPublisher.publishEvent(UserCreatedEvent(user.id!!, user.email))
        return UserResponse.from(user)
    }
}

// Listener
@Component
class UserEventListener {
    @EventListener
    fun onUserCreated(event: UserCreatedEvent) {
        logger.info("User created: ${event.userId}")
    }

    @Async
    @EventListener
    fun sendWelcomeEmail(event: UserCreatedEvent) {
        // Non-blocking: runs in separate thread
        emailService.sendWelcome(event.email)
    }

    @TransactionalEventListener(phase = TransactionPhase.AFTER_COMMIT)
    fun afterUserCommitted(event: UserCreatedEvent) {
        // Only runs after transaction commits successfully
        analyticsService.trackSignup(event.userId)
    }
}
```
