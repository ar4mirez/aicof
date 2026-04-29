# Spring Boot (Java) Patterns Reference

## Contents

- [JPA Advanced Patterns](#jpa-advanced-patterns)
- [MapStruct Mapping](#mapstruct-mapping)
- [Security with JWT](#security-with-jwt)
- [WebFlux Reactive](#webflux-reactive)
- [Testing Strategies](#testing-strategies)
- [Actuator & Monitoring](#actuator--monitoring)
- [Configuration Patterns](#configuration-patterns)
- [Database Migration](#database-migration)
- [Caching](#caching)
- [Async Processing](#async-processing)
- [Deployment](#deployment)

## JPA Advanced Patterns

### Entity Relationships

```java
@Entity
@Table(name = "orders")
@Getter @Setter
@NoArgsConstructor @Builder @AllArgsConstructor
public class Order {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "user_id", nullable = false)
    private User user;

    @OneToMany(mappedBy = "order", cascade = CascadeType.ALL, orphanRemoval = true)
    @Builder.Default
    private List<OrderItem> items = new ArrayList<>();

    @Column(nullable = false)
    @Enumerated(EnumType.STRING)
    @Builder.Default
    private OrderStatus status = OrderStatus.PENDING;

    @CreationTimestamp
    @Column(updatable = false)
    private LocalDateTime createdAt;

    // Helper methods for bidirectional relationship management
    public void addItem(OrderItem item) {
        items.add(item);
        item.setOrder(this);
    }

    public void removeItem(OrderItem item) {
        items.remove(item);
        item.setOrder(null);
    }
}
```

### Entity Auditing

```java
@EntityListeners(AuditingEntityListener.class)
@MappedSuperclass
@Getter @Setter
public abstract class AuditableEntity {

    @CreatedDate
    @Column(updatable = false)
    private LocalDateTime createdAt;

    @LastModifiedDate
    private LocalDateTime updatedAt;

    @CreatedBy
    @Column(updatable = false)
    private String createdBy;

    @LastModifiedBy
    private String updatedBy;
}

// Enable in configuration
@Configuration
@EnableJpaAuditing
public class JpaConfig {

    @Bean
    public AuditorAware<String> auditorProvider() {
        return () -> Optional.ofNullable(SecurityContextHolder.getContext())
            .map(SecurityContext::getAuthentication)
            .filter(Authentication::isAuthenticated)
            .map(Authentication::getName);
    }
}
```

### Specifications for Dynamic Queries

```java
public class UserSpecifications {

    public static Specification<User> hasEmail(String email) {
        return (root, query, cb) ->
            email == null ? null : cb.equal(root.get("email"), email);
    }

    public static Specification<User> isActive() {
        return (root, query, cb) -> cb.isTrue(root.get("active"));
    }

    public static Specification<User> hasRole(User.Role role) {
        return (root, query, cb) ->
            role == null ? null : cb.equal(root.get("role"), role);
    }

    public static Specification<User> nameLike(String name) {
        return (root, query, cb) ->
            name == null ? null : cb.like(cb.lower(root.get("name")),
                "%" + name.toLowerCase() + "%");
    }
}

// Usage in service
public Page<UserResponse> searchUsers(UserSearchCriteria criteria, Pageable pageable) {
    Specification<User> spec = Specification
        .where(UserSpecifications.isActive())
        .and(UserSpecifications.hasRole(criteria.role()))
        .and(UserSpecifications.nameLike(criteria.name()));

    return userRepository.findAll(spec, pageable).map(userMapper::toResponse);
}

// Repository must extend JpaSpecificationExecutor
public interface UserRepository extends JpaRepository<User, Long>,
        JpaSpecificationExecutor<User> {}
```

### Projections for Optimized Queries

```java
// Interface-based projection (read-only, optimized SQL)
public interface UserSummary {
    Long getId();
    String getEmail();
    String getName();
}

// Record-based projection via JPQL constructor expression
@Query("""
    SELECT new com.example.dto.UserStats(
        u.role, COUNT(u), AVG(u.loginCount))
    FROM User u
    GROUP BY u.role
    """)
List<UserStats> getUserStatsByRole();

public record UserStats(User.Role role, Long count, Double avgLogins) {}
```

### Batch Operations

```java
// application.yml configuration for batch inserts
// spring.jpa.properties.hibernate.jdbc.batch_size: 25
// spring.jpa.properties.hibernate.order_inserts: true
// spring.jpa.properties.hibernate.order_updates: true

@Transactional
public void importUsers(List<UserRequest> requests) {
    List<User> users = requests.stream()
        .map(userMapper::toEntity)
        .toList();

    // saveAll uses batching when configured
    userRepository.saveAll(users);
}

// For very large imports, use batch chunks
@Transactional
public void bulkImport(List<UserRequest> requests) {
    int batchSize = 500;
    for (int i = 0; i < requests.size(); i += batchSize) {
        List<UserRequest> batch = requests.subList(
            i, Math.min(i + batchSize, requests.size()));
        List<User> users = batch.stream()
            .map(userMapper::toEntity)
            .toList();
        userRepository.saveAll(users);
        entityManager.flush();
        entityManager.clear();
    }
}
```

## MapStruct Mapping

### Advanced Mapper Configuration

```java
@Mapper(componentModel = "spring",
        unmappedTargetPolicy = ReportingPolicy.ERROR,
        nullValuePropertyMappingStrategy = NullValuePropertyMappingStrategy.IGNORE)
public interface OrderMapper {

    @Mapping(target = "id", ignore = true)
    @Mapping(target = "createdAt", ignore = true)
    @Mapping(target = "status", constant = "PENDING")
    @Mapping(target = "totalAmount", expression = "java(calculateTotal(request))")
    Order toEntity(OrderRequest request);

    @Mapping(target = "userName", source = "user.name")
    @Mapping(target = "itemCount", expression = "java(order.getItems().size())")
    OrderResponse toResponse(Order order);

    List<OrderResponse> toResponseList(List<Order> orders);

    @BeanMapping(nullValuePropertyMappingStrategy =
        NullValuePropertyMappingStrategy.IGNORE)
    @Mapping(target = "id", ignore = true)
    void updateEntity(OrderUpdateRequest request, @MappingTarget Order order);

    default BigDecimal calculateTotal(OrderRequest request) {
        return request.items().stream()
            .map(i -> i.price().multiply(BigDecimal.valueOf(i.quantity())))
            .reduce(BigDecimal.ZERO, BigDecimal::add);
    }
}
```

## Security with JWT

### JWT Filter

```java
@Component
@RequiredArgsConstructor
public class JwtAuthenticationFilter extends OncePerRequestFilter {

    private final JwtTokenProvider tokenProvider;
    private final UserDetailsService userDetailsService;

    @Override
    protected void doFilterInternal(HttpServletRequest request,
            HttpServletResponse response, FilterChain chain)
            throws ServletException, IOException {

        String token = extractToken(request);
        if (token != null && tokenProvider.validateToken(token)) {
            String username = tokenProvider.getUsername(token);
            UserDetails userDetails = userDetailsService.loadUserByUsername(username);

            var authentication = new UsernamePasswordAuthenticationToken(
                userDetails, null, userDetails.getAuthorities());
            authentication.setDetails(
                new WebAuthenticationDetailsSource().buildDetails(request));

            SecurityContextHolder.getContext().setAuthentication(authentication);
        }

        chain.doFilter(request, response);
    }

    private String extractToken(HttpServletRequest request) {
        String header = request.getHeader("Authorization");
        if (header != null && header.startsWith("Bearer ")) {
            return header.substring(7);
        }
        return null;
    }
}
```

### JWT Token Provider

```java
@Component
public class JwtTokenProvider {

    @Value("${jwt.secret}")
    private String secret;

    @Value("${jwt.expiration-ms:3600000}")
    private long expirationMs;

    private SecretKey getSigningKey() {
        return Keys.hmacShaKeyFor(secret.getBytes(StandardCharsets.UTF_8));
    }

    public String generateToken(UserDetails userDetails) {
        return Jwts.builder()
            .subject(userDetails.getUsername())
            .issuedAt(new Date())
            .expiration(new Date(System.currentTimeMillis() + expirationMs))
            .signWith(getSigningKey())
            .compact();
    }

    public String getUsername(String token) {
        return Jwts.parser()
            .verifyWith(getSigningKey())
            .build()
            .parseSignedClaims(token)
            .getPayload()
            .getSubject();
    }

    public boolean validateToken(String token) {
        try {
            Jwts.parser().verifyWith(getSigningKey()).build()
                .parseSignedClaims(token);
            return true;
        } catch (JwtException | IllegalArgumentException e) {
            return false;
        }
    }
}
```

### Security Config with JWT

```java
@Configuration
@EnableWebSecurity
@EnableMethodSecurity
@RequiredArgsConstructor
public class SecurityConfig {

    private final JwtAuthenticationFilter jwtFilter;

    @Bean
    public SecurityFilterChain securityFilterChain(HttpSecurity http) throws Exception {
        return http
            .csrf(AbstractHttpConfigurer::disable)
            .sessionManagement(session ->
                session.sessionCreationPolicy(SessionCreationPolicy.STATELESS))
            .authorizeHttpRequests(auth -> auth
                .requestMatchers("/api/v1/auth/**").permitAll()
                .requestMatchers("/api-docs/**", "/swagger-ui/**").permitAll()
                .requestMatchers("/actuator/health").permitAll()
                .anyRequest().authenticated())
            .addFilterBefore(jwtFilter, UsernamePasswordAuthenticationFilter.class)
            .build();
    }

    @Bean
    public AuthenticationManager authenticationManager(
            AuthenticationConfiguration config) throws Exception {
        return config.getAuthenticationManager();
    }

    @Bean
    public PasswordEncoder passwordEncoder() {
        return new BCryptPasswordEncoder(12);
    }
}
```

## WebFlux Reactive

### Reactive Controller

```java
@RestController
@RequestMapping("/api/v1/users")
@RequiredArgsConstructor
public class UserReactiveController {

    private final UserReactiveService userService;

    @GetMapping("/{id}")
    public Mono<ResponseEntity<UserResponse>> getUserById(@PathVariable String id) {
        return userService.findById(id)
            .map(ResponseEntity::ok)
            .defaultIfEmpty(ResponseEntity.notFound().build());
    }

    @GetMapping(produces = MediaType.TEXT_EVENT_STREAM_VALUE)
    public Flux<UserResponse> streamUsers() {
        return userService.findAll();
    }

    @PostMapping
    public Mono<ResponseEntity<UserResponse>> createUser(
            @Valid @RequestBody Mono<UserRequest> request) {
        return request
            .flatMap(userService::create)
            .map(user -> ResponseEntity.status(HttpStatus.CREATED).body(user));
    }
}
```

### Reactive Service with R2DBC

```java
@Service
@RequiredArgsConstructor
@Slf4j
public class UserReactiveService {

    private final UserReactiveRepository userRepository;

    public Mono<UserResponse> findById(String id) {
        return userRepository.findById(id)
            .map(this::toResponse)
            .switchIfEmpty(Mono.error(
                new ResourceNotFoundException("User", "id", id)));
    }

    public Flux<UserResponse> findAll() {
        return userRepository.findAll().map(this::toResponse);
    }

    public Mono<UserResponse> create(UserRequest request) {
        return userRepository.existsByEmail(request.email())
            .flatMap(exists -> {
                if (exists) {
                    return Mono.error(
                        new DuplicateResourceException("User", "email", request.email()));
                }
                return userRepository.save(toEntity(request));
            })
            .map(this::toResponse);
    }

    private UserResponse toResponse(User user) {
        return new UserResponse(user.getId(), user.getEmail(),
            user.getName(), user.getRole().name(), user.getCreatedAt());
    }

    private User toEntity(UserRequest request) {
        return User.builder()
            .email(request.email())
            .name(request.name())
            .role(User.Role.USER)
            .build();
    }
}
```

### WebClient for External APIs

```java
@Service
@Slf4j
public class ExternalApiClient {

    private final WebClient webClient;

    public ExternalApiClient(WebClient.Builder builder,
            @Value("${external.api.base-url}") String baseUrl) {
        this.webClient = builder
            .baseUrl(baseUrl)
            .defaultHeader(HttpHeaders.CONTENT_TYPE, MediaType.APPLICATION_JSON_VALUE)
            .filter(ExchangeFilterFunctions.basicAuthentication("user", "pass"))
            .build();
    }

    public Mono<ExternalData> fetchData(String id) {
        return webClient.get()
            .uri("/data/{id}", id)
            .retrieve()
            .onStatus(HttpStatusCode::is4xxClientError,
                response -> Mono.error(new ResourceNotFoundException("Data", "id", id)))
            .bodyToMono(ExternalData.class)
            .timeout(Duration.ofSeconds(5))
            .retryWhen(Retry.backoff(3, Duration.ofMillis(500)));
    }
}
```

## Testing Strategies

### Slice Tests (Controller Layer)

```java
@WebMvcTest(UserController.class)
@Import(SecurityConfig.class)
class UserControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @MockBean
    private UserService userService;

    @Autowired
    private ObjectMapper objectMapper;

    @Test
    @WithMockUser
    void shouldReturnUserWhenFound() throws Exception {
        var response = new UserResponse(1L, "test@example.com", "Test", "USER", null);
        when(userService.getUserById(1L)).thenReturn(response);

        mockMvc.perform(get("/api/v1/users/1"))
            .andExpect(status().isOk())
            .andExpect(jsonPath("$.email").value("test@example.com"))
            .andExpect(jsonPath("$.name").value("Test"));
    }

    @Test
    @WithMockUser
    void shouldReturnBadRequestForInvalidInput() throws Exception {
        var request = new UserRequest("", "", "");

        mockMvc.perform(post("/api/v1/users")
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(request)))
            .andExpect(status().isBadRequest())
            .andExpect(jsonPath("$.errors").isNotEmpty());
    }

    @Test
    @WithMockUser(roles = "USER")
    void shouldForbidDeleteForNonAdmin() throws Exception {
        mockMvc.perform(delete("/api/v1/users/1"))
            .andExpect(status().isForbidden());
    }

    @Test
    @WithMockUser(roles = "ADMIN")
    void shouldAllowDeleteForAdmin() throws Exception {
        mockMvc.perform(delete("/api/v1/users/1"))
            .andExpect(status().isNoContent());
        verify(userService).deleteUser(1L);
    }
}
```

### Repository Tests

```java
@DataJpaTest
@AutoConfigureTestDatabase(replace = AutoConfigureTestDatabase.Replace.NONE)
@Testcontainers
class UserRepositoryTest {

    @Container
    static PostgreSQLContainer<?> postgres =
        new PostgreSQLContainer<>("postgres:15-alpine");

    @DynamicPropertySource
    static void configureProperties(DynamicPropertyRegistry registry) {
        registry.add("spring.datasource.url", postgres::getJdbcUrl);
        registry.add("spring.datasource.username", postgres::getUsername);
        registry.add("spring.datasource.password", postgres::getPassword);
    }

    @Autowired
    private UserRepository userRepository;

    @Test
    void shouldFindByEmail() {
        User user = userRepository.save(User.builder()
            .email("test@example.com")
            .password("encoded")
            .name("Test")
            .build());

        Optional<User> found = userRepository.findByEmail("test@example.com");

        assertThat(found).isPresent();
        assertThat(found.get().getId()).isEqualTo(user.getId());
    }

    @Test
    void shouldReturnEmptyForNonExistentEmail() {
        Optional<User> found = userRepository.findByEmail("nobody@example.com");
        assertThat(found).isEmpty();
    }
}
```

### Test Fixtures and Builders

```java
public class TestFixtures {

    public static User aUser() {
        return User.builder()
            .email("test@example.com")
            .password("encodedPassword")
            .name("Test User")
            .role(User.Role.USER)
            .active(true)
            .build();
    }

    public static UserRequest aUserRequest() {
        return UserRequest.builder()
            .email("test@example.com")
            .password("Password123!")
            .name("Test User")
            .build();
    }

    public static UserResponse aUserResponse() {
        return UserResponse.builder()
            .id(1L)
            .email("test@example.com")
            .name("Test User")
            .role("USER")
            .createdAt(LocalDateTime.now())
            .build();
    }
}
```

### Abstract Integration Test Base

```java
@SpringBootTest(webEnvironment = SpringBootTest.WebEnvironment.RANDOM_PORT)
@Testcontainers
public abstract class AbstractIntegrationTest {

    @Container
    static PostgreSQLContainer<?> postgres =
        new PostgreSQLContainer<>("postgres:15-alpine")
            .withDatabaseName("testdb")
            .withUsername("test")
            .withPassword("test");

    @DynamicPropertySource
    static void configureProperties(DynamicPropertyRegistry registry) {
        registry.add("spring.datasource.url", postgres::getJdbcUrl);
        registry.add("spring.datasource.username", postgres::getUsername);
        registry.add("spring.datasource.password", postgres::getPassword);
    }

    @Autowired
    protected TestRestTemplate restTemplate;

    @Autowired
    protected ObjectMapper objectMapper;
}

// Subclass for specific tests
class OrderIntegrationTest extends AbstractIntegrationTest {

    @Test
    void shouldCreateOrder() {
        var response = restTemplate.postForEntity(
            "/api/v1/orders", orderRequest, OrderResponse.class);
        assertThat(response.getStatusCode()).isEqualTo(HttpStatus.CREATED);
    }
}
```

## Actuator & Monitoring

### Custom Health Indicator

```java
@Component
public class DatabaseHealthIndicator implements HealthIndicator {

    private final DataSource dataSource;

    public DatabaseHealthIndicator(DataSource dataSource) {
        this.dataSource = dataSource;
    }

    @Override
    public Health health() {
        try (var conn = dataSource.getConnection()) {
            if (conn.isValid(2)) {
                return Health.up()
                    .withDetail("database", "PostgreSQL")
                    .withDetail("status", "Connection valid")
                    .build();
            }
        } catch (SQLException e) {
            return Health.down()
                .withDetail("error", e.getMessage())
                .build();
        }
        return Health.down().build();
    }
}
```

### Custom Metrics with Micrometer

```java
@Service
@RequiredArgsConstructor
public class OrderServiceImpl implements OrderService {

    private final OrderRepository orderRepository;
    private final MeterRegistry meterRegistry;

    @Override
    @Transactional
    public OrderResponse createOrder(OrderRequest request) {
        Timer.Sample sample = Timer.start(meterRegistry);

        try {
            Order order = processOrder(request);
            meterRegistry.counter("orders.created",
                "status", order.getStatus().name()).increment();
            return toResponse(order);
        } finally {
            sample.stop(meterRegistry.timer("orders.create.duration"));
        }
    }
}
```

### Prometheus Configuration

```yaml
# application.yml
management:
  endpoints:
    web:
      exposure:
        include: health,info,metrics,prometheus
  endpoint:
    health:
      show-details: when-authorized
      probes:
        enabled: true
  metrics:
    distribution:
      percentiles-histogram:
        http.server.requests: true
    tags:
      application: ${spring.application.name}
```

## Configuration Patterns

### Type-Safe Configuration Properties

```java
@ConfigurationProperties(prefix = "app")
@Validated
public record AppProperties(
    @NotBlank String name,
    @NotNull SecurityProperties security,
    @NotNull CacheProperties cache
) {
    public record SecurityProperties(
        @NotBlank String jwtSecret,
        @Positive long tokenExpirationMs,
        @NotEmpty List<String> allowedOrigins
    ) {}

    public record CacheProperties(
        @Positive int ttlSeconds,
        @Positive int maxSize
    ) {}
}

// Enable in main class
@SpringBootApplication
@EnableConfigurationProperties(AppProperties.class)
public class MyProjectApplication { }

// Usage
@Service
@RequiredArgsConstructor
public class TokenService {
    private final AppProperties appProperties;

    public String generateToken() {
        var security = appProperties.security();
        // use security.jwtSecret(), security.tokenExpirationMs()
    }
}
```

### Profile-Specific Configuration

```yaml
# application.yml (shared)
spring:
  profiles:
    active: ${SPRING_PROFILES_ACTIVE:dev}

app:
  name: myproject

---
# application-dev.yml
spring:
  jpa:
    show-sql: true
  flyway:
    clean-disabled: false

logging:
  level:
    org.hibernate.SQL: DEBUG

---
# application-prod.yml
spring:
  jpa:
    show-sql: false

server:
  error:
    include-message: never
    include-stacktrace: never

logging:
  level:
    root: WARN
    com.example.myproject: INFO
```

## Database Migration

### Flyway Versioned Migrations

```sql
-- V1__create_users_table.sql
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
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_active ON users(active);

-- V2__create_orders_table.sql
CREATE TABLE orders (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    total_amount DECIMAL(12, 2) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status);
```

### Repeatable Migrations

```sql
-- R__create_views.sql (runs when checksum changes)
CREATE OR REPLACE VIEW active_users AS
SELECT id, email, name, role, created_at
FROM users
WHERE active = true;
```

## Caching

### Spring Cache with Caffeine

```java
@Configuration
@EnableCaching
public class CacheConfig {

    @Bean
    public CacheManager cacheManager() {
        CaffeineCacheManager cacheManager = new CaffeineCacheManager();
        cacheManager.setCaffeine(Caffeine.newBuilder()
            .maximumSize(1000)
            .expireAfterWrite(Duration.ofMinutes(10))
            .recordStats());
        return cacheManager;
    }
}

@Service
@RequiredArgsConstructor
@Transactional(readOnly = true)
public class UserServiceImpl implements UserService {

    private final UserRepository userRepository;

    @Override
    @Cacheable(value = "users", key = "#id")
    public UserResponse getUserById(Long id) {
        return userRepository.findById(id)
            .map(userMapper::toResponse)
            .orElseThrow(() -> new ResourceNotFoundException("User", "id", id));
    }

    @Override
    @Transactional
    @CacheEvict(value = "users", key = "#id")
    public UserResponse updateUser(Long id, UserRequest request) {
        // update logic
    }

    @Override
    @Transactional
    @CacheEvict(value = "users", allEntries = true)
    public void deleteUser(Long id) {
        userRepository.deleteById(id);
    }
}
```

## Async Processing

### Async Configuration

```java
@Configuration
@EnableAsync
public class AsyncConfig {

    @Bean(name = "taskExecutor")
    public Executor taskExecutor() {
        ThreadPoolTaskExecutor executor = new ThreadPoolTaskExecutor();
        executor.setCorePoolSize(5);
        executor.setMaxPoolSize(10);
        executor.setQueueCapacity(100);
        executor.setThreadNamePrefix("async-");
        executor.setRejectedExecutionHandler(new ThreadPoolExecutor.CallerRunsPolicy());
        executor.initialize();
        return executor;
    }
}

@Service
@RequiredArgsConstructor
@Slf4j
public class NotificationService {

    private final EmailSender emailSender;

    @Async("taskExecutor")
    public CompletableFuture<Void> sendWelcomeEmail(String email, String name) {
        log.info("Sending welcome email to {}", email);
        emailSender.send(email, "Welcome", "Hello " + name);
        return CompletableFuture.completedFuture(null);
    }
}
```

### Event-Driven with Application Events

```java
// Event record
public record UserCreatedEvent(Long userId, String email, Instant timestamp) {}

// Publishing
@Service
@RequiredArgsConstructor
public class UserServiceImpl implements UserService {

    private final ApplicationEventPublisher eventPublisher;

    @Override
    @Transactional
    public UserResponse createUser(UserRequest request) {
        User saved = userRepository.save(userMapper.toEntity(request));

        eventPublisher.publishEvent(new UserCreatedEvent(
            saved.getId(), saved.getEmail(), Instant.now()));

        return userMapper.toResponse(saved);
    }
}

// Listening
@Component
@RequiredArgsConstructor
@Slf4j
public class UserEventListener {

    private final NotificationService notificationService;

    @TransactionalEventListener(phase = TransactionPhase.AFTER_COMMIT)
    @Async
    public void onUserCreated(UserCreatedEvent event) {
        log.info("User created event: {}", event.userId());
        notificationService.sendWelcomeEmail(event.email(), "New User");
    }
}
```

## Deployment

### Dockerfile (Multi-Stage Build)

```dockerfile
# Build stage
FROM eclipse-temurin:21-jdk-alpine AS build
WORKDIR /app
COPY pom.xml .
COPY .mvn .mvn
COPY mvnw .
RUN chmod +x mvnw && ./mvnw dependency:go-offline -B
COPY src src
RUN ./mvnw clean package -DskipTests -B

# Runtime stage
FROM eclipse-temurin:21-jre-alpine
WORKDIR /app

RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

COPY --from=build /app/target/*.jar app.jar

USER appuser
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=30s \
    CMD wget -qO- http://localhost:8080/actuator/health || exit 1

ENTRYPOINT ["java", "-jar", "app.jar"]
```

### Kubernetes Readiness and Liveness

```yaml
# application.yml
management:
  endpoint:
    health:
      probes:
        enabled: true
  health:
    livenessState:
      enabled: true
    readinessState:
      enabled: true
```

```yaml
# k8s deployment snippet
containers:
  - name: myproject
    image: myproject:latest
    ports:
      - containerPort: 8080
    livenessProbe:
      httpGet:
        path: /actuator/health/liveness
        port: 8080
      initialDelaySeconds: 30
      periodSeconds: 10
    readinessProbe:
      httpGet:
        path: /actuator/health/readiness
        port: 8080
      initialDelaySeconds: 10
      periodSeconds: 5
    resources:
      requests:
        memory: "256Mi"
        cpu: "250m"
      limits:
        memory: "512Mi"
        cpu: "500m"
    env:
      - name: SPRING_PROFILES_ACTIVE
        value: "prod"
      - name: DB_PASSWORD
        valueFrom:
          secretKeyRef:
            name: db-credentials
            key: password
```

### GraalVM Native Image

```xml
<!-- pom.xml profile for native compilation -->
<profiles>
    <profile>
        <id>native</id>
        <build>
            <plugins>
                <plugin>
                    <groupId>org.graalvm.buildtools</groupId>
                    <artifactId>native-maven-plugin</artifactId>
                    <configuration>
                        <buildArgs>
                            <buildArg>--no-fallback</buildArg>
                            <buildArg>-H:+ReportExceptionStackTraces</buildArg>
                        </buildArgs>
                    </configuration>
                </plugin>
            </plugins>
        </build>
    </profile>
</profiles>
```

```bash
# Build native executable
./mvnw -Pnative native:compile

# Build native Docker image
./mvnw -Pnative spring-boot:build-image

# Run native executable
./target/myproject
```
