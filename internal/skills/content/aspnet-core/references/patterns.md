# ASP.NET Core Patterns Reference

## Contents

- [EF Core Advanced Patterns](#ef-core-advanced-patterns)
- [Domain Exceptions](#domain-exceptions)
- [Repository Pattern](#repository-pattern)
- [Identity and Security](#identity-and-security)
- [SignalR Real-Time](#signalr-real-time)
- [Blazor Integration](#blazor-integration)
- [Testing Strategies](#testing-strategies)
- [Deployment](#deployment)
- [Performance](#performance)

## EF Core Advanced Patterns

### Audit Trail with SaveChanges Override

```csharp
public interface IAuditable
{
    DateTime CreatedAt { get; set; }
    DateTime? UpdatedAt { get; set; }
    string? CreatedBy { get; set; }
    string? UpdatedBy { get; set; }
}

public class AuditableDbContext : DbContext
{
    private readonly IHttpContextAccessor _httpContextAccessor;

    public AuditableDbContext(
        DbContextOptions options,
        IHttpContextAccessor httpContextAccessor) : base(options)
    {
        _httpContextAccessor = httpContextAccessor;
    }

    public override Task<int> SaveChangesAsync(
        CancellationToken cancellationToken = default)
    {
        var userId = _httpContextAccessor.HttpContext?.User
            .FindFirst(ClaimTypes.NameIdentifier)?.Value;

        foreach (var entry in ChangeTracker.Entries<IAuditable>())
        {
            switch (entry.State)
            {
                case EntityState.Added:
                    entry.Entity.CreatedAt = DateTime.UtcNow;
                    entry.Entity.CreatedBy = userId;
                    break;
                case EntityState.Modified:
                    entry.Entity.UpdatedAt = DateTime.UtcNow;
                    entry.Entity.UpdatedBy = userId;
                    break;
            }
        }

        return base.SaveChangesAsync(cancellationToken);
    }
}
```

### Soft Delete with Global Query Filters

```csharp
public interface ISoftDeletable
{
    bool IsDeleted { get; set; }
    DateTime? DeletedAt { get; set; }
}

// In DbContext.OnModelCreating
protected override void OnModelCreating(ModelBuilder modelBuilder)
{
    // Apply soft delete filter to all ISoftDeletable entities
    foreach (var entityType in modelBuilder.Model.GetEntityTypes())
    {
        if (typeof(ISoftDeletable).IsAssignableFrom(entityType.ClrType))
        {
            modelBuilder.Entity(entityType.ClrType)
                .HasQueryFilter(
                    BuildSoftDeleteFilter(entityType.ClrType));
        }
    }
}

private static LambdaExpression BuildSoftDeleteFilter(Type entityType)
{
    var parameter = Expression.Parameter(entityType, "e");
    var property = Expression.Property(parameter, nameof(ISoftDeletable.IsDeleted));
    var condition = Expression.Equal(property, Expression.Constant(false));
    return Expression.Lambda(condition, parameter);
}

// Override Delete to soft-delete instead
public override Task<int> SaveChangesAsync(
    CancellationToken cancellationToken = default)
{
    foreach (var entry in ChangeTracker.Entries<ISoftDeletable>()
        .Where(e => e.State == EntityState.Deleted))
    {
        entry.State = EntityState.Modified;
        entry.Entity.IsDeleted = true;
        entry.Entity.DeletedAt = DateTime.UtcNow;
    }

    return base.SaveChangesAsync(cancellationToken);
}
```

### Specification Pattern for Complex Queries

```csharp
public interface ISpecification<T>
{
    Expression<Func<T, bool>> Criteria { get; }
    List<Expression<Func<T, object>>> Includes { get; }
    Expression<Func<T, object>>? OrderBy { get; }
    Expression<Func<T, object>>? OrderByDescending { get; }
    int? Take { get; }
    int? Skip { get; }
}

public abstract class BaseSpecification<T> : ISpecification<T>
{
    public Expression<Func<T, bool>> Criteria { get; }
    public List<Expression<Func<T, object>>> Includes { get; } = new();
    public Expression<Func<T, object>>? OrderBy { get; private set; }
    public Expression<Func<T, object>>? OrderByDescending { get; private set; }
    public int? Take { get; private set; }
    public int? Skip { get; private set; }

    protected BaseSpecification(Expression<Func<T, bool>> criteria)
    {
        Criteria = criteria;
    }

    protected void AddInclude(Expression<Func<T, object>> include)
        => Includes.Add(include);

    protected void AddOrderBy(Expression<Func<T, object>> orderBy)
        => OrderBy = orderBy;

    protected void AddOrderByDescending(Expression<Func<T, object>> orderBy)
        => OrderByDescending = orderBy;

    protected void ApplyPaging(int skip, int take)
    {
        Skip = skip;
        Take = take;
    }
}

// Usage
public class ActiveUsersSpec : BaseSpecification<User>
{
    public ActiveUsersSpec(int page, int pageSize)
        : base(u => u.Active)
    {
        AddOrderByDescending(u => u.CreatedAt);
        ApplyPaging((page - 1) * pageSize, pageSize);
    }
}
```

### Compiled Queries for Hot Paths

```csharp
public class UserRepository
{
    private static readonly Func<AppDbContext, string, Task<User?>>
        GetByEmailQuery = EF.CompileAsyncQuery(
            (AppDbContext ctx, string email) =>
                ctx.Users.FirstOrDefault(u => u.Email == email));

    public Task<User?> GetByEmailAsync(string email)
        => GetByEmailQuery(_context, email);
}
```

### Migration with Rollback

```csharp
public partial class AddUserRoles : Migration
{
    protected override void Up(MigrationBuilder migrationBuilder)
    {
        migrationBuilder.AddColumn<string>(
            name: "role",
            table: "users",
            type: "varchar(50)",
            maxLength: 50,
            nullable: false,
            defaultValue: "User");

        migrationBuilder.CreateIndex(
            name: "IX_users_role",
            table: "users",
            column: "role");
    }

    protected override void Down(MigrationBuilder migrationBuilder)
    {
        migrationBuilder.DropIndex(
            name: "IX_users_role",
            table: "users");

        migrationBuilder.DropColumn(
            name: "role",
            table: "users");
    }
}
```

## Domain Exceptions

### Full Exception Hierarchy

```csharp
public abstract class DomainException : Exception
{
    public abstract int StatusCode { get; }
    protected DomainException(string message) : base(message) { }
}

public class NotFoundException : DomainException
{
    public override int StatusCode => 404;
    public NotFoundException(string entity, object id)
        : base($"{entity} with ID {id} was not found") { }
}

public class ConflictException : DomainException
{
    public override int StatusCode => 409;
    public ConflictException(string entity, string field, object value)
        : base($"{entity} with {field} '{value}' already exists") { }
}

public class ValidationException : DomainException
{
    public override int StatusCode => 400;
    public IDictionary<string, string[]> Errors { get; }
    public ValidationException(IDictionary<string, string[]> errors)
        : base("Validation failed") { Errors = errors; }
}

public class ForbiddenException : DomainException
{
    public override int StatusCode => 403;
    public ForbiddenException(string message = "Access denied")
        : base(message) { }
}
```

## Repository Pattern

### Generic Repository with Pagination

```csharp
public interface IRepository<T> where T : class
{
    Task<T?> GetByIdAsync(long id, CancellationToken ct = default);
    Task<(IReadOnlyList<T> Items, long TotalCount)> GetPagedAsync(
        int page, int pageSize, CancellationToken ct = default);
    Task<T> CreateAsync(T entity, CancellationToken ct = default);
    Task<T> UpdateAsync(T entity, CancellationToken ct = default);
    Task DeleteAsync(T entity, CancellationToken ct = default);
}

public class Repository<T> : IRepository<T> where T : class
{
    protected readonly AppDbContext Context;
    protected readonly DbSet<T> DbSet;

    public Repository(AppDbContext context)
    {
        Context = context;
        DbSet = context.Set<T>();
    }

    public async Task<T?> GetByIdAsync(long id, CancellationToken ct = default)
        => await DbSet.FindAsync([id], ct);

    public async Task<(IReadOnlyList<T> Items, long TotalCount)> GetPagedAsync(
        int page, int pageSize, CancellationToken ct = default)
    {
        var totalCount = await DbSet.LongCountAsync(ct);
        var items = await DbSet.AsNoTracking()
            .Skip((page - 1) * pageSize)
            .Take(pageSize)
            .ToListAsync(ct);
        return (items, totalCount);
    }

    public async Task<T> CreateAsync(T entity, CancellationToken ct = default)
    {
        DbSet.Add(entity);
        await Context.SaveChangesAsync(ct);
        return entity;
    }

    public async Task<T> UpdateAsync(T entity, CancellationToken ct = default)
    {
        DbSet.Update(entity);
        await Context.SaveChangesAsync(ct);
        return entity;
    }

    public async Task DeleteAsync(T entity, CancellationToken ct = default)
    {
        DbSet.Remove(entity);
        await Context.SaveChangesAsync(ct);
    }
}
```

### Paged Response DTO

```csharp
public record PagedResponse<T>(
    IReadOnlyList<T> Items,
    int Page,
    int PageSize,
    long TotalCount,
    int TotalPages
);

// Extension method for building paged responses
public static class PagingExtensions
{
    public static PagedResponse<T> ToPagedResponse<T>(
        this (IReadOnlyList<T> Items, long TotalCount) data,
        int page, int pageSize)
    {
        var totalPages = (int)Math.Ceiling(data.TotalCount / (double)pageSize);
        return new PagedResponse<T>(
            data.Items, page, pageSize, data.TotalCount, totalPages);
    }
}
```

## Identity and Security

### ASP.NET Core Identity Setup

```csharp
// Program.cs
builder.Services.AddIdentity<ApplicationUser, IdentityRole<long>>(options =>
{
    options.Password.RequireDigit = true;
    options.Password.RequireLowercase = true;
    options.Password.RequireUppercase = true;
    options.Password.RequireNonAlphanumeric = true;
    options.Password.RequiredLength = 8;

    options.Lockout.DefaultLockoutTimeSpan = TimeSpan.FromMinutes(15);
    options.Lockout.MaxFailedAccessAttempts = 5;

    options.User.RequireUniqueEmail = true;
})
.AddEntityFrameworkStores<AppDbContext>()
.AddDefaultTokenProviders();
```

### JWT Token Generation

```csharp
public class TokenService : ITokenService
{
    private readonly JwtSettings _settings;

    public TokenService(IOptions<JwtSettings> settings)
    {
        _settings = settings.Value;
    }

    public string GenerateToken(ApplicationUser user, IList<string> roles)
    {
        var claims = new List<Claim>
        {
            new(ClaimTypes.NameIdentifier, user.Id.ToString()),
            new(ClaimTypes.Email, user.Email!),
            new(ClaimTypes.Name, user.UserName!),
        };

        claims.AddRange(roles.Select(
            role => new Claim(ClaimTypes.Role, role)));

        var key = new SymmetricSecurityKey(
            Encoding.UTF8.GetBytes(_settings.Secret));
        var creds = new SigningCredentials(key, SecurityAlgorithms.HmacSha256);

        var token = new JwtSecurityToken(
            issuer: _settings.Issuer,
            audience: _settings.Audience,
            claims: claims,
            expires: DateTime.UtcNow.AddMinutes(_settings.ExpirationMinutes),
            signingCredentials: creds);

        return new JwtSecurityTokenHandler().WriteToken(token);
    }
}
```

### Policy-Based Authorization

```csharp
// Program.cs
builder.Services.AddAuthorization(options =>
{
    options.AddPolicy("AdminOnly", policy =>
        policy.RequireRole("Admin"));

    options.AddPolicy("CanManageUsers", policy =>
        policy.RequireAssertion(context =>
            context.User.IsInRole("Admin") ||
            context.User.HasClaim("permission", "manage_users")));

    options.AddPolicy("MinimumAge", policy =>
        policy.RequireAssertion(context =>
        {
            var dobClaim = context.User.FindFirst("date_of_birth");
            if (dobClaim == null) return false;
            var dob = DateTime.Parse(dobClaim.Value);
            return DateTime.UtcNow.Year - dob.Year >= 18;
        }));
});

// Usage
[Authorize(Policy = "CanManageUsers")]
public async Task<IResult> DeleteUser(long id) { /* ... */ }
```

### CORS Configuration

```csharp
builder.Services.AddCors(options =>
{
    options.AddPolicy("AllowFrontend", policy =>
    {
        policy.WithOrigins(
                builder.Configuration.GetSection("Cors:Origins").Get<string[]>()!)
            .AllowAnyHeader()
            .AllowAnyMethod()
            .AllowCredentials();
    });
});

// In middleware pipeline (before auth)
app.UseCors("AllowFrontend");
```

## SignalR Real-Time

### Hub Definition

```csharp
[Authorize]
public class NotificationHub : Hub
{
    private readonly ILogger<NotificationHub> _logger;

    public NotificationHub(ILogger<NotificationHub> logger)
    {
        _logger = logger;
    }

    public override async Task OnConnectedAsync()
    {
        var userId = Context.User?.FindFirst(ClaimTypes.NameIdentifier)?.Value;
        if (userId != null)
        {
            await Groups.AddToGroupAsync(Context.ConnectionId, $"user_{userId}");
        }
        await base.OnConnectedAsync();
    }

    public async Task SendToUser(string userId, string message)
    {
        await Clients.Group($"user_{userId}")
            .SendAsync("ReceiveNotification", new { Message = message, Timestamp = DateTime.UtcNow });
    }

    public async Task BroadcastMessage(string message)
    {
        await Clients.All.SendAsync("ReceiveMessage", message);
    }
}

// Program.cs registration
builder.Services.AddSignalR();
app.MapHub<NotificationHub>("/hubs/notifications");
```

### Sending from Services (Outside Hubs)

```csharp
public class OrderService
{
    private readonly IHubContext<NotificationHub> _hubContext;

    public OrderService(IHubContext<NotificationHub> hubContext)
    {
        _hubContext = hubContext;
    }

    public async Task CompleteOrderAsync(long orderId)
    {
        // ... business logic ...

        await _hubContext.Clients.Group($"user_{order.UserId}")
            .SendAsync("OrderCompleted", new { OrderId = orderId });
    }
}
```

## Blazor Integration

### Blazor Server with Existing API

```csharp
// Program.cs additions for Blazor Server
builder.Services.AddRazorComponents()
    .AddInteractiveServerComponents();

builder.Services.AddHttpClient("Api", client =>
{
    client.BaseAddress = new Uri("https://localhost:5001");
});

// After existing middleware
app.MapRazorComponents<App>()
    .AddInteractiveServerRenderMode();
```

### Shared Components Pattern

```csharp
// Components/UserList.razor
@inject HttpClient Http

@if (_users == null)
{
    <p>Loading...</p>
}
else
{
    <table>
        @foreach (var user in _users)
        {
            <tr>
                <td>@user.Email</td>
                <td>@user.FirstName @user.LastName</td>
                <td>@user.Role</td>
            </tr>
        }
    </table>
}

@code {
    private List<UserResponse>? _users;

    protected override async Task OnInitializedAsync()
    {
        _users = await Http.GetFromJsonAsync<List<UserResponse>>("/api/users");
    }
}
```

## Testing Strategies

### Custom WebApplicationFactory

```csharp
public class CustomWebApplicationFactory : WebApplicationFactory<Program>
{
    private readonly PostgreSqlContainer _postgres = new PostgreSqlBuilder()
        .WithImage("postgres:15-alpine")
        .Build();

    protected override void ConfigureWebHost(IWebHostBuilder builder)
    {
        builder.ConfigureServices(services =>
        {
            // Remove existing DbContext registration
            var descriptor = services.SingleOrDefault(
                d => d.ServiceType == typeof(DbContextOptions<AppDbContext>));
            if (descriptor != null) services.Remove(descriptor);

            services.AddDbContext<AppDbContext>(options =>
                options.UseNpgsql(_postgres.GetConnectionString()));

            // Seed test data
            using var scope = services.BuildServiceProvider().CreateScope();
            var db = scope.ServiceProvider.GetRequiredService<AppDbContext>();
            db.Database.Migrate();
            SeedTestData(db);
        });
    }

    public async Task InitializeContainerAsync()
        => await _postgres.StartAsync();

    public async Task DisposeContainerAsync()
        => await _postgres.DisposeAsync();

    private static void SeedTestData(AppDbContext db)
    {
        db.Users.Add(new User
        {
            Email = "admin@test.com",
            PasswordHash = BCrypt.Net.BCrypt.HashPassword("Admin123!"),
            FirstName = "Test",
            LastName = "Admin",
            Role = UserRole.Admin
        });
        db.SaveChanges();
    }
}
```

### Testing with Authentication

```csharp
public class AuthenticatedTests : IAsyncLifetime
{
    private readonly CustomWebApplicationFactory _factory = new();
    private HttpClient _client = null!;

    public async Task InitializeAsync()
    {
        await _factory.InitializeContainerAsync();
        _client = _factory.CreateClient();

        // Login and set auth header
        var loginResponse = await _client.PostAsJsonAsync(
            "/api/auth/login",
            new { Email = "admin@test.com", Password = "Admin123!" });

        var token = await loginResponse.Content
            .ReadFromJsonAsync<TokenResponse>();
        _client.DefaultRequestHeaders.Authorization =
            new AuthenticationHeaderValue("Bearer", token!.AccessToken);
    }

    public async Task DisposeAsync()
        => await _factory.DisposeContainerAsync();

    [Fact]
    public async Task GetUsers_Authenticated_ReturnsOk()
    {
        var response = await _client.GetAsync("/api/users");
        response.StatusCode.Should().Be(HttpStatusCode.OK);
    }

    [Fact]
    public async Task DeleteUser_AsAdmin_ReturnsNoContent()
    {
        var response = await _client.DeleteAsync("/api/users/1");
        response.StatusCode.Should().Be(HttpStatusCode.NoContent);
    }
}
```

### Validator Unit Tests

```csharp
public class CreateUserRequestValidatorTests
{
    private readonly CreateUserRequestValidator _validator = new();

    [Theory]
    [InlineData("", "Password123!", "John", "Doe", false)]
    [InlineData("invalid-email", "Password123!", "John", "Doe", false)]
    [InlineData("test@example.com", "short", "John", "Doe", false)]
    [InlineData("test@example.com", "Password123!", "", "Doe", false)]
    [InlineData("test@example.com", "Password123!", "John", "Doe", true)]
    public async Task Validate_ReturnsExpectedResult(
        string email, string password, string first, string last, bool expected)
    {
        var request = new CreateUserRequest(email, password, first, last);
        var result = await _validator.ValidateAsync(request);
        result.IsValid.Should().Be(expected);
    }
}
```

## Deployment

### Dockerfile (Multi-Stage)

```dockerfile
# Build stage
FROM mcr.microsoft.com/dotnet/sdk:8.0 AS build
WORKDIR /src

COPY ["MyApp.Api/MyApp.Api.csproj", "MyApp.Api/"]
COPY ["MyApp.Core/MyApp.Core.csproj", "MyApp.Core/"]
COPY ["MyApp.Infrastructure/MyApp.Infrastructure.csproj", "MyApp.Infrastructure/"]
COPY ["MyApp.Contracts/MyApp.Contracts.csproj", "MyApp.Contracts/"]
COPY ["Directory.Build.props", "."]
COPY ["Directory.Packages.props", "."]

RUN dotnet restore "MyApp.Api/MyApp.Api.csproj"
COPY . .
RUN dotnet publish "MyApp.Api/MyApp.Api.csproj" -c Release -o /app/publish

# Runtime stage
FROM mcr.microsoft.com/dotnet/aspnet:8.0 AS runtime
WORKDIR /app
EXPOSE 8080

COPY --from=build /app/publish .
USER app
ENTRYPOINT ["dotnet", "MyApp.Api.dll"]
```

### Health Check Endpoint

```csharp
builder.Services.AddHealthChecks()
    .AddDbContextCheck<AppDbContext>("database")
    .AddCheck("redis", () =>
    {
        // Custom health check
        try
        {
            // Check Redis connection
            return HealthCheckResult.Healthy();
        }
        catch (Exception ex)
        {
            return HealthCheckResult.Unhealthy(exception: ex);
        }
    });

app.MapHealthChecks("/health", new HealthCheckOptions
{
    ResponseWriter = async (context, report) =>
    {
        context.Response.ContentType = "application/json";
        var result = new
        {
            Status = report.Status.ToString(),
            Duration = report.TotalDuration,
            Checks = report.Entries.Select(e => new
            {
                Name = e.Key,
                Status = e.Value.Status.ToString(),
                Duration = e.Value.Duration
            })
        };
        await context.Response.WriteAsJsonAsync(result);
    }
});
```

### docker-compose.yml

```yaml
version: "3.8"

services:
  api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - ASPNETCORE_ENVIRONMENT=Production
      - ConnectionStrings__DefaultConnection=Host=db;Database=myapp;Username=postgres;Password=postgres
    depends_on:
      db:
        condition: service_healthy

  db:
    image: postgres:15-alpine
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

## Performance

### Response Caching

```csharp
builder.Services.AddResponseCaching();
builder.Services.AddOutputCache(options =>
{
    options.AddPolicy("ShortCache", builder =>
        builder.Expire(TimeSpan.FromMinutes(1)));
    options.AddPolicy("LongCache", builder =>
        builder.Expire(TimeSpan.FromMinutes(30)).Tag("static"));
});

// On endpoint
group.MapGet("/", GetAll)
    .CacheOutput("ShortCache");
```

### Rate Limiting

```csharp
builder.Services.AddRateLimiter(options =>
{
    options.AddFixedWindowLimiter("fixed", opt =>
    {
        opt.Window = TimeSpan.FromMinutes(1);
        opt.PermitLimit = 60;
        opt.QueueLimit = 10;
    });

    options.AddSlidingWindowLimiter("sliding", opt =>
    {
        opt.Window = TimeSpan.FromMinutes(1);
        opt.SegmentsPerWindow = 6;
        opt.PermitLimit = 60;
    });

    options.OnRejected = async (context, token) =>
    {
        context.HttpContext.Response.StatusCode = 429;
        await context.HttpContext.Response
            .WriteAsync("Rate limit exceeded", token);
    };
});

app.UseRateLimiter();

// On endpoint
group.MapPost("/", Create).RequireRateLimiting("fixed");
```

### Background Services

```csharp
public class CleanupBackgroundService : BackgroundService
{
    private readonly IServiceScopeFactory _scopeFactory;
    private readonly ILogger<CleanupBackgroundService> _logger;

    public CleanupBackgroundService(
        IServiceScopeFactory scopeFactory,
        ILogger<CleanupBackgroundService> logger)
    {
        _scopeFactory = scopeFactory;
        _logger = logger;
    }

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        while (!stoppingToken.IsCancellationRequested)
        {
            try
            {
                using var scope = _scopeFactory.CreateScope();
                var db = scope.ServiceProvider
                    .GetRequiredService<AppDbContext>();

                var cutoff = DateTime.UtcNow.AddDays(-30);
                var deleted = await db.Users
                    .Where(u => u.IsDeleted && u.DeletedAt < cutoff)
                    .ExecuteDeleteAsync(stoppingToken);

                _logger.LogInformation(
                    "Purged {Count} soft-deleted users", deleted);
            }
            catch (Exception ex)
            {
                _logger.LogError(ex, "Cleanup failed");
            }

            await Task.Delay(TimeSpan.FromHours(24), stoppingToken);
        }
    }
}

// Register: builder.Services.AddHostedService<CleanupBackgroundService>();
```

### Performance Guardrails

- Use `AsNoTracking()` for all read-only queries
- Use `AsSplitQuery()` for queries with multiple includes (avoid cartesian explosion)
- Use compiled queries (`EF.CompileAsyncQuery`) for frequently executed queries
- Implement response caching and output caching for read-heavy endpoints
- Use `IMemoryCache` or `IDistributedCache` for application-level caching
- Batch database operations with `ExecuteUpdateAsync` / `ExecuteDeleteAsync` (EF Core 7+)
- Use `AddHttpClient` with `IHttpClientFactory` (avoid socket exhaustion)
- Enable response compression: `builder.Services.AddResponseCompression()`
- Profile with `dotnet-counters`, `dotnet-trace`, and Application Insights
