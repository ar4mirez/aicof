# Blazor Patterns Reference

## Contents

- [Form Patterns](#form-patterns)
- [Component Lifecycle](#component-lifecycle)
- [Service Layer](#service-layer)
- [State Management](#state-management)
- [SignalR Real-Time](#signalr-real-time)
- [Authentication & Authorization](#authentication--authorization)
- [Error Handling](#error-handling)
- [Testing Patterns](#testing-patterns)
- [Performance Patterns](#performance-patterns)
- [JavaScript Interop](#javascript-interop)

## Form Patterns

### Create/Edit Form with Validation

```razor
@page "/users/new"
@page "/users/{Id:long}/edit"
@attribute [Authorize]
@inject IUserService UserService
@inject NavigationManager Navigation
@inject IToastService ToastService

<PageTitle>@(IsEdit ? "Edit User" : "New User")</PageTitle>

@if (loading)
{
    <LoadingSpinner />
}
else
{
    <EditForm Model="model" OnValidSubmit="HandleSubmit">
        <DataAnnotationsValidator />
        <ValidationSummary class="alert alert-danger" />

        <div class="mb-3">
            <label for="email" class="form-label">Email</label>
            <InputText id="email" class="form-control"
                       @bind-Value="model.Email" disabled="@IsEdit" />
            <ValidationMessage For="@(() => model.Email)" class="text-danger" />
        </div>

        @if (!IsEdit)
        {
            <div class="mb-3">
                <label for="password" class="form-label">Password</label>
                <InputText id="password" type="password" class="form-control"
                           @bind-Value="model.Password" />
                <ValidationMessage For="@(() => model.Password)" class="text-danger" />
            </div>
        }

        <div class="mb-3">
            <label for="firstName" class="form-label">First Name</label>
            <InputText id="firstName" class="form-control"
                       @bind-Value="model.FirstName" />
            <ValidationMessage For="@(() => model.FirstName)" class="text-danger" />
        </div>

        <div class="d-flex gap-2">
            <button type="submit" class="btn btn-primary" disabled="@submitting">
                @if (submitting)
                {
                    <span class="spinner-border spinner-border-sm"></span>
                }
                @(IsEdit ? "Update" : "Create")
            </button>
            <button type="button" class="btn btn-secondary"
                    @onclick="() => Navigation.NavigateTo("/users")">
                Cancel
            </button>
        </div>
    </EditForm>
}

@code {
    [Parameter]
    public long? Id { get; set; }

    private bool IsEdit => Id.HasValue;
    private UserFormModel model = new();
    private bool loading;
    private bool submitting;

    protected override async Task OnInitializedAsync()
    {
        if (IsEdit)
        {
            loading = true;
            try
            {
                var user = await UserService.GetUserAsync(Id!.Value);
                model = new UserFormModel
                {
                    Email = user.Email,
                    FirstName = user.FirstName,
                    LastName = user.LastName,
                    Active = user.Active
                };
            }
            catch (Exception ex)
            {
                ToastService.ShowError($"Failed to load user: {ex.Message}");
                Navigation.NavigateTo("/users");
            }
            finally
            {
                loading = false;
            }
        }
    }

    private async Task HandleSubmit()
    {
        submitting = true;
        try
        {
            if (IsEdit)
            {
                await UserService.UpdateUserAsync(Id!.Value,
                    new UpdateUserRequest(model.FirstName, model.LastName, model.Active));
                ToastService.ShowSuccess("User updated successfully");
            }
            else
            {
                await UserService.CreateUserAsync(
                    new CreateUserRequest(model.Email, model.Password,
                        model.FirstName, model.LastName));
                ToastService.ShowSuccess("User created successfully");
            }
            Navigation.NavigateTo("/users");
        }
        catch (Exception ex)
        {
            ToastService.ShowError(ex.Message);
        }
        finally
        {
            submitting = false;
        }
    }
}
```

### Custom Validation Model

```csharp
private class UserFormModel
{
    [Required]
    [EmailAddress]
    public string Email { get; set; } = string.Empty;

    [Required(ErrorMessage = "Password is required for new users")]
    [MinLength(8)]
    public string Password { get; set; } = string.Empty;

    [Required]
    [MaxLength(100)]
    public string FirstName { get; set; } = string.Empty;

    [Required]
    [MaxLength(100)]
    public string LastName { get; set; } = string.Empty;

    public bool Active { get; set; } = true;
}
```

### FluentValidation Integration

```csharp
// Install: Blazored.FluentValidation
// In form: replace <DataAnnotationsValidator /> with:
// <FluentValidationValidator />

public class UserFormValidator : AbstractValidator<UserFormModel>
{
    public UserFormValidator()
    {
        RuleFor(x => x.Email).NotEmpty().EmailAddress();
        RuleFor(x => x.Password).MinimumLength(8)
            .When(x => string.IsNullOrEmpty(x.Email) == false);
        RuleFor(x => x.FirstName).NotEmpty().MaximumLength(100);
        RuleFor(x => x.LastName).NotEmpty().MaximumLength(100);
    }
}
```

## Component Lifecycle

### Complete Lifecycle Order

```
1. SetParametersAsync    — Parameters received from parent
2. OnInitialized         — First initialization (sync)
3. OnInitializedAsync    — First initialization (async)
4. OnParametersSet       — After every parameter update (sync)
5. OnParametersSetAsync  — After every parameter update (async)
6. ShouldRender          — Return false to skip rendering
7. BuildRenderTree       — Render markup
8. OnAfterRender         — DOM available (sync, firstRender flag)
9. OnAfterRenderAsync    — DOM available (async, firstRender flag)
10. Dispose/DisposeAsync — Cleanup when removed from tree
```

### Disposal Pattern

```razor
@implements IAsyncDisposable

@code {
    private CancellationTokenSource _cts = new();
    private Timer? _timer;

    protected override void OnInitialized()
    {
        _timer = new Timer(Tick, null, TimeSpan.Zero, TimeSpan.FromSeconds(30));
    }

    protected override async Task OnInitializedAsync()
    {
        await LoadDataAsync(_cts.Token);
    }

    private async void Tick(object? state)
    {
        await InvokeAsync(async () =>
        {
            await RefreshDataAsync(_cts.Token);
            StateHasChanged();
        });
    }

    public async ValueTask DisposeAsync()
    {
        _cts.Cancel();
        _cts.Dispose();
        if (_timer is not null)
        {
            await _timer.DisposeAsync();
        }
    }
}
```

### ShouldRender Override

```razor
@code {
    private int _previousCount;

    protected override bool ShouldRender()
    {
        // Only re-render when the count actually changes
        if (Items?.Count == _previousCount)
            return false;

        _previousCount = Items?.Count ?? 0;
        return true;
    }
}
```

## Service Layer

### Typed HttpClient Service

```csharp
public interface IUserService
{
    Task<PagedResponse<UserResponse>> GetUsersAsync(int page = 1, int pageSize = 10);
    Task<UserResponse> GetUserAsync(long id);
    Task<UserResponse> CreateUserAsync(CreateUserRequest request);
    Task<UserResponse> UpdateUserAsync(long id, UpdateUserRequest request);
    Task<bool> DeleteUserAsync(long id);
}

public class UserService : IUserService
{
    private readonly HttpClient _httpClient;

    public UserService(HttpClient httpClient)
    {
        _httpClient = httpClient;
    }

    public async Task<PagedResponse<UserResponse>> GetUsersAsync(
        int page = 1, int pageSize = 10)
    {
        return await _httpClient.GetFromJsonAsync<PagedResponse<UserResponse>>(
            $"api/users?page={page}&pageSize={pageSize}")
            ?? new PagedResponse<UserResponse>([], 0, 0, 0, 0);
    }

    public async Task<UserResponse> GetUserAsync(long id)
    {
        var response = await _httpClient.GetAsync($"api/users/{id}");
        response.EnsureSuccessStatusCode();
        return (await response.Content.ReadFromJsonAsync<UserResponse>())!;
    }

    public async Task<UserResponse> CreateUserAsync(CreateUserRequest request)
    {
        var response = await _httpClient.PostAsJsonAsync("api/users", request);
        if (!response.IsSuccessStatusCode)
        {
            var error = await response.Content.ReadAsStringAsync();
            throw new HttpRequestException(error);
        }
        return (await response.Content.ReadFromJsonAsync<UserResponse>())!;
    }

    public async Task<UserResponse> UpdateUserAsync(long id, UpdateUserRequest request)
    {
        var response = await _httpClient.PutAsJsonAsync($"api/users/{id}", request);
        if (!response.IsSuccessStatusCode)
        {
            var error = await response.Content.ReadAsStringAsync();
            throw new HttpRequestException(error);
        }
        return (await response.Content.ReadFromJsonAsync<UserResponse>())!;
    }

    public async Task<bool> DeleteUserAsync(long id)
    {
        var response = await _httpClient.DeleteAsync($"api/users/{id}");
        return response.IsSuccessStatusCode;
    }
}
```

### DI Registration (Program.cs)

```csharp
var builder = WebAssemblyHostBuilder.CreateDefault(args);
builder.RootComponents.Add<App>("#app");
builder.RootComponents.Add<HeadOutlet>("head::after");

// Typed HTTP client
builder.Services.AddHttpClient<IUserService, UserService>(client =>
    client.BaseAddress = new Uri(builder.HostEnvironment.BaseAddress));

// State management
builder.Services.AddScoped<AppState>();

// Third-party services
builder.Services.AddBlazoredLocalStorage();
builder.Services.AddBlazoredToast();

// Authentication
builder.Services.AddAuthorizationCore();

await builder.Build().RunAsync();
```

## State Management

### Fluxor (Redux-Style)

```csharp
// State record
[FeatureState]
public record UsersState
{
    public bool Loading { get; init; }
    public List<UserResponse> Users { get; init; } = [];
    public string? Error { get; init; }
}

// Actions
public record LoadUsersAction;
public record LoadUsersSuccessAction(List<UserResponse> Users);
public record LoadUsersFailureAction(string Error);

// Reducers
public static class UsersReducers
{
    [ReducerMethod]
    public static UsersState OnLoadUsers(UsersState state, LoadUsersAction action) =>
        state with { Loading = true, Error = null };

    [ReducerMethod]
    public static UsersState OnLoadUsersSuccess(
        UsersState state, LoadUsersSuccessAction action) =>
        state with { Loading = false, Users = action.Users };

    [ReducerMethod]
    public static UsersState OnLoadUsersFailure(
        UsersState state, LoadUsersFailureAction action) =>
        state with { Loading = false, Error = action.Error };
}

// Effects (side effects for async operations)
public class UsersEffects
{
    private readonly IUserService _userService;

    public UsersEffects(IUserService userService) => _userService = userService;

    [EffectMethod]
    public async Task HandleLoadUsers(LoadUsersAction action, IDispatcher dispatcher)
    {
        try
        {
            var result = await _userService.GetUsersAsync();
            dispatcher.Dispatch(new LoadUsersSuccessAction(result.Items.ToList()));
        }
        catch (Exception ex)
        {
            dispatcher.Dispatch(new LoadUsersFailureAction(ex.Message));
        }
    }
}
```

### Using Fluxor in Components

```razor
@inherits FluxorComponent
@inject IState<UsersState> UsersState
@inject IDispatcher Dispatcher

@if (UsersState.Value.Loading)
{
    <LoadingSpinner />
}
else
{
    @foreach (var user in UsersState.Value.Users)
    {
        <UserCard @key="user.Id" User="user" />
    }
}

@code {
    protected override void OnInitialized()
    {
        base.OnInitialized();
        Dispatcher.Dispatch(new LoadUsersAction());
    }
}
```

## SignalR Real-Time

### SignalR Hub (Server)

```csharp
public class NotificationHub : Hub
{
    public async Task SendMessage(string user, string message)
    {
        await Clients.All.SendAsync("ReceiveMessage", user, message);
    }

    public async Task JoinGroup(string groupName)
    {
        await Groups.AddToGroupAsync(Context.ConnectionId, groupName);
    }

    public async Task SendToGroup(string groupName, string message)
    {
        await Clients.Group(groupName).SendAsync("ReceiveMessage", message);
    }

    public override async Task OnConnectedAsync()
    {
        await Clients.Caller.SendAsync("Connected", Context.ConnectionId);
        await base.OnConnectedAsync();
    }
}
```

### SignalR Client (Blazor Component)

```razor
@implements IAsyncDisposable
@inject NavigationManager Navigation

<div class="chat">
    @foreach (var msg in messages)
    {
        <div class="message">
            <strong>@msg.User:</strong> @msg.Text
        </div>
    }
</div>

<input @bind="messageInput" @onkeyup="HandleKeyUp" />
<button @onclick="SendMessage">Send</button>

@code {
    private HubConnection? hubConnection;
    private List<ChatMessage> messages = [];
    private string messageInput = string.Empty;

    protected override async Task OnInitializedAsync()
    {
        hubConnection = new HubConnectionBuilder()
            .WithUrl(Navigation.ToAbsoluteUri("/hubs/notification"))
            .WithAutomaticReconnect()
            .Build();

        hubConnection.On<string, string>("ReceiveMessage", (user, message) =>
        {
            messages.Add(new ChatMessage(user, message));
            InvokeAsync(StateHasChanged);
        });

        hubConnection.Reconnecting += error =>
        {
            // Show reconnecting UI
            return Task.CompletedTask;
        };

        await hubConnection.StartAsync();
    }

    private async Task SendMessage()
    {
        if (hubConnection is not null && !string.IsNullOrWhiteSpace(messageInput))
        {
            await hubConnection.SendAsync("SendMessage", "User", messageInput);
            messageInput = string.Empty;
        }
    }

    private async Task HandleKeyUp(KeyboardEventArgs e)
    {
        if (e.Key == "Enter") await SendMessage();
    }

    public async ValueTask DisposeAsync()
    {
        if (hubConnection is not null)
        {
            await hubConnection.DisposeAsync();
        }
    }

    private record ChatMessage(string User, string Text);
}
```

## Authentication & Authorization

### App.razor with Auth

```razor
<CascadingAuthenticationState>
    <Router AppAssembly="@typeof(App).Assembly">
        <Found Context="routeData">
            <AuthorizeRouteView RouteData="@routeData"
                                DefaultLayout="@typeof(MainLayout)">
                <NotAuthorized>
                    <RedirectToLogin />
                </NotAuthorized>
            </AuthorizeRouteView>
            <FocusOnNavigate RouteData="@routeData" Selector="h1" />
        </Found>
        <NotFound>
            <LayoutView Layout="@typeof(MainLayout)">
                <div class="alert alert-warning">Page not found.</div>
            </LayoutView>
        </NotFound>
    </Router>
</CascadingAuthenticationState>
```

### AuthorizeView in Components

```razor
<AuthorizeView>
    <Authorized>
        <span>Hello, @context.User.Identity?.Name!</span>
        <a href="authentication/logout">Log out</a>
    </Authorized>
    <NotAuthorized>
        <a href="authentication/login">Log in</a>
    </NotAuthorized>
</AuthorizeView>

<!-- Role-based visibility -->
<AuthorizeView Roles="Admin">
    <NavLink class="nav-link" href="admin">Admin Panel</NavLink>
</AuthorizeView>

<!-- Policy-based visibility -->
<AuthorizeView Policy="CanEditUsers">
    <button @onclick="EditUser">Edit</button>
</AuthorizeView>
```

### Custom AuthenticationStateProvider (WASM)

```csharp
public class JwtAuthStateProvider : AuthenticationStateProvider
{
    private readonly ILocalStorageService _localStorage;
    private readonly HttpClient _httpClient;

    public JwtAuthStateProvider(
        ILocalStorageService localStorage, HttpClient httpClient)
    {
        _localStorage = localStorage;
        _httpClient = httpClient;
    }

    public override async Task<AuthenticationState> GetAuthenticationStateAsync()
    {
        var token = await _localStorage.GetItemAsync<string>("authToken");

        if (string.IsNullOrWhiteSpace(token))
            return new AuthenticationState(new ClaimsPrincipal(new ClaimsIdentity()));

        _httpClient.DefaultRequestHeaders.Authorization =
            new AuthenticationHeaderValue("Bearer", token);

        var claims = ParseClaimsFromJwt(token);
        var identity = new ClaimsIdentity(claims, "jwt");

        return new AuthenticationState(new ClaimsPrincipal(identity));
    }

    public void NotifyAuthStateChanged()
    {
        NotifyAuthenticationStateChanged(GetAuthenticationStateAsync());
    }

    private static IEnumerable<Claim> ParseClaimsFromJwt(string jwt)
    {
        var payload = jwt.Split('.')[1];
        var jsonBytes = Convert.FromBase64String(
            payload.PadRight(payload.Length + (4 - payload.Length % 4) % 4, '='));
        var pairs = JsonSerializer.Deserialize<Dictionary<string, object>>(jsonBytes)!;

        return pairs.Select(kvp => new Claim(kvp.Key, kvp.Value.ToString()!));
    }
}
```

## Error Handling

### ErrorBoundary in Layout

```razor
<article class="content px-4">
    <ErrorBoundary @ref="errorBoundary">
        <ChildContent>
            @Body
        </ChildContent>
        <ErrorContent Context="ex">
            <div class="alert alert-danger">
                <h4>Something went wrong</h4>
                <p>@ex.Message</p>
                <button class="btn btn-primary" @onclick="Recover">Try again</button>
            </div>
        </ErrorContent>
    </ErrorBoundary>
</article>

@code {
    private ErrorBoundary? errorBoundary;
    private void Recover() => errorBoundary?.Recover();
}
```

### Custom Error Component

```razor
@if (Error is not null)
{
    <div class="alert alert-danger">
        <h4>@Title</h4>
        <p>@Error</p>
        @if (OnRetry.HasDelegate)
        {
            <button class="btn btn-outline-danger btn-sm" @onclick="OnRetry">
                Retry
            </button>
        }
    </div>
}

@code {
    [Parameter] public string Title { get; set; } = "Error";
    [Parameter] public string? Error { get; set; }
    [Parameter] public EventCallback OnRetry { get; set; }
}
```

### Global Exception Handling (WASM)

```csharp
// In Program.cs -- log unhandled exceptions
builder.Logging.AddConfiguration(
    builder.Configuration.GetSection("Logging"));

// Custom logger that sends errors to API
public class ApiErrorLogger : ILogger
{
    private readonly HttpClient _http;

    public void Log<TState>(LogLevel logLevel, EventId eventId,
        TState state, Exception? exception, Func<TState, Exception?, string> formatter)
    {
        if (logLevel >= LogLevel.Error && exception is not null)
        {
            _ = _http.PostAsJsonAsync("api/errors", new
            {
                Message = exception.Message,
                StackTrace = exception.StackTrace,
                Timestamp = DateTime.UtcNow
            });
        }
    }
}
```

## Testing Patterns

### Page Test with Mocked Services

```csharp
public class UserListTests : TestContext
{
    private readonly Mock<IUserService> _userServiceMock;

    public UserListTests()
    {
        _userServiceMock = new Mock<IUserService>();
        Services.AddSingleton(_userServiceMock.Object);
    }

    [Fact]
    public async Task UserList_LoadsAndDisplaysUsers()
    {
        var users = new List<UserResponse>
        {
            new(1, "user1@example.com", "John", "Doe", "User", true, DateTime.UtcNow),
            new(2, "user2@example.com", "Jane", "Smith", "Admin", true, DateTime.UtcNow)
        };

        _userServiceMock.Setup(s => s.GetUsersAsync(1, 10))
            .ReturnsAsync(new PagedResponse<UserResponse>(users, 1, 10, 2, 1));

        var cut = RenderComponent<UserList>();
        await cut.InvokeAsync(() => Task.Delay(100));

        cut.FindAll(".card").Count.Should().Be(2);
    }

    [Fact]
    public async Task UserList_ShowsLoading_WhenFetching()
    {
        _userServiceMock.Setup(s => s.GetUsersAsync(1, 10))
            .Returns(async () =>
            {
                await Task.Delay(1000);
                return new PagedResponse<UserResponse>([], 1, 10, 0, 0);
            });

        var cut = RenderComponent<UserList>();

        cut.Find(".spinner-border").Should().NotBeNull();
    }

    [Fact]
    public async Task UserList_ShowsError_WhenServiceFails()
    {
        _userServiceMock.Setup(s => s.GetUsersAsync(1, 10))
            .ThrowsAsync(new HttpRequestException("Network error"));

        var cut = RenderComponent<UserList>();
        await cut.InvokeAsync(() => Task.Delay(100));

        cut.Find(".alert-danger").TextContent.Should().Contain("Network error");
    }
}
```

### Testing Event Callbacks

```csharp
[Fact]
public void UserCard_DeleteFlow_ShowsConfirmation_ThenDeletes()
{
    var user = new UserResponse(1, "t@e.com", "John", "Doe", "User", true, DateTime.UtcNow);
    var deleted = false;

    var cut = RenderComponent<UserCard>(p => p
        .Add(x => x.User, user)
        .Add(x => x.OnDelete,
            EventCallback.Factory.Create(this, () => deleted = true)));

    // Click delete -- should show confirmation modal
    cut.Find("button.btn-outline-danger").Click();
    cut.Find(".modal").Should().NotBeNull();
    cut.Find(".modal-body").TextContent.Should().Contain("Are you sure");

    // Confirm delete
    cut.Find(".modal .btn-danger").Click();
    deleted.Should().BeTrue();
}
```

### Testing Forms

```csharp
[Fact]
public void UserForm_ValidSubmit_CallsService()
{
    var submitted = false;
    Services.AddSingleton(Mock.Of<IUserService>(s =>
        s.CreateUserAsync(It.IsAny<CreateUserRequest>()) ==
        Task.FromResult(new UserResponse(1, "a@b.com", "A", "B", "User", true, DateTime.UtcNow))));

    var cut = RenderComponent<UserForm>();

    cut.Find("#email").Change("test@example.com");
    cut.Find("#password").Change("password123");
    cut.Find("#firstName").Change("John");
    cut.Find("#lastName").Change("Doe");

    cut.Find("form").Submit();

    // Verify navigation or toast (depending on implementation)
}
```

## Performance Patterns

### Virtualize for Large Lists

```razor
<Virtualize Items="@allUsers" Context="user" ItemSize="60">
    <ItemContent>
        <div class="user-row" style="height:60px;">
            @user.FirstName @user.LastName - @user.Email
        </div>
    </ItemContent>
    <Placeholder>
        <div class="user-row placeholder" style="height:60px;">
            Loading...
        </div>
    </Placeholder>
</Virtualize>
```

### Virtualize with ItemsProvider (Server-Side Paging)

```razor
<Virtualize ItemsProvider="LoadUsers" Context="user" ItemSize="60">
    <div class="user-row">@user.FirstName @user.LastName</div>
</Virtualize>

@code {
    private async ValueTask<ItemsProviderResult<UserResponse>> LoadUsers(
        ItemsProviderRequest request)
    {
        var result = await UserService.GetUsersAsync(
            request.StartIndex, request.Count);

        return new ItemsProviderResult<UserResponse>(
            result.Items, result.TotalCount);
    }
}
```

### Lazy Assembly Loading (WASM)

```csharp
// In router or page that needs a heavy assembly
@inject LazyAssemblyLoader AssemblyLoader

@code {
    private List<Assembly> _lazyLoadedAssemblies = [];

    protected override async Task OnInitializedAsync()
    {
        var assemblies = await AssemblyLoader.LoadAssembliesAsync(
            ["ChartLibrary.wasm", "ReportEngine.wasm"]);
        _lazyLoadedAssemblies.AddRange(assemblies);
    }
}
```

### Debounced Search Input

```razor
<input @oninput="HandleInput" placeholder="Search users..." />

@code {
    private CancellationTokenSource? _debounceToken;

    private async Task HandleInput(ChangeEventArgs e)
    {
        _debounceToken?.Cancel();
        _debounceToken = new CancellationTokenSource();

        try
        {
            await Task.Delay(300, _debounceToken.Token);
            await SearchAsync(e.Value?.ToString() ?? string.Empty);
        }
        catch (TaskCanceledException)
        {
            // Debounced -- ignore
        }
    }

    private async Task SearchAsync(string query)
    {
        users = await UserService.SearchAsync(query);
        StateHasChanged();
    }
}
```

## JavaScript Interop

### JS Module Import Pattern

```csharp
// Preferred: import JS modules instead of global scripts
public class ChartInterop : IAsyncDisposable
{
    private readonly Lazy<Task<IJSObjectReference>> _moduleTask;

    public ChartInterop(IJSRuntime jsRuntime)
    {
        _moduleTask = new(() => jsRuntime.InvokeAsync<IJSObjectReference>(
            "import", "./js/chart.js").AsTask());
    }

    public async Task RenderChartAsync(string elementId, object data)
    {
        var module = await _moduleTask.Value;
        await module.InvokeVoidAsync("renderChart", elementId, data);
    }

    public async Task DestroyChartAsync(string elementId)
    {
        var module = await _moduleTask.Value;
        await module.InvokeVoidAsync("destroyChart", elementId);
    }

    public async ValueTask DisposeAsync()
    {
        if (_moduleTask.IsValueCreated)
        {
            var module = await _moduleTask.Value;
            await module.DisposeAsync();
        }
    }
}
```

### Calling .NET from JavaScript

```javascript
// wwwroot/js/interop.js
window.blazorCallbacks = {
    registerResizeHandler: function (dotNetRef) {
        window.addEventListener('resize', () => {
            dotNetRef.invokeMethodAsync('OnWindowResize', window.innerWidth);
        });
    }
};
```

```razor
@inject IJSRuntime JS
@implements IDisposable

@code {
    private DotNetObjectReference<MyComponent>? _dotNetRef;

    protected override async Task OnAfterRenderAsync(bool firstRender)
    {
        if (firstRender)
        {
            _dotNetRef = DotNetObjectReference.Create(this);
            await JS.InvokeVoidAsync(
                "blazorCallbacks.registerResizeHandler", _dotNetRef);
        }
    }

    [JSInvokable]
    public Task OnWindowResize(int width)
    {
        // Handle resize
        StateHasChanged();
        return Task.CompletedTask;
    }

    public void Dispose()
    {
        _dotNetRef?.Dispose();
    }
}
```

### File Download via JS Interop

```javascript
// wwwroot/js/interop.js
window.blazorInterop = {
    downloadFile: function (filename, contentType, content) {
        const blob = new Blob([content], { type: contentType });
        const url = URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        link.download = filename;
        link.click();
        URL.revokeObjectURL(url);
    }
};
```
