# Java Patterns Reference

## Contents

- [Spring Dependency Injection](#spring-dependency-injection)
- [Spring REST Controller](#spring-rest-controller)
- [Spring Service Layer](#spring-service-layer)
- [Spring Repository with JPA](#spring-repository-with-jpa)
- [Spring Exception Handling](#spring-exception-handling)
- [Stream API Recipes](#stream-api-recipes)
- [Optional Chaining](#optional-chaining)
- [Builder with Records](#builder-with-records)

## Spring Dependency Injection

```java
// Prefer constructor injection (no @Autowired needed with single constructor)
@Service
public class OrderService {

    private final OrderRepository orderRepository;
    private final PaymentGateway paymentGateway;
    private final NotificationService notificationService;

    public OrderService(OrderRepository orderRepository,
                        PaymentGateway paymentGateway,
                        NotificationService notificationService) {
        this.orderRepository = orderRepository;
        this.paymentGateway = paymentGateway;
        this.notificationService = notificationService;
    }
}
```

## Spring REST Controller

```java
@RestController
@RequestMapping("/api/v1/users")
public class UserController {

    private final UserService userService;

    public UserController(UserService userService) {
        this.userService = userService;
    }

    @GetMapping("/{id}")
    public ResponseEntity<UserResponse> getUser(@PathVariable String id) {
        return userService.findById(id)
                .map(UserResponse::from)
                .map(ResponseEntity::ok)
                .orElse(ResponseEntity.notFound().build());
    }

    @PostMapping
    public ResponseEntity<UserResponse> createUser(
            @Valid @RequestBody CreateUserRequest request) {
        User created = userService.create(request.toUser());
        URI location = URI.create("/api/v1/users/" + created.id());
        return ResponseEntity.created(location)
                .body(UserResponse.from(created));
    }

    @GetMapping
    public ResponseEntity<List<UserResponse>> listUsers(
            @RequestParam(defaultValue = "0") int page,
            @RequestParam(defaultValue = "20") int size) {
        Page<User> users = userService.findAll(PageRequest.of(page, size));
        List<UserResponse> body = users.stream()
                .map(UserResponse::from)
                .toList();
        return ResponseEntity.ok(body);
    }
}
```

## Spring Service Layer

```java
@Service
@Transactional(readOnly = true)
public class UserService {

    private final UserRepository userRepository;

    public UserService(UserRepository userRepository) {
        this.userRepository = userRepository;
    }

    public Optional<User> findById(String id) {
        return userRepository.findById(id);
    }

    @Transactional
    public User create(User user) {
        if (userRepository.existsByEmail(user.email())) {
            throw new DuplicateEmailException(user.email());
        }
        return userRepository.save(user);
    }

    @Transactional
    public User update(String id, UpdateUserRequest request) {
        User existing = userRepository.findById(id)
                .orElseThrow(() -> new UserNotFoundException(id));
        User updated = existing.withDisplayName(request.displayName());
        return userRepository.save(updated);
    }
}
```

## Spring Repository with JPA

```java
public interface UserRepository extends JpaRepository<User, String> {

    boolean existsByEmail(String email);

    Optional<User> findByEmail(String email);

    @Query("SELECT u FROM User u WHERE u.active = true AND u.createdAt > :since")
    List<User> findActiveUsersSince(@Param("since") LocalDateTime since);

    @Query("""
            SELECT u FROM User u
            JOIN u.roles r
            WHERE r.name = :roleName
            ORDER BY u.createdAt DESC
            """)
    Page<User> findByRoleName(@Param("roleName") String roleName, Pageable pageable);
}
```

## Spring Exception Handling

```java
@RestControllerAdvice
public class GlobalExceptionHandler {

    @ExceptionHandler(UserNotFoundException.class)
    public ResponseEntity<ErrorResponse> handleNotFound(UserNotFoundException ex) {
        var error = new ErrorResponse("NOT_FOUND", ex.getMessage());
        return ResponseEntity.status(HttpStatus.NOT_FOUND).body(error);
    }

    @ExceptionHandler(MethodArgumentNotValidException.class)
    public ResponseEntity<ErrorResponse> handleValidation(
            MethodArgumentNotValidException ex) {
        List<String> errors = ex.getBindingResult().getFieldErrors().stream()
                .map(fe -> fe.getField() + ": " + fe.getDefaultMessage())
                .toList();
        var error = new ErrorResponse("VALIDATION_ERROR", String.join("; ", errors));
        return ResponseEntity.badRequest().body(error);
    }

    @ExceptionHandler(Exception.class)
    public ResponseEntity<ErrorResponse> handleUnexpected(Exception ex) {
        var error = new ErrorResponse("INTERNAL_ERROR", "An unexpected error occurred");
        return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR).body(error);
    }

    public record ErrorResponse(String code, String message) {}
}
```

## Stream API Recipes

```java
// Grouping and counting
Map<String, Long> countByCity = users.stream()
        .collect(Collectors.groupingBy(User::city, Collectors.counting()));

// Grouping with downstream transformation
Map<Department, Double> avgSalary = employees.stream()
        .collect(Collectors.groupingBy(
                Employee::department,
                Collectors.averagingDouble(Employee::salary)));

// Collecting to a map (handle duplicates)
Map<String, User> byEmail = users.stream()
        .collect(Collectors.toMap(
                User::email,
                Function.identity(),
                (existing, replacement) -> existing));

// Reducing with identity
BigDecimal total = lineItems.stream()
        .map(LineItem::total)
        .reduce(BigDecimal.ZERO, BigDecimal::add);

// Teeing collector (Java 12+): compute two results in one pass
record Stats(long count, double average) {}
Stats stats = scores.stream()
        .collect(Collectors.teeing(
                Collectors.counting(),
                Collectors.averagingDouble(Integer::doubleValue),
                Stats::new));
```

## Optional Chaining

```java
// Deep navigation without null checks
Optional<String> zipCode = order.customer()
        .flatMap(Customer::shippingAddress)
        .map(Address::zipCode);

// Fallback chain: cache -> database -> remote
User user = cache.findUser(id)
        .or(() -> database.findUser(id))
        .or(() -> remoteService.fetchUser(id))
        .orElseThrow(() -> new UserNotFoundException(id));

// Filtering within Optional
Optional<User> activeAdmin = userRepository.findById(id)
        .filter(User::isActive)
        .filter(u -> u.hasRole("ADMIN"));

// Transforming to a different type
Optional<UserSummary> summary = userRepository.findById(id)
        .map(user -> new UserSummary(
                user.id(),
                user.displayName(),
                user.email()));
```

## Builder with Records

```java
// For records that need many optional fields, use a companion builder
public record SearchCriteria(
        String query,
        List<String> categories,
        LocalDate fromDate,
        LocalDate toDate,
        int page,
        int pageSize) {

    public static Builder builder(String query) {
        return new Builder(query);
    }

    public static final class Builder {
        private final String query;
        private List<String> categories = List.of();
        private LocalDate fromDate;
        private LocalDate toDate;
        private int page = 0;
        private int pageSize = 20;

        private Builder(String query) {
            this.query = Objects.requireNonNull(query);
        }

        public Builder categories(List<String> categories) {
            this.categories = List.copyOf(categories);
            return this;
        }

        public Builder dateRange(LocalDate from, LocalDate to) {
            this.fromDate = from;
            this.toDate = to;
            return this;
        }

        public Builder page(int page, int pageSize) {
            this.page = page;
            this.pageSize = pageSize;
            return this;
        }

        public SearchCriteria build() {
            return new SearchCriteria(query, categories,
                    fromDate, toDate, page, pageSize);
        }
    }
}

// Usage
SearchCriteria criteria = SearchCriteria.builder("java streams")
        .categories(List.of("tutorials", "guides"))
        .dateRange(LocalDate.of(2024, 1, 1), LocalDate.now())
        .page(0, 10)
        .build();
```
