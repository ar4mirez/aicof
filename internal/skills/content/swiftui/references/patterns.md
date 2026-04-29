# SwiftUI Advanced Patterns

Detailed patterns, animations, Combine integration, Core Data/SwiftData, testing, accessibility, and platform adaptations for SwiftUI development.

---

## View Patterns

### Reusable Stat Component

```swift
struct StatView: View {
    let title: String
    let value: Int

    var body: some View {
        VStack(spacing: 4) {
            Text("\(value)").font(.headline)
            Text(title).font(.caption).foregroundStyle(.secondary)
        }
    }
}
```

### Loading Overlay Modifier

```swift
struct LoadingModifier: ViewModifier {
    let isLoading: Bool

    func body(content: Content) -> some View {
        ZStack {
            content
                .disabled(isLoading)
                .blur(radius: isLoading ? 2 : 0)

            if isLoading {
                ProgressView()
                    .scaleEffect(1.5)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                    .background(Color.black.opacity(0.2))
            }
        }
    }
}

extension View {
    func loading(_ isLoading: Bool) -> some View {
        modifier(LoadingModifier(isLoading: isLoading))
    }
}
```

### Conditional Modifier Pattern

```swift
extension View {
    @ViewBuilder
    func `if`<Content: View>(_ condition: Bool, transform: (Self) -> Content) -> some View {
        if condition {
            transform(self)
        } else {
            self
        }
    }
}

// Usage
Text("Hello")
    .if(isHighlighted) { $0.foregroundStyle(.yellow).fontWeight(.bold) }
```

### Custom Row View

```swift
struct PostRowView: View {
    let post: Post

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(post.title).font(.headline)
                Spacer()
                if post.isFavorite {
                    Image(systemName: "star.fill").foregroundStyle(.yellow)
                }
            }

            Text(post.excerpt)
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .lineLimit(2)

            HStack {
                Label("\(post.likeCount)", systemImage: "heart")
                Label("\(post.commentCount)", systemImage: "bubble.right")
                Spacer()
                Text(post.createdAt, style: .relative)
            }
            .font(.caption)
            .foregroundStyle(.tertiary)
        }
        .padding(.vertical, 4)
    }
}
```

### LazyVGrid with Adaptive Columns

```swift
struct PhotoGridView: View {
    let photos: [Photo]

    private let columns = [
        GridItem(.adaptive(minimum: 100, maximum: 200), spacing: 4)
    ]

    var body: some View {
        ScrollView {
            LazyVGrid(columns: columns, spacing: 4) {
                ForEach(photos) { photo in
                    AsyncImage(url: photo.thumbnailURL) { image in
                        image.resizable().aspectRatio(1, contentMode: .fill)
                    } placeholder: {
                        Rectangle().fill(Color.gray.opacity(0.3))
                    }
                    .aspectRatio(1, contentMode: .fit)
                    .clipped()
                }
            }
            .padding(4)
        }
    }
}
```

### Horizontal Scroll with Pinned Section Headers

```swift
struct CategoryView: View {
    let categories: [Category]

    var body: some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 24, pinnedViews: .sectionHeaders) {
                ForEach(categories) { category in
                    Section {
                        ScrollView(.horizontal, showsIndicators: false) {
                            LazyHStack(spacing: 12) {
                                ForEach(category.items) { item in
                                    ItemCardView(item: item).frame(width: 150)
                                }
                            }
                            .padding(.horizontal)
                        }
                    } header: {
                        Text(category.name)
                            .font(.title2).fontWeight(.bold)
                            .padding(.horizontal)
                            .padding(.vertical, 8)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .background(.bar)
                    }
                }
            }
        }
    }
}
```

---

## Animation Patterns

### Implicit Animations

```swift
struct AnimatedCard: View {
    @State private var isExpanded = false

    var body: some View {
        VStack {
            Text("Card Content")
                .frame(height: isExpanded ? 200 : 100)
                .frame(maxWidth: .infinity)
                .background(Color.blue.opacity(0.2))
                .cornerRadius(12)
        }
        .animation(.spring(response: 0.4, dampingFraction: 0.7), value: isExpanded)
        .onTapGesture { isExpanded.toggle() }
    }
}
```

### Explicit Animations with withAnimation

```swift
struct AnimatedList: View {
    @State private var items: [Item] = []

    var body: some View {
        List {
            ForEach(items) { item in
                ItemRow(item: item)
                    .transition(.asymmetric(
                        insertion: .slide.combined(with: .opacity),
                        removal: .opacity
                    ))
            }
        }
    }

    private func addItem(_ item: Item) {
        withAnimation(.easeInOut(duration: 0.3)) {
            items.append(item)
        }
    }

    private func removeItem(_ item: Item) {
        withAnimation(.easeOut(duration: 0.2)) {
            items.removeAll { $0.id == item.id }
        }
    }
}
```

### Phase Animator (iOS 17+)

```swift
struct PulsingDot: View {
    var body: some View {
        Circle()
            .fill(.blue)
            .frame(width: 20, height: 20)
            .phaseAnimator([false, true]) { content, phase in
                content
                    .scaleEffect(phase ? 1.5 : 1.0)
                    .opacity(phase ? 0.5 : 1.0)
            } animation: { phase in
                .easeInOut(duration: 0.8)
            }
    }
}
```

### Keyframe Animator (iOS 17+)

```swift
struct BouncingView: View {
    @State private var trigger = false

    var body: some View {
        Image(systemName: "bell.fill")
            .font(.largeTitle)
            .keyframeAnimator(initialValue: AnimationValues(), trigger: trigger) { content, value in
                content
                    .scaleEffect(value.scale)
                    .rotationEffect(value.angle)
            } keyframes: { _ in
                KeyframeTrack(\.scale) {
                    SpringKeyframe(1.2, duration: 0.15)
                    SpringKeyframe(0.9, duration: 0.15)
                    SpringKeyframe(1.0, duration: 0.2)
                }
                KeyframeTrack(\.angle) {
                    SpringKeyframe(.degrees(-15), duration: 0.1)
                    SpringKeyframe(.degrees(15), duration: 0.1)
                    SpringKeyframe(.degrees(0), duration: 0.2)
                }
            }
            .onTapGesture { trigger.toggle() }
    }
}

struct AnimationValues {
    var scale: CGFloat = 1.0
    var angle: Angle = .zero
}
```

### Matched Geometry Effect

```swift
struct HeroAnimationView: View {
    @Namespace private var animation
    @State private var isExpanded = false
    @State private var selectedItem: Item?

    var body: some View {
        ZStack {
            if let item = selectedItem {
                expandedView(item: item)
            } else {
                gridView
            }
        }
        .animation(.spring(response: 0.35, dampingFraction: 0.85), value: selectedItem)
    }

    private var gridView: some View {
        LazyVGrid(columns: [GridItem(.adaptive(minimum: 150))]) {
            ForEach(items) { item in
                ItemCard(item: item)
                    .matchedGeometryEffect(id: item.id, in: animation)
                    .onTapGesture { selectedItem = item }
            }
        }
    }

    private func expandedView(item: Item) -> some View {
        ItemDetailView(item: item)
            .matchedGeometryEffect(id: item.id, in: animation)
            .onTapGesture { selectedItem = nil }
    }
}
```

---

## Combine Integration

### Publisher-Based View Model (iOS 14-16)

```swift
import Combine

final class SearchViewModel: ObservableObject {
    @Published var query = ""
    @Published var results: [SearchResult] = []
    @Published var isSearching = false

    private var cancellables = Set<AnyCancellable>()
    private let searchService: SearchServiceProtocol

    init(searchService: SearchServiceProtocol = SearchService()) {
        self.searchService = searchService
        setupSearch()
    }

    private func setupSearch() {
        $query
            .debounce(for: .milliseconds(300), scheduler: RunLoop.main)
            .removeDuplicates()
            .filter { !$0.isEmpty }
            .sink { [weak self] query in
                Task { await self?.search(query: query) }
            }
            .store(in: &cancellables)
    }

    @MainActor
    private func search(query: String) async {
        isSearching = true
        defer { isSearching = false }

        do {
            results = try await searchService.search(query: query)
        } catch {
            results = []
        }
    }
}

struct SearchView: View {
    @StateObject private var viewModel = SearchViewModel()

    var body: some View {
        List(viewModel.results) { result in
            Text(result.title)
        }
        .searchable(text: $viewModel.query)
        .overlay {
            if viewModel.isSearching { ProgressView() }
        }
    }
}
```

### Timer Publisher

```swift
struct CountdownView: View {
    @State private var timeRemaining = 60

    let timer = Timer.publish(every: 1, on: .main, in: .common).autoconnect()

    var body: some View {
        Text("Time: \(timeRemaining)")
            .font(.largeTitle)
            .onReceive(timer) { _ in
                if timeRemaining > 0 { timeRemaining -= 1 }
            }
    }
}
```

---

## Core Data Integration

### Data Controller

```swift
import CoreData

final class DataController: ObservableObject {
    let container: NSPersistentContainer

    init(inMemory: Bool = false) {
        container = NSPersistentContainer(name: "MyApp")

        if inMemory {
            container.persistentStoreDescriptions.first?.url = URL(fileURLWithPath: "/dev/null")
        }

        container.loadPersistentStores { _, error in
            if let error { fatalError("Core Data failed: \(error)") }
        }
        container.viewContext.automaticallyMergesChangesFromParent = true
        container.viewContext.mergePolicy = NSMergeByPropertyObjectTrumpMergePolicy
    }

    func save() {
        let context = container.viewContext
        guard context.hasChanges else { return }
        try? context.save()
    }

    static var preview: DataController {
        let controller = DataController(inMemory: true)
        // Add preview data here
        return controller
    }
}
```

### Core Data View

```swift
struct TaskListView: View {
    @Environment(\.managedObjectContext) private var viewContext

    @FetchRequest(
        sortDescriptors: [NSSortDescriptor(keyPath: \Task.createdAt, ascending: false)],
        predicate: NSPredicate(format: "isCompleted == %@", NSNumber(value: false)),
        animation: .default
    )
    private var tasks: FetchedResults<Task>

    var body: some View {
        List {
            ForEach(tasks) { task in
                TaskRowView(task: task)
            }
            .onDelete(perform: deleteTasks)
        }
    }

    private func deleteTasks(offsets: IndexSet) {
        withAnimation {
            offsets.map { tasks[$0] }.forEach(viewContext.delete)
            try? viewContext.save()
        }
    }
}
```

### SwiftData (iOS 17+)

```swift
import SwiftData

@Model
final class Note {
    var title: String
    var content: String
    var createdAt: Date
    var tags: [Tag]

    init(title: String, content: String = "") {
        self.title = title
        self.content = content
        self.createdAt = .now
        self.tags = []
    }
}

struct NoteListView: View {
    @Query(sort: \Note.createdAt, order: .reverse)
    private var notes: [Note]

    @Environment(\.modelContext) private var modelContext

    var body: some View {
        List(notes) { note in
            NavigationLink(value: note) {
                VStack(alignment: .leading) {
                    Text(note.title).font(.headline)
                    Text(note.createdAt, style: .relative).font(.caption)
                }
            }
        }
        .toolbar {
            Button("Add", systemImage: "plus") {
                let note = Note(title: "New Note")
                modelContext.insert(note)
            }
        }
    }
}
```

---

## Networking

### Async/Await API Client

```swift
protocol APIClientProtocol {
    func fetch<T: Decodable>(_ endpoint: Endpoint) async throws -> T
    func send<T: Decodable, U: Encodable>(_ endpoint: Endpoint, body: U) async throws -> T
}

final class APIClient: APIClientProtocol {
    private let session: URLSession
    private let decoder: JSONDecoder
    private let baseURL: URL

    init(baseURL: URL = URL(string: "https://api.example.com")!, session: URLSession = .shared) {
        self.baseURL = baseURL
        self.session = session
        self.decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        decoder.dateDecodingStrategy = .iso8601
    }

    func fetch<T: Decodable>(_ endpoint: Endpoint) async throws -> T {
        let request = try buildRequest(for: endpoint)
        let (data, response) = try await session.data(for: request)
        try validateResponse(response)
        return try decoder.decode(T.self, from: data)
    }

    func send<T: Decodable, U: Encodable>(_ endpoint: Endpoint, body: U) async throws -> T {
        var request = try buildRequest(for: endpoint)
        request.httpBody = try JSONEncoder().encode(body)
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        let (data, response) = try await session.data(for: request)
        try validateResponse(response)
        return try decoder.decode(T.self, from: data)
    }

    private func buildRequest(for endpoint: Endpoint) throws -> URLRequest {
        guard let url = URL(string: endpoint.path, relativeTo: baseURL) else {
            throw APIError.invalidURL
        }
        var request = URLRequest(url: url)
        request.httpMethod = endpoint.method.rawValue
        request.timeoutInterval = 30
        return request
    }

    private func validateResponse(_ response: URLResponse) throws {
        guard let http = response as? HTTPURLResponse,
              (200...299).contains(http.statusCode) else {
            throw APIError.invalidResponse
        }
    }
}

struct Endpoint {
    let path: String
    let method: HTTPMethod
    enum HTTPMethod: String { case get = "GET", post = "POST", put = "PUT", delete = "DELETE" }
}

enum APIError: LocalizedError {
    case invalidURL, invalidResponse, httpError(statusCode: Int)
    var errorDescription: String? {
        switch self {
        case .invalidURL: "Invalid URL"
        case .invalidResponse: "Invalid response from server"
        case .httpError(let code): "HTTP error: \(code)"
        }
    }
}
```

---

## Testing Patterns

### View Model Unit Testing

```swift
import XCTest
@testable import MyApp

final class UserViewModelTests: XCTestCase {
    var sut: UserViewModel!
    var mockService: MockUserService!

    override func setUp() {
        super.setUp()
        mockService = MockUserService()
        sut = UserViewModel(userService: mockService)
    }

    override func tearDown() {
        sut = nil
        mockService = nil
        super.tearDown()
    }

    func test_loadUser_success_updatesUser() async {
        let expectedUser = User(id: "1", name: "John", email: "john@example.com")
        mockService.userToReturn = expectedUser

        await sut.loadUser(id: "1")

        XCTAssertEqual(sut.user, expectedUser)
        XCTAssertFalse(sut.isLoading)
        XCTAssertNil(sut.errorMessage)
    }

    func test_loadUser_failure_setsErrorMessage() async {
        mockService.errorToThrow = APIError.invalidResponse

        await sut.loadUser(id: "1")

        XCTAssertNil(sut.user)
        XCTAssertFalse(sut.isLoading)
        XCTAssertNotNil(sut.errorMessage)
    }
}

final class MockUserService: UserServiceProtocol {
    var userToReturn: User?
    var errorToThrow: Error?

    func fetchUser(id: String) async throws -> User {
        if let error = errorToThrow { throw error }
        guard let user = userToReturn else { throw APIError.invalidResponse }
        return user
    }
}
```

### UI Testing

```swift
import XCTest

final class LoginUITests: XCTestCase {
    var app: XCUIApplication!

    override func setUp() {
        super.setUp()
        continueAfterFailure = false
        app = XCUIApplication()
        app.launchArguments = ["UI_TESTING"]
        app.launch()
    }

    func test_login_withValidCredentials_navigatesToHome() {
        let emailField = app.textFields["Email"]
        emailField.tap()
        emailField.typeText("test@example.com")

        let passwordField = app.secureTextFields["Password"]
        passwordField.tap()
        passwordField.typeText("password123")

        app.buttons["Login"].tap()

        XCTAssertTrue(app.navigationBars["Home"].waitForExistence(timeout: 5))
    }

    func test_login_withInvalidCredentials_showsError() {
        let emailField = app.textFields["Email"]
        emailField.tap()
        emailField.typeText("invalid")

        XCTAssertFalse(app.buttons["Login"].isEnabled)
    }
}
```

### Preview Testing Strategy

Provide multiple preview configurations to catch layout issues early:

```swift
#Preview("Default") {
    UserProfileView(user: .preview)
}

#Preview("Dark Mode") {
    UserProfileView(user: .preview)
        .preferredColorScheme(.dark)
}

#Preview("iPhone SE") {
    UserProfileView(user: .preview)
        .previewDevice("iPhone SE (3rd generation)")
}

#Preview("Large Text") {
    UserProfileView(user: .preview)
        .environment(\.sizeCategory, .accessibilityExtraLarge)
}

#Preview("Right-to-Left") {
    UserProfileView(user: .preview)
        .environment(\.layoutDirection, .rightToLeft)
}
```

---

## Accessibility

### Accessible Components

```swift
struct AccessibleToggleCard: View {
    let title: String
    let description: String
    @Binding var isEnabled: Bool

    var body: some View {
        HStack {
            VStack(alignment: .leading) {
                Text(title).font(.headline)
                Text(description).font(.caption).foregroundStyle(.secondary)
            }
            Spacer()
            Toggle("", isOn: $isEnabled)
                .labelsHidden()
        }
        .padding()
        .cardStyle()
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(title): \(description)")
        .accessibilityValue(isEnabled ? "Enabled" : "Disabled")
        .accessibilityAddTraits(.isButton)
        .accessibilityHint("Double tap to \(isEnabled ? "disable" : "enable")")
    }
}
```

### Accessibility Best Practices

- Always provide `accessibilityLabel` for images and icons
- Use `accessibilityElement(children: .combine)` to group related elements
- Provide `accessibilityHint` for non-obvious interactions
- Support Dynamic Type with `.font(.body)` and relative sizes
- Test with VoiceOver and Accessibility Inspector
- Use semantic colors (`.primary`, `.secondary`) that adapt to settings
- Support Reduce Motion: check `@Environment(\.accessibilityReduceMotion)`

```swift
struct MotionAwareView: View {
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var isVisible = false

    var body: some View {
        Text("Hello")
            .opacity(isVisible ? 1 : 0)
            .offset(y: isVisible ? 0 : (reduceMotion ? 0 : 20))
            .animation(reduceMotion ? nil : .easeOut(duration: 0.3), value: isVisible)
            .onAppear { isVisible = true }
    }
}
```

---

## Platform Adaptations

### Conditional Platform Views

```swift
struct AdaptiveDetailView: View {
    let item: Item

    var body: some View {
        #if os(iOS)
        NavigationStack {
            detailContent
                .navigationBarTitleDisplayMode(.inline)
        }
        #elseif os(macOS)
        detailContent
            .frame(minWidth: 400, minHeight: 300)
        #elseif os(watchOS)
        ScrollView {
            detailContent
                .padding(.horizontal)
        }
        #endif
    }

    private var detailContent: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(item.title).font(.title)
            Text(item.description).font(.body)
        }
        .padding()
    }
}
```

### Adaptive Layout with ViewThatFits

```swift
struct AdaptiveStack: View {
    let items: [Item]

    var body: some View {
        ViewThatFits {
            // Try horizontal first
            HStack(spacing: 16) {
                ForEach(items) { item in ItemCard(item: item) }
            }

            // Fall back to vertical if horizontal doesn't fit
            VStack(spacing: 12) {
                ForEach(items) { item in ItemCard(item: item) }
            }
        }
    }
}
```

### iPad Split View

```swift
struct SplitContentView: View {
    @State private var selectedItem: Item?

    var body: some View {
        NavigationSplitView {
            List(items, selection: $selectedItem) { item in
                NavigationLink(value: item) {
                    Text(item.title)
                }
            }
            .navigationTitle("Items")
        } detail: {
            if let item = selectedItem {
                ItemDetailView(item: item)
            } else {
                ContentUnavailableView("Select an Item", systemImage: "doc.text")
            }
        }
    }
}
```

---

## ObservableObject to @Observable Migration

When migrating from iOS 14-16 patterns to iOS 17+:

### Before (ObservableObject)

```swift
final class SettingsViewModel: ObservableObject {
    @Published var isDarkMode = false
    @Published var fontSize: CGFloat = 14
    @Published var notifications = true
}

struct SettingsView: View {
    @StateObject private var viewModel = SettingsViewModel()
    // ...
}
```

### After (@Observable)

```swift
@Observable
final class SettingsViewModel {
    var isDarkMode = false
    var fontSize: CGFloat = 14
    var notifications = true
}

struct SettingsView: View {
    @State private var viewModel = SettingsViewModel()
    // ...
}
```

**Migration checklist:**
- Replace `ObservableObject` conformance with `@Observable` macro
- Remove `@Published` from all properties (observation is automatic)
- Replace `@StateObject` with `@State` for owned instances
- Replace `@ObservedObject` with direct property (observation is automatic)
- Replace `@EnvironmentObject` with `@Environment(MyType.self)`
- Replace `.environmentObject(obj)` with `.environment(obj)`

---

## Form Validation Model

```swift
struct RegistrationForm {
    var name = ""
    var email = ""
    var dateOfBirth = Date()
    var password = ""
    var confirmPassword = ""
    var receiveNewsletter = false
    var notificationFrequency = NotificationFrequency.daily

    var isValidEmail: Bool { email.contains("@") && email.contains(".") }
    var isValidPassword: Bool { password.count >= 8 }
    var passwordsMatch: Bool { password == confirmPassword }
    var isValid: Bool { !name.isEmpty && isValidEmail && isValidPassword && passwordsMatch }
}

enum NotificationFrequency: String, CaseIterable, Identifiable {
    case daily = "Daily"
    case weekly = "Weekly"
    case monthly = "Monthly"
    case never = "Never"
    var id: String { rawValue }
}
```

---

## Scene Phase Handling

```swift
@main
struct MyApp: App {
    @Environment(\.scenePhase) private var scenePhase
    @StateObject private var dataController = DataController()

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environment(\.managedObjectContext, dataController.container.viewContext)
        }
        .onChange(of: scenePhase) { _, newPhase in
            switch newPhase {
            case .active:
                // Refresh data, resume tasks
                break
            case .inactive:
                // Pause ongoing work
                break
            case .background:
                dataController.save()
            @unknown default:
                break
            }
        }
    }
}
```

---

## Common Anti-Patterns

### State Management

- **Wrong:** `@ObservedObject` for owned state (recreated on re-render)
- **Right:** `@StateObject` (iOS 14-16) or `@State` with `@Observable` (iOS 17+)

- **Wrong:** Creating `@StateObject` inside computed property or closure
- **Right:** Declare as stored `@StateObject` property

- **Wrong:** Storing derived state (filtering, sorting results)
- **Right:** Compute derived values in `body` or computed properties

### Performance

- **Wrong:** `VStack { ForEach(largeArray) { ... } }` in `ScrollView`
- **Right:** `LazyVStack { ForEach(largeArray) { ... } }` in `ScrollView`

- **Wrong:** Complex calculations inside `body`
- **Right:** Move to view model, use `.task`, or cache with `@State`

- **Wrong:** Using `AnyView` for type erasure
- **Right:** Use `@ViewBuilder`, `Group`, or generics

### Navigation

- **Wrong:** Deep NavigationLink nesting without path management
- **Right:** `NavigationStack(path:)` with `Hashable` route enum

- **Wrong:** Mixing NavigationView and NavigationStack
- **Right:** Use `NavigationStack` exclusively (iOS 16+)
