# Android Compose Patterns Reference

## Contents

- [UI Composition Patterns](#ui-composition-patterns)
- [Animation Patterns](#animation-patterns)
- [Performance Patterns](#performance-patterns)
- [Accessibility Patterns](#accessibility-patterns)
- [Testing Recipes](#testing-recipes)
- [Data Layer Integration](#data-layer-integration)

## UI Composition Patterns

### Slot-Based Component API

Use lambda slots instead of fixed content to make components flexible and reusable.

```kotlin
@Composable
fun AppScaffold(
    topBar: @Composable () -> Unit = {},
    bottomBar: @Composable () -> Unit = {},
    floatingActionButton: @Composable () -> Unit = {},
    content: @Composable (PaddingValues) -> Unit,
) {
    Scaffold(
        topBar = topBar,
        bottomBar = bottomBar,
        floatingActionButton = floatingActionButton,
        content = content,
    )
}

// Usage -- caller decides what goes in each slot
AppScaffold(
    topBar = { TopAppBar(title = { Text("Home") }) },
    floatingActionButton = {
        FloatingActionButton(onClick = onAdd) {
            Icon(Icons.Default.Add, contentDescription = "Add item")
        }
    },
) { padding ->
    UserList(modifier = Modifier.padding(padding), users = users)
}
```

### State Holder Pattern

Extract complex UI logic into a plain class that survives recomposition via `remember`.

```kotlin
class SearchBarState(
    initialQuery: String = "",
    private val onSearch: (String) -> Unit,
) {
    var query by mutableStateOf(initialQuery)
        private set
    var isExpanded by mutableStateOf(false)
        private set

    fun onQueryChange(newQuery: String) {
        query = newQuery
    }

    fun expand() { isExpanded = true }
    fun collapse() { isExpanded = false; query = "" }

    fun submitSearch() {
        if (query.isNotBlank()) onSearch(query)
    }
}

@Composable
fun rememberSearchBarState(
    initialQuery: String = "",
    onSearch: (String) -> Unit,
): SearchBarState = remember(onSearch) {
    SearchBarState(initialQuery, onSearch)
}

@Composable
fun SearchBar(state: SearchBarState, modifier: Modifier = Modifier) {
    AnimatedVisibility(visible = state.isExpanded) {
        OutlinedTextField(
            value = state.query,
            onValueChange = state::onQueryChange,
            modifier = modifier.fillMaxWidth(),
            placeholder = { Text("Search...") },
            singleLine = true,
            trailingIcon = {
                IconButton(onClick = state::collapse) {
                    Icon(Icons.Default.Close, contentDescription = "Close search")
                }
            },
            keyboardActions = KeyboardActions(onSearch = { state.submitSearch() }),
        )
    }
}
```

### Pull-to-Refresh Pattern

```kotlin
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RefreshableUserList(
    users: List<User>,
    isRefreshing: Boolean,
    onRefresh: () -> Unit,
    onUserClick: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    val pullRefreshState = rememberPullToRefreshState()

    PullToRefreshBox(
        isRefreshing = isRefreshing,
        onRefresh = onRefresh,
        state = pullRefreshState,
        modifier = modifier,
    ) {
        LazyColumn(
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            items(items = users, key = { it.id }) { user ->
                UserCard(user = user, onClick = { onUserClick(user.id) })
            }
        }
    }
}
```

### Bottom Sheet Pattern

```kotlin
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun UserActionSheet(
    user: User,
    onDismiss: () -> Unit,
    onEdit: () -> Unit,
    onDelete: () -> Unit,
) {
    val sheetState = rememberModalBottomSheetState()

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Text(
                text = user.name,
                style = MaterialTheme.typography.titleLarge,
                modifier = Modifier.padding(bottom = 16.dp),
            )
            ListItem(
                headlineContent = { Text("Edit") },
                leadingContent = { Icon(Icons.Default.Edit, contentDescription = null) },
                modifier = Modifier.clickable(onClick = onEdit),
            )
            ListItem(
                headlineContent = { Text("Delete") },
                leadingContent = {
                    Icon(Icons.Default.Delete, contentDescription = null,
                        tint = MaterialTheme.colorScheme.error)
                },
                modifier = Modifier.clickable(onClick = onDelete),
            )
        }
    }
}
```

### Form Validation Pattern

```kotlin
data class FormField(
    val value: String = "",
    val error: String? = null,
) {
    val isValid: Boolean get() = error == null && value.isNotBlank()
}

class LoginFormState {
    var email by mutableStateOf(FormField())
        private set
    var password by mutableStateOf(FormField())
        private set

    val isValid: Boolean get() = email.isValid && password.isValid

    fun onEmailChange(value: String) {
        email = FormField(
            value = value,
            error = when {
                value.isBlank() -> "Email is required"
                !Patterns.EMAIL_ADDRESS.matcher(value).matches() -> "Invalid email format"
                else -> null
            },
        )
    }

    fun onPasswordChange(value: String) {
        password = FormField(
            value = value,
            error = when {
                value.isBlank() -> "Password is required"
                value.length < 8 -> "Password must be at least 8 characters"
                else -> null
            },
        )
    }
}

@Composable
fun ValidatedTextField(
    field: FormField,
    onValueChange: (String) -> Unit,
    label: String,
    modifier: Modifier = Modifier,
    keyboardOptions: KeyboardOptions = KeyboardOptions.Default,
    visualTransformation: VisualTransformation = VisualTransformation.None,
) {
    Column(modifier = modifier) {
        OutlinedTextField(
            value = field.value,
            onValueChange = onValueChange,
            label = { Text(label) },
            isError = field.error != null,
            keyboardOptions = keyboardOptions,
            visualTransformation = visualTransformation,
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )
        field.error?.let { errorText ->
            Text(
                text = errorText,
                color = MaterialTheme.colorScheme.error,
                style = MaterialTheme.typography.bodySmall,
                modifier = Modifier.padding(start = 16.dp, top = 4.dp),
            )
        }
    }
}
```

## Animation Patterns

### Animated Visibility with Enter/Exit

```kotlin
@Composable
fun AnimatedBanner(
    visible: Boolean,
    message: String,
    modifier: Modifier = Modifier,
) {
    AnimatedVisibility(
        visible = visible,
        enter = slideInVertically(initialOffsetY = { -it }) + fadeIn(),
        exit = slideOutVertically(targetOffsetY = { -it }) + fadeOut(),
        modifier = modifier,
    ) {
        Surface(
            color = MaterialTheme.colorScheme.primaryContainer,
            modifier = Modifier.fillMaxWidth(),
        ) {
            Text(
                text = message,
                modifier = Modifier.padding(16.dp),
                style = MaterialTheme.typography.bodyMedium,
            )
        }
    }
}
```

### Animated Content Transitions

```kotlin
@Composable
fun AnimatedCounter(count: Int, modifier: Modifier = Modifier) {
    AnimatedContent(
        targetState = count,
        transitionSpec = {
            if (targetState > initialState) {
                slideInVertically { it } + fadeIn() togetherWith
                    slideOutVertically { -it } + fadeOut()
            } else {
                slideInVertically { -it } + fadeIn() togetherWith
                    slideOutVertically { it } + fadeOut()
            }.using(SizeTransform(clip = false))
        },
        label = "counter_animation",
        modifier = modifier,
    ) { targetCount ->
        Text(
            text = "$targetCount",
            style = MaterialTheme.typography.headlineMedium,
        )
    }
}
```

### Infinite Animations

```kotlin
@Composable
fun PulsingDot(modifier: Modifier = Modifier) {
    val infiniteTransition = rememberInfiniteTransition(label = "pulse")
    val scale by infiniteTransition.animateFloat(
        initialValue = 0.8f,
        targetValue = 1.2f,
        animationSpec = infiniteRepeatable(
            animation = tween(durationMillis = 800, easing = EaseInOutCubic),
            repeatMode = RepeatMode.Reverse,
        ),
        label = "pulse_scale",
    )

    Box(
        modifier = modifier
            .size(12.dp)
            .graphicsLayer { scaleX = scale; scaleY = scale }
            .background(MaterialTheme.colorScheme.primary, CircleShape),
    )
}
```

### Shared Element Transitions (Navigation)

```kotlin
// Using Compose Navigation shared element transitions (1.7+)
@Composable
fun SharedElementExample(navController: NavHostController) {
    NavHost(navController, startDestination = "list") {
        composable("list") {
            LazyColumn {
                items(items) { item ->
                    SharedTransitionLayout {
                        AnimatedVisibilityScope {
                            Image(
                                painter = rememberAsyncImagePainter(item.imageUrl),
                                contentDescription = item.title,
                                modifier = Modifier
                                    .sharedElement(
                                        state = rememberSharedContentState(key = "image_${item.id}"),
                                        animatedVisibilityScope = this@AnimatedVisibilityScope,
                                    )
                                    .clickable { navController.navigate("detail/${item.id}") },
                            )
                        }
                    }
                }
            }
        }
    }
}
```

## Performance Patterns

### Stable Keys in LazyColumn

Always provide stable, unique keys to `items()` for correct diffing and animation.

```kotlin
LazyColumn {
    items(
        items = users,
        key = { user -> user.id }, // stable unique key
    ) { user ->
        UserCard(user = user)
    }
}
```

### Deferred Reading with Lambda Modifiers

Defer state reads to the layout/draw phase to skip unnecessary recompositions.

```kotlin
// BAD: recomposes on every scroll offset change
Box(modifier = Modifier.offset(y = scrollState.value.dp))

// GOOD: defers read to layout phase, no recomposition
Box(modifier = Modifier.offset { IntOffset(0, scrollState.value) })
```

### derivedStateOf for Expensive Computations

```kotlin
@Composable
fun FilteredList(allItems: List<Item>, searchQuery: String) {
    // Only recomputes when allItems or searchQuery actually change
    val filteredItems by remember(allItems, searchQuery) {
        derivedStateOf {
            allItems.filter { it.name.contains(searchQuery, ignoreCase = true) }
        }
    }
    LazyColumn {
        items(filteredItems, key = { it.id }) { item -> ItemRow(item) }
    }
}
```

### Immutable and Stable Annotations

Help the Compose compiler skip recomposition of unchanged parameters.

```kotlin
@Immutable
data class ChartData(
    val points: List<DataPoint>,
    val label: String,
)

@Stable
class ThemeConfiguration(
    val primaryColor: Color,
    val isDarkMode: Boolean,
)
```

### Image Loading Best Practices

```kotlin
@Composable
fun UserAvatar(avatarUrl: String?, modifier: Modifier = Modifier) {
    AsyncImage(
        model = ImageRequest.Builder(LocalContext.current)
            .data(avatarUrl)
            .crossfade(true)
            .memoryCachePolicy(CachePolicy.ENABLED)
            .diskCachePolicy(CachePolicy.ENABLED)
            .build(),
        contentDescription = "User avatar",
        placeholder = painterResource(R.drawable.avatar_placeholder),
        error = painterResource(R.drawable.avatar_placeholder),
        contentScale = ContentScale.Crop,
        modifier = modifier
            .size(48.dp)
            .clip(CircleShape),
    )
}
```

## Accessibility Patterns

### Content Descriptions

```kotlin
// Decorative icon (no description needed, screenreader skips it)
Icon(Icons.Default.Star, contentDescription = null)

// Functional icon (screenreader announces action)
IconButton(onClick = onDelete) {
    Icon(Icons.Default.Delete, contentDescription = "Delete user ${user.name}")
}

// Merge child semantics for a card that acts as a single element
Card(
    modifier = Modifier
        .semantics(mergeDescendants = true) {}
        .clickable(onClick = onClick),
) {
    Text(user.name)
    Text(user.email)
}
```

### Touch Target Sizing

```kotlin
// Minimum 48dp touch target per Material guidelines
IconButton(
    onClick = onAction,
    modifier = Modifier.sizeIn(minWidth = 48.dp, minHeight = 48.dp),
) {
    Icon(Icons.Default.Info, contentDescription = "Info")
}
```

### Custom Semantics for Complex Widgets

```kotlin
@Composable
fun RatingBar(
    rating: Float,
    onRatingChange: (Float) -> Unit,
    modifier: Modifier = Modifier,
) {
    Row(
        modifier = modifier.semantics {
            stateDescription = "$rating out of 5 stars"
            contentDescription = "Rating"
            role = Role.Slider
        },
    ) {
        (1..5).forEach { star ->
            Icon(
                imageVector = if (star <= rating) Icons.Filled.Star else Icons.Outlined.Star,
                contentDescription = null, // parent merges
                tint = if (star <= rating) Color(0xFFFFD700) else Color.Gray,
                modifier = Modifier
                    .size(32.dp)
                    .clickable { onRatingChange(star.toFloat()) },
            )
        }
    }
}
```

### Screen Reader Announcements

```kotlin
@Composable
fun StatusBanner(message: String, isError: Boolean) {
    val announcement = if (isError) "Error: $message" else message

    // Announce changes to screen readers
    LaunchedEffect(announcement) {
        // AccessibilityManager handles this via LiveRegion
    }

    Text(
        text = message,
        modifier = Modifier.semantics { liveRegion = LiveRegionMode.Polite },
        color = if (isError) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.primary,
    )
}
```

## Testing Recipes

### Testing Navigation

```kotlin
@HiltAndroidTest
class NavigationTest {
    @get:Rule(order = 0) val hiltRule = HiltAndroidRule(this)
    @get:Rule(order = 1) val composeTestRule = createAndroidComposeRule<MainActivity>()

    @Test
    fun navigates_to_detail_on_user_click() {
        // Setup: ensure HomeScreen shows a user
        composeTestRule.onNodeWithText("Alice").performClick()

        // Assert: DetailScreen is now displayed
        composeTestRule.onNodeWithText("User Details").assertIsDisplayed()
    }
}
```

### Testing with Fake ViewModels

```kotlin
class FakeHomeViewModel : HomeViewModel(mockk()) {
    private val _uiState = MutableStateFlow<HomeUiState>(HomeUiState.Loading)
    override val uiState: StateFlow<HomeUiState> = _uiState.asStateFlow()

    fun emitState(state: HomeUiState) { _uiState.value = state }
}

@Test
fun shows_error_with_retry_button() {
    val fakeViewModel = FakeHomeViewModel()
    fakeViewModel.emitState(HomeUiState.Error("Network timeout"))

    composeTestRule.setContent {
        MyAppTheme { HomeScreen(viewModel = fakeViewModel, onUserClick = {}, onLogout = {}) }
    }

    composeTestRule.onNodeWithText("Network timeout").assertIsDisplayed()
    composeTestRule.onNodeWithText("Retry").assertIsDisplayed().performClick()
}
```

### Testing Asynchronous UI Updates

```kotlin
@Test
fun shows_loading_then_content() = runTest {
    val viewModel = FakeHomeViewModel()
    viewModel.emitState(HomeUiState.Loading)

    composeTestRule.setContent {
        MyAppTheme { HomeScreen(viewModel = viewModel) }
    }

    // Loading state
    composeTestRule.onNode(hasTestTag("loading")).assertIsDisplayed()

    // Transition to loaded
    viewModel.emitState(HomeUiState.Success(listOf(testUser)))
    composeTestRule.waitForIdle()

    composeTestRule.onNodeWithText("Alice").assertIsDisplayed()
    composeTestRule.onNode(hasTestTag("loading")).assertDoesNotExist()
}
```

### Screenshot Testing (Roborazzi / Paparazzi)

```kotlin
// Using Paparazzi for snapshot tests (no device needed)
class UserCardSnapshotTest {
    @get:Rule val paparazzi = Paparazzi(
        deviceConfig = DeviceConfig.PIXEL_5,
        theme = "Theme.Material3.DayNight",
    )

    @Test
    fun userCard_default() {
        paparazzi.snapshot {
            MyAppTheme {
                UserCard(
                    user = User("1", "alice@example.com", "Alice"),
                    onClick = {},
                    onDeleteClick = {},
                )
            }
        }
    }

    @Test
    fun userCard_darkTheme() {
        paparazzi.snapshot {
            MyAppTheme(darkTheme = true) {
                UserCard(
                    user = User("1", "alice@example.com", "Alice"),
                    onClick = {},
                    onDeleteClick = {},
                )
            }
        }
    }
}
```

## Data Layer Integration

### Repository with Offline-First Strategy

```kotlin
@Singleton
class UserRepository @Inject constructor(
    private val apiService: ApiService,
    private val userDao: UserDao,
) {
    fun observeUsers(): Flow<List<User>> = userDao.observeAll()
        .onStart { refreshFromNetwork() }
        .catch { emit(emptyList()) }

    private suspend fun refreshFromNetwork() {
        try {
            val remoteUsers = apiService.getUsers()
            userDao.insertAll(remoteUsers.map { it.toEntity() })
        } catch (e: Exception) {
            // Network failure is non-fatal; local cache is used
            Timber.w(e, "Failed to refresh users from network")
        }
    }
}
```

### Room Database with Flow

```kotlin
@Database(entities = [UserEntity::class], version = 1)
abstract class AppDatabase : RoomDatabase() {
    abstract fun userDao(): UserDao
}

@Dao
interface UserDao {
    @Query("SELECT * FROM users ORDER BY name ASC")
    fun observeAll(): Flow<List<UserEntity>>

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertAll(users: List<UserEntity>)

    @Delete
    suspend fun delete(user: UserEntity)
}

@Entity(tableName = "users")
data class UserEntity(
    @PrimaryKey val id: String,
    val email: String,
    val name: String,
    val avatarUrl: String? = null,
)
```

### DataStore for Preferences

```kotlin
@Singleton
class UserPreferences @Inject constructor(
    private val dataStore: DataStore<Preferences>,
) {
    companion object {
        private val DARK_MODE_KEY = booleanPreferencesKey("dark_mode")
        private val AUTH_TOKEN_KEY = stringPreferencesKey("auth_token")
    }

    val isDarkMode: Flow<Boolean> = dataStore.data
        .map { prefs -> prefs[DARK_MODE_KEY] ?: false }
        .catch { emit(false) }

    suspend fun setDarkMode(enabled: Boolean) {
        dataStore.edit { prefs -> prefs[DARK_MODE_KEY] = enabled }
    }

    suspend fun setAuthToken(token: String) {
        dataStore.edit { prefs -> prefs[AUTH_TOKEN_KEY] = token }
    }

    suspend fun clearAuthToken() {
        dataStore.edit { prefs -> prefs.remove(AUTH_TOKEN_KEY) }
    }
}
```

### Hilt Network Module

```kotlin
@Module
@InstallIn(SingletonComponent::class)
object NetworkModule {

    @Provides
    @Singleton
    fun provideJson(): Json = Json {
        ignoreUnknownKeys = true
        isLenient = true
        encodeDefaults = true
    }

    @Provides
    @Singleton
    fun provideOkHttpClient(): OkHttpClient =
        OkHttpClient.Builder()
            .connectTimeout(30, TimeUnit.SECONDS)
            .readTimeout(30, TimeUnit.SECONDS)
            .addInterceptor(HttpLoggingInterceptor().apply {
                level = HttpLoggingInterceptor.Level.BODY
            })
            .build()

    @Provides
    @Singleton
    fun provideRetrofit(client: OkHttpClient, json: Json): Retrofit =
        Retrofit.Builder()
            .baseUrl("https://api.example.com/")
            .client(client)
            .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
            .build()

    @Provides
    @Singleton
    fun provideApiService(retrofit: Retrofit): ApiService =
        retrofit.create(ApiService::class.java)
}
```
