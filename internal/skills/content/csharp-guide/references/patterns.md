# C# Patterns Reference

## Contents

- [Async Patterns](#async-patterns)
- [Dependency Injection Registration](#dependency-injection-registration)
- [LINQ Patterns](#linq-patterns)
- [Result Pattern](#result-pattern)
- [Options Pattern](#options-pattern)

## Async Patterns

### Parallel Async Execution

```csharp
public async Task<DashboardData> LoadDashboardAsync(
    int userId, CancellationToken ct = default)
{
    // Fire all independent queries concurrently
    var ordersTask = _orderRepo.GetRecentAsync(userId, ct);
    var profileTask = _userRepo.GetProfileAsync(userId, ct);
    var statsTask = _analyticsService.GetStatsAsync(userId, ct);

    await Task.WhenAll(ordersTask, profileTask, statsTask);

    return new DashboardData(
        Orders: await ordersTask,
        Profile: await profileTask,
        Stats: await statsTask);
}
```

### Timeout with CancellationToken

```csharp
public async Task<PaymentResult> ChargeAsync(
    PaymentRequest request, CancellationToken ct = default)
{
    using var timeoutCts = new CancellationTokenSource(TimeSpan.FromSeconds(10));
    using var linkedCts = CancellationTokenSource
        .CreateLinkedTokenSource(ct, timeoutCts.Token);

    try
    {
        return await _gateway.ProcessAsync(request, linkedCts.Token);
    }
    catch (OperationCanceledException) when (timeoutCts.IsCancellationRequested)
    {
        throw new TimeoutException("Payment gateway timed out after 10s.");
    }
}
```

### Retry with Polly

```csharp
// Registration
services.AddHttpClient<ICatalogService, CatalogService>()
    .AddTransientHttpErrorPolicy(p =>
        p.WaitAndRetryAsync(3, attempt =>
            TimeSpan.FromMilliseconds(200 * Math.Pow(2, attempt))));

// Manual retry pattern (without Polly)
public static async Task<T> RetryAsync<T>(
    Func<CancellationToken, Task<T>> operation,
    int maxRetries = 3,
    CancellationToken ct = default)
{
    for (int attempt = 0; ; attempt++)
    {
        try
        {
            return await operation(ct);
        }
        catch (HttpRequestException) when (attempt < maxRetries)
        {
            var delay = TimeSpan.FromMilliseconds(100 * Math.Pow(2, attempt));
            await Task.Delay(delay, ct);
        }
    }
}
```

### Channel-based Producer/Consumer

```csharp
public sealed class BackgroundProcessor : BackgroundService
{
    private readonly Channel<WorkItem> _channel;

    public BackgroundProcessor()
    {
        _channel = Channel.CreateBounded<WorkItem>(
            new BoundedChannelOptions(100)
            {
                FullMode = BoundedChannelFullMode.Wait,
                SingleReader = true
            });
    }

    public async ValueTask EnqueueAsync(
        WorkItem item, CancellationToken ct = default)
    {
        await _channel.Writer.WriteAsync(item, ct);
    }

    protected override async Task ExecuteAsync(CancellationToken ct)
    {
        await foreach (var item in _channel.Reader.ReadAllAsync(ct))
        {
            await ProcessItemAsync(item, ct);
        }
    }

    private static Task ProcessItemAsync(WorkItem item, CancellationToken ct)
    {
        // Process the work item
        return Task.CompletedTask;
    }
}
```

## Dependency Injection Registration

### Extension Method Pattern

```csharp
// Group registrations by layer into extension methods
public static class ServiceCollectionExtensions
{
    public static IServiceCollection AddDomainServices(
        this IServiceCollection services)
    {
        services.AddScoped<IPricingEngine, PricingEngine>();
        services.AddScoped<IInventoryChecker, InventoryChecker>();
        return services;
    }

    public static IServiceCollection AddInfrastructure(
        this IServiceCollection services, IConfiguration config)
    {
        services.AddDbContext<AppDbContext>(opts =>
            opts.UseNpgsql(config.GetConnectionString("Default")));

        services.AddScoped<IOrderRepository, OrderRepository>();
        services.AddScoped<IUserRepository, UserRepository>();

        services.AddSingleton<ICacheService, RedisCacheService>();

        return services;
    }

    public static IServiceCollection AddExternalClients(
        this IServiceCollection services, IConfiguration config)
    {
        services.AddHttpClient<IPaymentGateway, StripeGateway>(client =>
        {
            client.BaseAddress = new Uri(config["Stripe:BaseUrl"]!);
            client.Timeout = TimeSpan.FromSeconds(10);
        });

        services.AddHttpClient<IEmailSender, SendGridSender>(client =>
        {
            client.BaseAddress = new Uri("https://api.sendgrid.com/");
        });

        return services;
    }
}

// Usage in Program.cs
var builder = WebApplication.CreateBuilder(args);
builder.Services
    .AddDomainServices()
    .AddInfrastructure(builder.Configuration)
    .AddExternalClients(builder.Configuration);
```

### Keyed Services (.NET 8+)

```csharp
// Register multiple implementations of the same interface
services.AddKeyedScoped<INotificationSender, EmailSender>("email");
services.AddKeyedScoped<INotificationSender, SmsSender>("sms");
services.AddKeyedScoped<INotificationSender, PushSender>("push");

// Inject by key
public sealed class NotificationService(
    [FromKeyedServices("email")] INotificationSender emailSender,
    [FromKeyedServices("sms")] INotificationSender smsSender)
{
    // Use specific implementations
}
```

## LINQ Patterns

### Efficient Querying with EF Core

```csharp
// Projection: load only what you need
public async Task<List<OrderListItem>> GetOrderListAsync(
    int customerId, CancellationToken ct = default)
{
    return await _dbContext.Orders
        .AsNoTracking()
        .Where(o => o.CustomerId == customerId)
        .OrderByDescending(o => o.CreatedAt)
        .Select(o => new OrderListItem(
            o.Id,
            o.CreatedAt,
            o.Total,
            o.Items.Count))
        .ToListAsync(ct);
}

// Pagination
public async Task<PagedResult<Product>> GetProductsAsync(
    int page, int pageSize, CancellationToken ct = default)
{
    var query = _dbContext.Products.AsNoTracking();

    var totalCount = await query.CountAsync(ct);

    var items = await query
        .OrderBy(p => p.Name)
        .Skip((page - 1) * pageSize)
        .Take(pageSize)
        .ToListAsync(ct);

    return new PagedResult<Product>(items, totalCount, page, pageSize);
}
```

### In-Memory LINQ Patterns

```csharp
// GroupBy with aggregation
var salesByRegion = orders
    .GroupBy(o => o.Region)
    .Select(g => new
    {
        Region = g.Key,
        TotalSales = g.Sum(o => o.Total),
        OrderCount = g.Count(),
        AverageOrder = g.Average(o => o.Total)
    })
    .OrderByDescending(x => x.TotalSales);

// Chunk for batch processing (.NET 8+)
foreach (var batch in largeCollection.Chunk(100))
{
    await ProcessBatchAsync(batch, ct);
}

// Zip and aggregate
var paired = names.Zip(scores, (name, score) => new { name, score });
```

## Result Pattern

```csharp
// Lightweight discriminated union for operation results
public sealed record Result<T>
{
    public T? Value { get; }
    public string? Error { get; }
    public bool IsSuccess => Error is null;

    private Result(T value) { Value = value; }
    private Result(string error) { Error = error; }

    public static Result<T> Success(T value) => new(value);
    public static Result<T> Failure(string error) => new(error);

    public TResult Match<TResult>(
        Func<T, TResult> onSuccess,
        Func<string, TResult> onFailure) =>
        IsSuccess ? onSuccess(Value!) : onFailure(Error!);
}

// Usage
public async Task<Result<Order>> PlaceOrderAsync(OrderRequest request)
{
    if (!await _inventory.IsAvailableAsync(request.ProductId))
        return Result<Order>.Failure("Product is out of stock.");

    var order = Order.Create(request);
    await _repository.AddAsync(order);

    return Result<Order>.Success(order);
}
```

## Options Pattern

```csharp
// Strongly-typed configuration
public sealed class SmtpSettings
{
    public const string SectionName = "Smtp";

    public required string Host { get; init; }
    public required int Port { get; init; }
    public required string Username { get; init; }
    public required string Password { get; init; }
    public bool UseSsl { get; init; } = true;
}

// Registration with validation
services.AddOptions<SmtpSettings>()
    .BindConfiguration(SmtpSettings.SectionName)
    .ValidateDataAnnotations()
    .ValidateOnStart();

// Injection
public sealed class EmailSender(IOptions<SmtpSettings> options)
{
    private readonly SmtpSettings _settings = options.Value;
}
```
