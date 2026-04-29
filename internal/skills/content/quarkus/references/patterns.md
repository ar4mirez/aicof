# Quarkus Patterns Reference

Detailed patterns, recipes, and advanced guidance for Quarkus development.
Load this file when you need deeper examples beyond the core SKILL.md.

---

## Reactive Patterns with Mutiny

### Uni and Multi Basics

`Uni<T>` represents a single asynchronous result (or failure). `Multi<T>` represents a stream of items.

```java
// Uni: single async value
Uni<User> user = User.findById(1L);

// Transform
Uni<String> name = user.map(u -> u.name);

// Chain dependent async operations
Uni<UserResponse> result = userRepository.findById(id)
    .onItem().ifNull().failWith(
        () -> new ResourceNotFoundException("User", "id", id))
    .flatMap(user -> enrichUser(user))
    .map(mapper::toResponse);

// Multi: stream of items
Multi<User> activeUsers = User.streamAll()
    .filter(u -> u.active);
```

### Combining Multiple Uni Operations

```java
// Execute in parallel and combine
Uni<Tuple2<User, List<Order>>> combined = Uni.combine().all()
    .unis(
        userRepository.findById(userId),
        orderRepository.findByUserId(userId)
    )
    .asTuple();

// Use combined result
Uni<UserDetailResponse> detail = combined.map(tuple -> {
    User user = tuple.getItem1();
    List<Order> orders = tuple.getItem2();
    return new UserDetailResponse(user.name, user.email, orders.size());
});
```

### Error Recovery

```java
// Recover with fallback
Uni<Config> config = configService.fetchRemoteConfig()
    .onFailure().recoverWithItem(Config.defaults());

// Retry with backoff
Uni<Response> response = httpClient.get("/api/data")
    .onFailure().retry()
    .withBackOff(Duration.ofMillis(100), Duration.ofSeconds(2))
    .atMost(3);

// Transform failure type
Uni<User> user = repository.findById(id)
    .onFailure(PersistenceException.class)
    .transform(e -> new ServiceException("Database error", e));
```

### Reactive Streams with Multi

```java
// Server-Sent Events endpoint
@GET
@Path("/stream")
@Produces(MediaType.SERVER_SENT_EVENTS)
@RestSseElementType(MediaType.APPLICATION_JSON)
public Multi<UserEvent> streamUserEvents() {
    return Multi.createFrom().ticks().every(Duration.ofSeconds(1))
        .onItem().transformToUniAndMerge(
            tick -> userRepository.findRecentlyActive())
        .select().distinct();
}

// Batch processing with Multi
public Uni<Void> processAllUsers() {
    return User.<User>streamAll()
        .group().intoLists().of(100)
        .onItem().transformToUniAndMerge(batch ->
            processUserBatch(batch))
        .collect().asList()
        .replaceWithVoid();
}
```

### Blocking Operations in Reactive Context

```java
// When you MUST call blocking code from reactive pipeline
@ApplicationScoped
public class PdfService {

    @Inject
    ManagedExecutor executor;

    public Uni<byte[]> generatePdf(ReportData data) {
        return Uni.createFrom().item(() -> blockingPdfGeneration(data))
            .runSubscriptionOn(executor);
    }

    private byte[] blockingPdfGeneration(ReportData data) {
        // CPU-bound or blocking library call
        return pdfLibrary.generate(data);
    }
}

// Or annotate the resource method directly
@GET
@Path("/report")
@Blocking  // Moves execution to worker thread pool
public byte[] getReport(@QueryParam("id") Long id) {
    return pdfService.generateBlocking(id);
}
```

---

## Hibernate Reactive with Panache

### Active Record Pattern (Simple CRUD)

```java
@Entity
@Table(name = "products")
@Cacheable  // Enable second-level cache
public class Product extends PanacheEntity {

    @Column(nullable = false)
    public String name;

    @Column(nullable = false)
    public BigDecimal price;

    @Column(nullable = false)
    @Enumerated(EnumType.STRING)
    public Category category;

    public boolean active = true;

    // Named queries for common operations
    public static Uni<List<Product>> findByCategory(Category cat) {
        return list("category = ?1 and active = true", cat);
    }

    public static Uni<List<Product>> findCheaperThan(BigDecimal maxPrice) {
        return list("price < ?1 and active = true",
            Sort.by("price"), maxPrice);
    }

    public static Uni<Long> countByCategory(Category cat) {
        return count("category", cat);
    }

    // Pagination
    public static Uni<List<Product>> findPaginated(int page, int size) {
        return findAll(Sort.by("name"))
            .page(Page.of(page, size))
            .list();
    }
}
```

### Repository Pattern (Complex Queries)

```java
@ApplicationScoped
public class OrderRepository implements PanacheRepository<Order> {

    public Uni<List<Order>> findByUserAndStatus(Long userId, OrderStatus status) {
        return list("userId = ?1 and status = ?2",
            Sort.by("createdAt").descending(), userId, status);
    }

    public Uni<List<Order>> searchOrders(OrderSearchCriteria criteria) {
        StringBuilder query = new StringBuilder("1=1");
        Map<String, Object> params = new HashMap<>();

        if (criteria.userId() != null) {
            query.append(" and userId = :userId");
            params.put("userId", criteria.userId());
        }
        if (criteria.status() != null) {
            query.append(" and status = :status");
            params.put("status", criteria.status());
        }
        if (criteria.fromDate() != null) {
            query.append(" and createdAt >= :fromDate");
            params.put("fromDate", criteria.fromDate());
        }

        return list(query.toString(), Sort.by("createdAt").descending(), params);
    }

    // Bulk update
    public Uni<Integer> cancelExpiredOrders(LocalDateTime cutoff) {
        return update("status = ?1 where status = ?2 and createdAt < ?3",
            OrderStatus.CANCELLED, OrderStatus.PENDING, cutoff);
    }
}
```

### Entity Relationships

```java
@Entity
@Table(name = "orders")
public class Order extends PanacheEntity {

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "user_id", nullable = false)
    public User user;

    @OneToMany(mappedBy = "order", cascade = CascadeType.ALL, orphanRemoval = true)
    public List<OrderItem> items = new ArrayList<>();

    @Column(nullable = false)
    @Enumerated(EnumType.STRING)
    public OrderStatus status = OrderStatus.PENDING;

    // Helper method for bidirectional relationship
    public void addItem(OrderItem item) {
        items.add(item);
        item.order = this;
    }

    public BigDecimal getTotal() {
        return items.stream()
            .map(item -> item.price.multiply(BigDecimal.valueOf(item.quantity)))
            .reduce(BigDecimal.ZERO, BigDecimal::add);
    }
}
```

### Flyway Migrations

```sql
-- V1__create_users.sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'USER',
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_active ON users(active);

-- V2__create_orders.sql
CREATE TABLE orders (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    total DECIMAL(10,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status);
```

---

## Testing Strategies

### Unit Test with Mocked Dependencies

```java
@QuarkusTest
@DisplayName("UserService")
class UserServiceTest {

    @Inject
    UserService userService;

    @InjectMock
    UserRepository userRepository;

    @InjectMock
    UserMapper userMapper;

    @Test
    @DisplayName("createUser throws DuplicateResourceException when email exists")
    void createUserThrowsWhenEmailExists() {
        var request = new UserRequest("taken@example.com", "Pass123!", "Name");
        when(userRepository.existsByEmail("taken@example.com"))
            .thenReturn(Uni.createFrom().item(true));

        assertThatThrownBy(() ->
            userService.createUser(request).await().indefinitely())
            .isInstanceOf(DuplicateResourceException.class)
            .hasMessageContaining("email");

        verify(userRepository, never()).persist(any(User.class));
    }

    @Test
    @DisplayName("getUserById returns mapped response when user found")
    void getUserByIdReturnsWhenFound() {
        var user = new User();
        user.id = 1L;
        user.email = "test@example.com";
        var expected = new UserResponse(1L, "test@example.com", "Test", "USER", true, null);

        when(userRepository.findById(1L)).thenReturn(Uni.createFrom().item(user));
        when(userMapper.toResponse(user)).thenReturn(expected);

        UserResponse result = userService.getUserById(1L).await().indefinitely();

        assertThat(result.email()).isEqualTo("test@example.com");
    }
}
```

### REST-assured Integration Test

```java
@QuarkusTest
@DisplayName("UserResource Integration")
@TestMethodOrder(MethodOrderer.OrderAnnotation.class)
class UserResourceIntegrationTest {

    private static Long createdUserId;

    @Test
    @Order(1)
    @DisplayName("POST creates user and returns 201")
    void createUser() {
        createdUserId = given()
            .contentType(ContentType.JSON)
            .body(new UserRequest("integration@test.com", "Password1!", "Test User"))
        .when()
            .post("/api/v1/users")
        .then()
            .statusCode(201)
            .header("Location", containsString("/api/v1/users/"))
            .body("email", equalTo("integration@test.com"))
            .body("name", equalTo("Test User"))
            .extract().jsonPath().getLong("id");
    }

    @Test
    @Order(2)
    @TestSecurity(user = "admin", roles = "ADMIN")
    @DisplayName("GET returns previously created user")
    void getUserById() {
        given()
        .when()
            .get("/api/v1/users/" + createdUserId)
        .then()
            .statusCode(200)
            .body("id", equalTo(createdUserId.intValue()))
            .body("email", equalTo("integration@test.com"));
    }

    @Test
    @DisplayName("POST returns 400 for invalid input")
    void createUserValidationFails() {
        given()
            .contentType(ContentType.JSON)
            .body(new UserRequest("", "", ""))
        .when()
            .post("/api/v1/users")
        .then()
            .statusCode(400)
            .body("title", equalTo("Validation Error"));
    }
}
```

### Testcontainers with QuarkusTestResource

```java
public class PostgresTestResource implements QuarkusTestResourceLifecycleManager {

    static PostgreSQLContainer<?> postgres =
        new PostgreSQLContainer<>("postgres:15-alpine");

    @Override
    public Map<String, String> start() {
        postgres.start();
        return Map.of(
            "quarkus.datasource.reactive.url",
                "vertx-reactive:postgresql://" + postgres.getHost()
                    + ":" + postgres.getMappedPort(5432) + "/" + postgres.getDatabaseName(),
            "quarkus.datasource.jdbc.url", postgres.getJdbcUrl(),
            "quarkus.datasource.username", postgres.getUsername(),
            "quarkus.datasource.password", postgres.getPassword()
        );
    }

    @Override
    public void stop() {
        postgres.stop();
    }
}

@QuarkusTest
@QuarkusTestResource(PostgresTestResource.class)
class UserRepositoryIntegrationTest {
    // Tests run against real PostgreSQL via Testcontainers
}
```

### Native Image Test

```java
@NativeImageTest
@DisplayName("UserResource Native")
class UserResourceNativeTest extends UserResourceTest {
    // Inherits all tests from UserResourceTest
    // Runs them against the native executable
}
```

---

## GraalVM Native Compilation

### Reflection Registration

```java
// Register classes that use reflection at runtime
@RegisterForReflection
public class ExternalLibraryDTO {
    public String field;
}

// Register for serialization (Jackson, etc.)
@RegisterForReflection(serialization = true)
public class SerializableEvent {
    public String type;
    public Map<String, Object> payload;
}
```

### Native Build Configuration

```properties
# application.properties
quarkus.native.additional-build-args=\
    --initialize-at-run-time=com.example.SomeClass,\
    -H:+ReportExceptionStackTraces

# Resource includes for native
quarkus.native.resources.includes=templates/**,META-INF/resources/**

# Enable HTTPS in native image
quarkus.ssl.native=true
```

### Dockerfile.native (Multi-Stage)

```dockerfile
FROM quay.io/quarkus/ubi-quarkus-mandrel-builder-image:jdk-21 AS build
COPY --chown=quarkus:quarkus mvnw /code/mvnw
COPY --chown=quarkus:quarkus .mvn /code/.mvn
COPY --chown=quarkus:quarkus pom.xml /code/
USER quarkus
WORKDIR /code
RUN ./mvnw -B org.apache.maven.plugins:maven-dependency-plugin:3.1.2:go-offline
COPY src /code/src
RUN ./mvnw package -Pnative -DskipTests

FROM quay.io/quarkus/quarkus-micro-image:2.0
WORKDIR /work/
COPY --from=build /code/target/*-runner /work/application
RUN chmod 775 /work
EXPOSE 8080
CMD ["./application", "-Dquarkus.http.host=0.0.0.0"]
```

---

## Messaging (SmallRye Reactive Messaging)

### Kafka Producer

```java
@ApplicationScoped
public class UserEventProducer {

    @Inject
    @Channel("user-events-out")
    Emitter<UserEvent> emitter;

    public Uni<Void> publishUserCreated(User user) {
        var event = new UserEvent("USER_CREATED", user.id, user.email);
        return Uni.createFrom().completionStage(
            emitter.send(Message.of(event)));
    }
}
```

### Kafka Consumer

```java
@ApplicationScoped
public class UserEventConsumer {

    private static final Logger LOG = Logger.getLogger(UserEventConsumer.class);

    @Inject
    NotificationService notificationService;

    @Incoming("user-events-in")
    public Uni<Void> processUserEvent(UserEvent event) {
        LOG.infof("Received event: %s for user %d", event.type(), event.userId());

        return switch (event.type()) {
            case "USER_CREATED" -> notificationService.sendWelcomeEmail(event.userId());
            case "USER_DEACTIVATED" -> notificationService.sendGoodbyeEmail(event.userId());
            default -> Uni.createFrom().voidItem();
        };
    }
}
```

### Kafka Configuration

```properties
# Outgoing channel
mp.messaging.outgoing.user-events-out.connector=smallrye-kafka
mp.messaging.outgoing.user-events-out.topic=user-events
mp.messaging.outgoing.user-events-out.value.serializer=io.quarkus.kafka.client.serialization.ObjectMapperSerializer

# Incoming channel
mp.messaging.incoming.user-events-in.connector=smallrye-kafka
mp.messaging.incoming.user-events-in.topic=user-events
mp.messaging.incoming.user-events-in.value.deserializer=com.example.serde.UserEventDeserializer
mp.messaging.incoming.user-events-in.group.id=user-service
mp.messaging.incoming.user-events-in.auto.offset.reset=earliest

# Dev Services (auto-starts Kafka in dev mode)
%dev.quarkus.kafka.devservices.enabled=true
```

---

## Security

### JWT Authentication

```java
@Path("/api/v1/auth")
@Produces(MediaType.APPLICATION_JSON)
@Consumes(MediaType.APPLICATION_JSON)
public class AuthResource {

    @ConfigProperty(name = "mp.jwt.verify.issuer")
    String issuer;

    @POST
    @Path("/login")
    public Uni<TokenResponse> login(@Valid LoginRequest request) {
        return User.findByEmail(request.email())
            .onItem().ifNull().failWith(
                () -> new AuthenticationException("Invalid credentials"))
            .flatMap(user -> {
                if (!BcryptUtil.matches(request.password(), user.password)) {
                    return Uni.createFrom().failure(
                        new AuthenticationException("Invalid credentials"));
                }
                String token = Jwt.issuer(issuer)
                    .upn(user.email)
                    .groups(Set.of(user.role.name()))
                    .claim("userId", user.id)
                    .expiresIn(Duration.ofHours(24))
                    .sign();
                return Uni.createFrom().item(
                    new TokenResponse(token, "Bearer", 86400L));
            });
    }
}
```

### Role-Based Access Control

```java
// Secure individual endpoints
@GET
@RolesAllowed({"ADMIN"})
@SecurityRequirement(name = "jwt")
public Uni<List<UserResponse>> adminOnlyEndpoint() { ... }

// Secure entire resource class
@Path("/api/v1/admin")
@Authenticated  // Any authenticated user
public class AdminResource { ... }

// Programmatic security check
@Inject
SecurityIdentity identity;

public Uni<UserResponse> getProfile() {
    String email = identity.getPrincipal().getName();
    return userService.getUserByEmail(email);
}
```

### CORS Configuration

```properties
quarkus.http.cors=true
quarkus.http.cors.origins=http://localhost:3000,https://myapp.com
quarkus.http.cors.methods=GET,POST,PUT,DELETE,PATCH,OPTIONS
quarkus.http.cors.headers=Content-Type,Authorization
quarkus.http.cors.exposed-headers=Location
quarkus.http.cors.access-control-max-age=24H
```

---

## Observability

### Health Checks

```java
@Liveness
@ApplicationScoped
public class AppLivenessCheck implements HealthCheck {
    @Override
    public HealthCheckResponse call() {
        return HealthCheckResponse.up("Application is alive");
    }
}

@Readiness
@ApplicationScoped
public class DatabaseReadinessCheck implements HealthCheck {

    @Inject
    UserRepository userRepository;

    @Override
    public HealthCheckResponse call() {
        try {
            userRepository.count().await().atMost(Duration.ofSeconds(5));
            return HealthCheckResponse.up("Database is reachable");
        } catch (Exception e) {
            return HealthCheckResponse.down("Database unreachable")
                .withData("error", e.getMessage()).build();
        }
    }
}
```

### Custom Metrics

```java
@ApplicationScoped
public class UserService {

    @Inject
    MeterRegistry registry;

    @Counted(value = "users.created", description = "Number of users created")
    @Timed(value = "users.creation.time", description = "Time to create a user")
    @WithTransaction
    public Uni<UserResponse> createUser(UserRequest request) {
        return userRepository.existsByEmail(request.email())
            .flatMap(exists -> {
                if (exists) {
                    registry.counter("users.creation.duplicates").increment();
                    return Uni.createFrom().failure(
                        new DuplicateResourceException("User", "email", request.email()));
                }
                // ... create user
            });
    }
}
```

### Logging Best Practices

```java
// Use JBoss Logging (built-in, zero overhead when disabled)
private static final Logger LOG = Logger.getLogger(UserService.class);

// Parameterized logging (no string concatenation if level disabled)
LOG.infof("Creating user with email: %s", request.email());
LOG.debugf("User %d found with %d orders", userId, orderCount);

// Structured logging in production
// application.properties:
// %prod.quarkus.log.console.json=true
// %prod.quarkus.log.console.json.additional-field."service".value=user-service
```

---

## Caching

### Application-Level Cache

```java
@ApplicationScoped
public class ProductService {

    @CacheResult(cacheName = "products-by-id")
    public Uni<Product> getProductById(@CacheKey Long id) {
        return Product.findById(id);
    }

    @CacheInvalidate(cacheName = "products-by-id")
    @WithTransaction
    public Uni<Product> updateProduct(@CacheKey Long id, ProductRequest request) {
        return Product.<Product>findById(id)
            .flatMap(product -> {
                product.name = request.name();
                product.price = request.price();
                return product.persist();
            });
    }

    @CacheInvalidateAll(cacheName = "products-by-id")
    public Uni<Void> clearProductCache() {
        return Uni.createFrom().voidItem();
    }
}
```

### Cache Configuration

```properties
# Caffeine cache configuration
quarkus.cache.caffeine."products-by-id".expire-after-write=10M
quarkus.cache.caffeine."products-by-id".maximum-size=1000

# Redis cache (distributed)
quarkus.cache.redis.expire-after-write=10M
```

---

## MapStruct Mapper Patterns

```java
@Mapper(componentModel = "cdi",
        unmappedTargetPolicy = ReportingPolicy.IGNORE,
        nullValuePropertyMappingStrategy = NullValuePropertyMappingStrategy.IGNORE)
public interface UserMapper {

    @Mapping(target = "id", ignore = true)
    @Mapping(target = "createdAt", ignore = true)
    @Mapping(target = "updatedAt", ignore = true)
    @Mapping(target = "active", constant = "true")
    @Mapping(target = "role", expression = "java(mapRole(request.role()))")
    User toEntity(UserRequest request);

    UserResponse toResponse(User user);

    List<UserResponse> toResponseList(List<User> users);

    // Partial update: only non-null fields overwrite
    @BeanMapping(nullValuePropertyMappingStrategy = NullValuePropertyMappingStrategy.IGNORE)
    @Mapping(target = "id", ignore = true)
    @Mapping(target = "password", ignore = true)
    void updateEntity(UserRequest request, @MappingTarget User user);

    default User.Role mapRole(String role) {
        if (role == null || role.isBlank()) return User.Role.USER;
        return User.Role.valueOf(role.toUpperCase());
    }
}
```

---

## Scheduled Tasks

```java
@ApplicationScoped
public class MaintenanceScheduler {

    private static final Logger LOG = Logger.getLogger(MaintenanceScheduler.class);

    @Inject
    OrderRepository orderRepository;

    // Cron expression: every day at 2 AM
    @Scheduled(cron = "0 0 2 * * ?", identity = "cancel-expired-orders")
    Uni<Void> cancelExpiredOrders() {
        LOG.info("Running expired order cancellation");
        LocalDateTime cutoff = LocalDateTime.now().minusDays(7);
        return orderRepository.cancelExpiredOrders(cutoff)
            .invoke(count -> LOG.infof("Cancelled %d expired orders", count))
            .replaceWithVoid();
    }

    // Every 30 seconds
    @Scheduled(every = "30s", identity = "health-ping")
    void healthPing() {
        LOG.debug("Health ping");
    }
}
```

---

## REST Client (Calling External APIs)

```java
@Path("/api/v1")
@RegisterRestClient(configKey = "payment-api")
@Produces(MediaType.APPLICATION_JSON)
@Consumes(MediaType.APPLICATION_JSON)
public interface PaymentClient {

    @POST
    @Path("/charges")
    Uni<ChargeResponse> createCharge(ChargeRequest request);

    @GET
    @Path("/charges/{id}")
    Uni<ChargeResponse> getCharge(@PathParam("id") String chargeId);
}

// Configuration
// application.properties:
// quarkus.rest-client.payment-api.url=https://api.payment.com
// quarkus.rest-client.payment-api.scope=jakarta.enterprise.context.ApplicationScoped

// Usage in service
@ApplicationScoped
public class PaymentService {

    @Inject
    @RestClient
    PaymentClient paymentClient;

    public Uni<ChargeResponse> chargeUser(Long userId, BigDecimal amount) {
        return paymentClient.createCharge(new ChargeRequest(userId, amount))
            .onFailure().retry()
            .withBackOff(Duration.ofMillis(200), Duration.ofSeconds(1))
            .atMost(3);
    }
}
```

---

## Common Dependencies (pom.xml snippets)

```xml
<!-- RESTEasy Reactive + Jackson -->
<dependency>
    <groupId>io.quarkus</groupId>
    <artifactId>quarkus-resteasy-reactive-jackson</artifactId>
</dependency>

<!-- Hibernate Reactive + Panache -->
<dependency>
    <groupId>io.quarkus</groupId>
    <artifactId>quarkus-hibernate-reactive-panache</artifactId>
</dependency>

<!-- Reactive PostgreSQL client -->
<dependency>
    <groupId>io.quarkus</groupId>
    <artifactId>quarkus-reactive-pg-client</artifactId>
</dependency>

<!-- Bean Validation -->
<dependency>
    <groupId>io.quarkus</groupId>
    <artifactId>quarkus-hibernate-validator</artifactId>
</dependency>

<!-- JWT Security -->
<dependency>
    <groupId>io.quarkus</groupId>
    <artifactId>quarkus-smallrye-jwt</artifactId>
</dependency>

<!-- Health + Metrics -->
<dependency>
    <groupId>io.quarkus</groupId>
    <artifactId>quarkus-smallrye-health</artifactId>
</dependency>
<dependency>
    <groupId>io.quarkus</groupId>
    <artifactId>quarkus-micrometer-registry-prometheus</artifactId>
</dependency>

<!-- OpenAPI -->
<dependency>
    <groupId>io.quarkus</groupId>
    <artifactId>quarkus-smallrye-openapi</artifactId>
</dependency>

<!-- Flyway + JDBC (for migrations) -->
<dependency>
    <groupId>io.quarkus</groupId>
    <artifactId>quarkus-flyway</artifactId>
</dependency>
<dependency>
    <groupId>io.quarkus</groupId>
    <artifactId>quarkus-jdbc-postgresql</artifactId>
</dependency>

<!-- Kafka Messaging -->
<dependency>
    <groupId>io.quarkus</groupId>
    <artifactId>quarkus-smallrye-reactive-messaging-kafka</artifactId>
</dependency>

<!-- Caching -->
<dependency>
    <groupId>io.quarkus</groupId>
    <artifactId>quarkus-cache</artifactId>
</dependency>

<!-- REST Client -->
<dependency>
    <groupId>io.quarkus</groupId>
    <artifactId>quarkus-rest-client-reactive-jackson</artifactId>
</dependency>

<!-- Scheduler -->
<dependency>
    <groupId>io.quarkus</groupId>
    <artifactId>quarkus-scheduler</artifactId>
</dependency>

<!-- Testing -->
<dependency>
    <groupId>io.quarkus</groupId>
    <artifactId>quarkus-junit5</artifactId>
    <scope>test</scope>
</dependency>
<dependency>
    <groupId>io.rest-assured</groupId>
    <artifactId>rest-assured</artifactId>
    <scope>test</scope>
</dependency>
<dependency>
    <groupId>io.quarkus</groupId>
    <artifactId>quarkus-test-security</artifactId>
    <scope>test</scope>
</dependency>
<dependency>
    <groupId>io.quarkus</groupId>
    <artifactId>quarkus-junit5-mockito</artifactId>
    <scope>test</scope>
</dependency>
```
