# Micronaut Patterns Reference

## Contents

- [GraalVM Native Image](#graalvm-native-image)
- [Messaging & Events](#messaging--events)
- [Security](#security)
- [HTTP Client](#http-client)
- [Caching](#caching)
- [Scheduling](#scheduling)
- [Testing Patterns](#testing-patterns)
- [Deployment](#deployment)
- [Build Configuration](#build-configuration)

## GraalVM Native Image

### Build Configuration (build.gradle)

```groovy
plugins {
    id("io.micronaut.application") version "4.2.1"
    id("io.micronaut.aot") version "4.2.1"
}

micronaut {
    runtime("netty")
    processing {
        incremental(true)
        annotations("com.example.*")
    }
    aot {
        optimizeServiceLoading = false
        convertYamlToJava = false
        precomputeOperations = true
        cacheEnvironment = true
        optimizeClassLoading = true
        deduceEnvironment = true
        optimizeNetty = true
    }
}

graalvmNative.toolchainDetection = false
```

### Reflection Configuration

When using libraries that require reflection (uncommon in Micronaut), provide GraalVM metadata:

```json
// src/main/resources/META-INF/native-image/reflect-config.json
[
  {
    "name": "com.example.domain.User",
    "allDeclaredConstructors": true,
    "allDeclaredMethods": true,
    "allDeclaredFields": true
  }
]
```

### Native Image Best Practices

- Prefer Micronaut's compile-time processing over reflection-based libraries
- Use `@Serdeable` instead of Jackson annotations where possible
- Use `@Introspected` to generate bean introspection data at compile time
- Test native image builds in CI (they can fail on libraries using reflection)
- Use `@ReflectiveAccess` only as a last resort for third-party classes

```java
// Mark classes for compile-time introspection (avoids reflection)
@Introspected
public record ExternalDTO(String name, int value) {}
```

### Building Native Images

```bash
# Local build (requires GraalVM + native-image)
./gradlew nativeCompile
./build/native/nativeCompile/myapp

# Docker-based native build (no local GraalVM needed)
./gradlew dockerBuildNative

# Test native image
./gradlew nativeTest
```

## Messaging & Events

### Application Events

```java
// Publishing events
@Singleton
@RequiredArgsConstructor
public class UserService {
    private final ApplicationEventPublisher<UserCreatedEvent> eventPublisher;

    public UserResponse createUser(UserRequest request) {
        User user = userRepository.save(toEntity(request));
        eventPublisher.publishEvent(new UserCreatedEvent(user.getId(), user.getEmail()));
        return toResponse(user);
    }
}

// Event record
@Serdeable
public record UserCreatedEvent(Long userId, String email) {}

// Listening to events
@Singleton
public class UserEventListener {

    @EventListener
    @Async
    public void onUserCreated(UserCreatedEvent event) {
        // Send welcome email, update analytics, etc.
        log.info("User created: {}", event.userId());
    }
}
```

### Kafka Integration

```java
// Producer
@KafkaClient
public interface UserEventProducer {

    @Topic("user-events")
    void sendUserEvent(@KafkaKey String key, UserCreatedEvent event);
}

// Consumer
@KafkaListener(groupId = "user-service")
public class UserEventConsumer {

    @Topic("user-events")
    public void receiveUserEvent(@KafkaKey String key, UserCreatedEvent event) {
        log.info("Received user event: {}", event);
    }
}
```

### RabbitMQ Integration

```java
// Producer
@RabbitClient
public interface NotificationProducer {

    @Binding("notifications")
    void sendNotification(NotificationMessage message);
}

// Consumer
@RabbitListener
public class NotificationConsumer {

    @Queue("notifications")
    public void receiveNotification(NotificationMessage message) {
        log.info("Notification received: {}", message);
    }
}
```

## Security

### JWT Authentication Provider

```java
@Singleton
@RequiredArgsConstructor
public class AuthenticationProviderUserPassword
    implements HttpRequestAuthenticationProvider<HttpRequest<?>> {

    private final UserRepository userRepository;
    private final BCryptPasswordEncoder passwordEncoder;

    @Override
    public AuthenticationResponse authenticate(
            HttpRequest<?> httpRequest,
            AuthenticationRequest<String, String> authRequest) {

        User user = userRepository.findByEmail(authRequest.getIdentity())
            .orElseThrow(() -> new AuthenticationException(
                new AuthenticationFailed("Invalid credentials")));

        if (!passwordEncoder.matches(authRequest.getSecret(), user.getPassword())) {
            throw new AuthenticationException(new AuthenticationFailed("Invalid credentials"));
        }

        if (!user.getActive()) {
            throw new AuthenticationException(new AuthenticationFailed("Account deactivated"));
        }

        return AuthenticationResponse.success(
            user.getEmail(),
            List.of(user.getRole()),
            Map.of("userId", user.getId())
        );
    }
}
```

### Role-Based Access Control

```java
// Controller-level security
@Controller("/api/admin")
@Secured({"ROLE_ADMIN"})
public class AdminController { }

// Method-level security
@Get("/reports")
@Secured({"ROLE_ADMIN", "ROLE_MANAGER"})
public List<Report> getReports() { }

// Access current principal
@Get("/me")
public UserResponse getCurrentUser(Authentication authentication) {
    String email = authentication.getName();
    Long userId = (Long) authentication.getAttributes().get("userId");
    return userService.getUserByEmail(email);
}
```

### Security Configuration (application.yml)

```yaml
micronaut:
  security:
    authentication: bearer
    token:
      jwt:
        signatures:
          secret:
            generator:
              secret: ${JWT_SECRET}
              jws-algorithm: HS256
        generator:
          access-token:
            expiration: 3600
          refresh-token:
            enabled: true
            secret: ${JWT_REFRESH_SECRET}
    intercept-url-map:
      - pattern: /health/**
        http-method: GET
        access: [isAnonymous()]
      - pattern: /api/auth/**
        http-method: POST
        access: [isAnonymous()]
      - pattern: /swagger/**
        access: [isAnonymous()]
      - pattern: /api/**
        access: [isAuthenticated()]
```

## HTTP Client

### Declarative HTTP Client

```java
@Client("${services.user-service.url}")
public interface UserClient {

    @Get("/api/users/{id}")
    UserResponse getUserById(@PathVariable Long id);

    @Post("/api/users")
    @Header(name = "X-Request-Id", value = "${request.id}")
    UserResponse createUser(@Body UserRequest request);

    @Get("/api/users")
    PageResponse<UserResponse> listUsers(
        @QueryValue int page,
        @QueryValue int size);
}
```

### Resilience with Retry and Circuit Breaker

```java
@Client("${services.external-api.url}")
@Retryable(attempts = "3", delay = "500ms", multiplier = "1.5")
public interface ExternalApiClient {

    @Get("/data/{id}")
    @CircuitBreaker(delay = "30s", attempts = "5", reset = "120s")
    ExternalData getData(@PathVariable String id);
}
```

### Client Configuration

```yaml
services:
  user-service:
    url: http://localhost:8081
  external-api:
    url: https://api.external.com

micronaut:
  http:
    client:
      default:
        read-timeout: 10s
        connect-timeout: 5s
        max-content-length: 10485760
```

## Caching

### Cache Annotations

```java
@Singleton
public class ProductService {

    @Cacheable(cacheNames = "products")
    public Product getProduct(Long id) {
        return productRepository.findById(id).orElseThrow();
    }

    @CachePut(cacheNames = "products", parameters = "id")
    public Product updateProduct(Long id, ProductRequest request) {
        // Update logic
        return updatedProduct;
    }

    @CacheInvalidate(cacheNames = "products", parameters = "id")
    public void deleteProduct(Long id) {
        productRepository.deleteById(id);
    }
}
```

### Cache Configuration

```yaml
micronaut:
  caches:
    products:
      maximum-size: 1000
      expire-after-write: 10m
    sessions:
      maximum-size: 5000
      expire-after-access: 30m
```

## Scheduling

### Scheduled Tasks

```java
@Singleton
public class CleanupJob {

    @Scheduled(fixedDelay = "1h")
    public void cleanExpiredSessions() {
        log.info("Cleaning expired sessions");
        sessionRepository.deleteExpired();
    }

    @Scheduled(cron = "0 0 2 * * ?")  // Daily at 2 AM
    public void generateDailyReport() {
        log.info("Generating daily report");
        reportService.generateDaily();
    }

    @Scheduled(initialDelay = "30s", fixedRate = "5m")
    public void refreshCache() {
        log.info("Refreshing cache");
        cacheService.refresh();
    }
}
```

## Testing Patterns

### Controller Test with MockBean

```java
@MicronautTest
@DisplayName("UserController")
class UserControllerTest {

    @Inject @Client("/") HttpClient client;
    @Inject UserService userService;

    @MockBean(UserService.class)
    UserService mockUserService() {
        return mock(UserService.class);
    }

    @Test
    @DisplayName("POST /api/users should return 201 with created user")
    void createUser_shouldReturn201() {
        UserRequest request = new UserRequest("a@b.com", "pass1234", "Jo", "Do");
        UserResponse expected = new UserResponse(1L, "a@b.com", "Jo", "Do", true, now(), now());
        when(userService.createUser(any())).thenReturn(expected);

        HttpResponse<UserResponse> response = client.toBlocking()
            .exchange(HttpRequest.POST("/api/users", request), UserResponse.class);

        assertThat(response.status()).isEqualTo(HttpStatus.CREATED);
        assertThat(response.body().email()).isEqualTo("a@b.com");
    }

    @Test
    @DisplayName("GET /api/users/999 should return 404")
    void getUserById_notFound_shouldReturn404() {
        when(userService.getUserById(999L))
            .thenThrow(new ResourceNotFoundException("User", "id", 999L));

        assertThatThrownBy(() ->
            client.toBlocking().exchange(HttpRequest.GET("/api/users/999"))
        ).isInstanceOf(HttpClientResponseException.class)
         .satisfies(e -> assertThat(((HttpClientResponseException) e).getStatus())
             .isEqualTo(HttpStatus.NOT_FOUND));
    }
}
```

### Service Test with Nested Structure

```java
@ExtendWith(MockitoExtension.class)
@DisplayName("UserService")
class UserServiceTest {

    @Mock private UserRepository userRepository;
    @Mock private UserMapper userMapper;
    private UserServiceImpl userService;

    @BeforeEach
    void setUp() {
        userService = new UserServiceImpl(userRepository, userMapper);
    }

    @Nested
    @DisplayName("createUser")
    class CreateUser {

        @Test
        @DisplayName("should throw DuplicateResourceException when email exists")
        void shouldThrowWhenEmailExists() {
            when(userRepository.existsByEmail("dup@test.com")).thenReturn(true);

            assertThatThrownBy(() -> userService.createUser(
                new UserRequest("dup@test.com", "pass1234", "J", "D")))
                .isInstanceOf(DuplicateResourceException.class)
                .hasMessageContaining("email");

            verify(userRepository, never()).save(any());
        }
    }
}
```

### Testcontainers Integration Test

```java
@MicronautTest
@Testcontainers
@TestInstance(TestInstance.Lifecycle.PER_CLASS)
class UserIntegrationTest implements TestPropertyProvider {

    @Container
    static PostgreSQLContainer<?> postgres = new PostgreSQLContainer<>("postgres:15-alpine")
        .withDatabaseName("testdb")
        .withUsername("test")
        .withPassword("test");

    @Inject @Client("/") HttpClient client;

    @Override
    public Map<String, String> getProperties() {
        postgres.start();
        return Map.of(
            "datasources.default.url", postgres.getJdbcUrl(),
            "datasources.default.username", postgres.getUsername(),
            "datasources.default.password", postgres.getPassword()
        );
    }

    @Test
    void fullUserLifecycle() {
        // Create
        UserRequest createReq = new UserRequest("int@test.com", "pass1234", "Int", "Test");
        HttpResponse<UserResponse> createResp = client.toBlocking()
            .exchange(HttpRequest.POST("/api/users", createReq), UserResponse.class);
        assertThat(createResp.status()).isEqualTo(HttpStatus.CREATED);
        Long userId = createResp.body().id();

        // Read
        UserResponse getResp = client.toBlocking()
            .retrieve(HttpRequest.GET("/api/users/" + userId), UserResponse.class);
        assertThat(getResp.email()).isEqualTo("int@test.com");

        // Delete
        HttpResponse<?> deleteResp = client.toBlocking()
            .exchange(HttpRequest.DELETE("/api/users/" + userId));
        assertThat(deleteResp.status()).isEqualTo(HttpStatus.NO_CONTENT);
    }
}
```

## Deployment

### Dockerfile (JVM)

```dockerfile
FROM eclipse-temurin:21-jre-alpine
WORKDIR /app
COPY build/libs/myapp-*-all.jar app.jar
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget -qO- http://localhost:8080/health || exit 1
ENTRYPOINT ["java", "-jar", "app.jar"]
```

### Dockerfile (Native)

```dockerfile
FROM ghcr.io/graalvm/native-image-community:21 AS build
WORKDIR /app
COPY . .
RUN ./gradlew nativeCompile --no-daemon

FROM gcr.io/distroless/base-debian12
COPY --from=build /app/build/native/nativeCompile/myapp /app/myapp
EXPOSE 8080
ENTRYPOINT ["/app/myapp"]
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
        - name: myapp
          image: myapp:latest
          ports:
            - containerPort: 8080
          env:
            - name: MICRONAUT_ENVIRONMENTS
              value: "prod"
            - name: DB_USERNAME
              valueFrom:
                secretKeyRef:
                  name: db-credentials
                  key: username
            - name: DB_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: db-credentials
                  key: password
            - name: JWT_SECRET
              valueFrom:
                secretKeyRef:
                  name: jwt-secret
                  key: secret
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 5
          resources:
            requests:
              memory: "64Mi"
              cpu: "100m"
            limits:
              memory: "256Mi"
              cpu: "500m"
```

## Build Configuration

### Dependencies Template (build.gradle)

```groovy
plugins {
    id("io.micronaut.application") version "4.2.1"
    id("io.micronaut.aot") version "4.2.1"
}

version = "0.1"
group = "com.example"

repositories { mavenCentral() }

dependencies {
    // Core
    annotationProcessor("io.micronaut:micronaut-http-validation")
    annotationProcessor("io.micronaut.serde:micronaut-serde-processor")
    annotationProcessor("io.micronaut.validation:micronaut-validation-processor")
    implementation("io.micronaut:micronaut-http-client")
    implementation("io.micronaut.serde:micronaut-serde-jackson")
    implementation("io.micronaut.validation:micronaut-validation")

    // Database
    annotationProcessor("io.micronaut.data:micronaut-data-processor")
    implementation("io.micronaut.data:micronaut-data-jdbc")
    implementation("io.micronaut.sql:micronaut-jdbc-hikari")
    implementation("io.micronaut.flyway:micronaut-flyway")
    runtimeOnly("org.postgresql:postgresql")

    // Security
    annotationProcessor("io.micronaut.security:micronaut-security-annotations")
    implementation("io.micronaut.security:micronaut-security-jwt")

    // OpenAPI
    annotationProcessor("io.micronaut.openapi:micronaut-openapi")
    implementation("io.swagger.core.v3:swagger-annotations")

    // MapStruct + Lombok
    annotationProcessor("org.mapstruct:mapstruct-processor:1.5.5.Final")
    implementation("org.mapstruct:mapstruct:1.5.5.Final")
    compileOnly("org.projectlombok:lombok")
    annotationProcessor("org.projectlombok:lombok")
    annotationProcessor("org.projectlombok:lombok-mapstruct-binding:0.2.0")

    // Observability
    implementation("io.micronaut:micronaut-management")
    implementation("io.micronaut.micrometer:micronaut-micrometer-core")
    implementation("io.micronaut.micrometer:micronaut-micrometer-registry-prometheus")

    // Testing
    testImplementation("io.micronaut:micronaut-http-client")
    testImplementation("io.micronaut.test:micronaut-test-junit5")
    testImplementation("org.junit.jupiter:junit-jupiter-api")
    testImplementation("org.mockito:mockito-core")
    testImplementation("org.assertj:assertj-core")
    testRuntimeOnly("org.junit.jupiter:junit-jupiter-engine")
    testImplementation("org.testcontainers:junit-jupiter")
    testImplementation("org.testcontainers:postgresql")
}

application { mainClass.set("com.example.Application") }

java {
    sourceCompatibility = JavaVersion.toVersion("21")
    targetCompatibility = JavaVersion.toVersion("21")
}
```

### MapStruct Mapper Pattern

```java
@Mapper(componentModel = "jsr330")
public interface UserMapper {

    @Mapping(target = "id", ignore = true)
    @Mapping(target = "createdAt", ignore = true)
    @Mapping(target = "updatedAt", ignore = true)
    @Mapping(target = "active", constant = "true")
    User toEntity(UserRequest request);

    UserResponse toResponse(User user);

    List<UserResponse> toResponseList(List<User> users);

    @BeanMapping(nullValuePropertyMappingStrategy = NullValuePropertyMappingStrategy.IGNORE)
    @Mapping(target = "id", ignore = true)
    @Mapping(target = "createdAt", ignore = true)
    @Mapping(target = "updatedAt", ignore = true)
    @Mapping(target = "password", ignore = true)
    void updateEntity(UserRequest request, @MappingTarget User user);
}
```

### Database Migration (Flyway)

```sql
-- src/main/resources/db/migration/V1__create_users_table.sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'USER',
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_active ON users(active);
```
