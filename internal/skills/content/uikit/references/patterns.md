# UIKit Patterns Reference

## Contents

- [Coordinator Pattern (Full Implementation)](#coordinator-pattern-full-implementation)
- [Custom Views](#custom-views)
- [Animations](#animations)
- [Core Data Integration](#core-data-integration)
- [Networking Layer](#networking-layer)
- [Testing Patterns](#testing-patterns)
- [Multi-Section Collection Views](#multi-section-collection-views)
- [SceneDelegate Setup](#scenedelegate-setup)

## Coordinator Pattern (Full Implementation)

### Feature Coordinator with Delegate

```swift
import UIKit

protocol UserCoordinatorDelegate: AnyObject {
    func userCoordinatorDidFinish(_ coordinator: UserCoordinator)
}

final class UserCoordinator: Coordinator {
    var childCoordinators: [Coordinator] = []
    var navigationController: UINavigationController

    weak var delegate: UserCoordinatorDelegate?

    init(navigationController: UINavigationController) {
        self.navigationController = navigationController
    }

    func start() {
        let viewModel = UserListViewModel()
        let viewController = UserListViewController(viewModel: viewModel)
        viewController.coordinator = self
        navigationController.pushViewController(viewController, animated: true)
    }

    func showUserDetail(_ user: User) {
        let viewModel = UserDetailViewModel(userId: user.id)
        let viewController = UserDetailViewController(viewModel: viewModel)
        viewController.coordinator = self
        navigationController.pushViewController(viewController, animated: true)
    }

    func showAddUser() {
        let viewModel = AddUserViewModel()
        let viewController = AddUserViewController(viewModel: viewModel)
        viewController.delegate = self

        let navController = UINavigationController(rootViewController: viewController)
        navigationController.present(navController, animated: true)
    }

    func showEditUser(_ user: User) {
        let viewModel = EditUserViewModel(user: user)
        let viewController = EditUserViewController(viewModel: viewModel)
        viewController.delegate = self
        navigationController.pushViewController(viewController, animated: true)
    }
}

extension UserCoordinator: AddUserViewControllerDelegate {
    func addUserDidSave(_ controller: AddUserViewController, user: User) {
        controller.dismiss(animated: true)
    }

    func addUserDidCancel(_ controller: AddUserViewController) {
        controller.dismiss(animated: true)
    }
}
```

### Tab Bar Coordinator

```swift
final class MainTabCoordinator: Coordinator {
    var childCoordinators: [Coordinator] = []
    var navigationController: UINavigationController

    private var tabBarController: UITabBarController!

    init(navigationController: UINavigationController) {
        self.navigationController = navigationController
    }

    func start() {
        tabBarController = UITabBarController()

        let homeNav = UINavigationController()
        let homeCoordinator = HomeCoordinator(navigationController: homeNav)
        homeNav.tabBarItem = UITabBarItem(
            title: "Home",
            image: UIImage(systemName: "house"),
            selectedImage: UIImage(systemName: "house.fill")
        )

        let profileNav = UINavigationController()
        let profileCoordinator = ProfileCoordinator(navigationController: profileNav)
        profileNav.tabBarItem = UITabBarItem(
            title: "Profile",
            image: UIImage(systemName: "person"),
            selectedImage: UIImage(systemName: "person.fill")
        )

        addChild(homeCoordinator)
        addChild(profileCoordinator)

        tabBarController.viewControllers = [homeNav, profileNav]

        homeCoordinator.start()
        profileCoordinator.start()

        navigationController.setNavigationBarHidden(true, animated: false)
        navigationController.viewControllers = [tabBarController]
    }
}
```

## Custom Views

### Reusable Table View Cell with Avatar

```swift
final class UserCell: UITableViewCell {
    static let identifier = "UserCell"

    private let avatarImageView: UIImageView = {
        let imageView = UIImageView()
        imageView.translatesAutoresizingMaskIntoConstraints = false
        imageView.contentMode = .scaleAspectFill
        imageView.clipsToBounds = true
        imageView.backgroundColor = .systemGray5
        imageView.layer.cornerRadius = 24
        return imageView
    }()

    private let nameLabel: UILabel = {
        let label = UILabel()
        label.translatesAutoresizingMaskIntoConstraints = false
        label.font = .preferredFont(forTextStyle: .headline)
        label.textColor = .label
        return label
    }()

    private let emailLabel: UILabel = {
        let label = UILabel()
        label.translatesAutoresizingMaskIntoConstraints = false
        label.font = .preferredFont(forTextStyle: .subheadline)
        label.textColor = .secondaryLabel
        return label
    }()

    private let stackView: UIStackView = {
        let stack = UIStackView()
        stack.translatesAutoresizingMaskIntoConstraints = false
        stack.axis = .vertical
        stack.spacing = 4
        return stack
    }()

    override init(style: UITableViewCell.CellStyle, reuseIdentifier: String?) {
        super.init(style: style, reuseIdentifier: reuseIdentifier)
        setupUI()
    }

    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    private func setupUI() {
        contentView.addSubview(avatarImageView)
        contentView.addSubview(stackView)
        stackView.addArrangedSubview(nameLabel)
        stackView.addArrangedSubview(emailLabel)
        accessoryType = .disclosureIndicator

        NSLayoutConstraint.activate([
            avatarImageView.leadingAnchor.constraint(equalTo: contentView.leadingAnchor, constant: 16),
            avatarImageView.centerYAnchor.constraint(equalTo: contentView.centerYAnchor),
            avatarImageView.widthAnchor.constraint(equalToConstant: 48),
            avatarImageView.heightAnchor.constraint(equalToConstant: 48),
            avatarImageView.topAnchor.constraint(
                greaterThanOrEqualTo: contentView.topAnchor, constant: 12
            ),
            avatarImageView.bottomAnchor.constraint(
                lessThanOrEqualTo: contentView.bottomAnchor, constant: -12
            ),
            stackView.leadingAnchor.constraint(
                equalTo: avatarImageView.trailingAnchor, constant: 12
            ),
            stackView.trailingAnchor.constraint(
                equalTo: contentView.trailingAnchor, constant: -16
            ),
            stackView.centerYAnchor.constraint(equalTo: contentView.centerYAnchor)
        ])
    }

    func configure(with user: User) {
        nameLabel.text = user.name
        emailLabel.text = user.email
        if let url = URL(string: user.avatarURL) {
            ImageLoader.shared.load(url: url) { [weak self] image in
                self?.avatarImageView.image = image
            }
        }
    }

    override func prepareForReuse() {
        super.prepareForReuse()
        avatarImageView.image = nil
        nameLabel.text = nil
        emailLabel.text = nil
    }
}
```

### Reusable Button Component

```swift
final class PrimaryButton: UIButton {
    private var originalBackgroundColor: UIColor?

    override var isEnabled: Bool {
        didSet { alpha = isEnabled ? 1.0 : 0.5 }
    }

    override var isHighlighted: Bool {
        didSet {
            UIView.animate(withDuration: 0.1) {
                self.transform = self.isHighlighted
                    ? CGAffineTransform(scaleX: 0.95, y: 0.95) : .identity
            }
        }
    }

    override init(frame: CGRect) {
        super.init(frame: frame)
        setupUI()
    }

    required init?(coder: NSCoder) {
        super.init(coder: coder)
        setupUI()
    }

    convenience init(title: String) {
        self.init(frame: .zero)
        setTitle(title, for: .normal)
    }

    private func setupUI() {
        backgroundColor = .systemBlue
        setTitleColor(.white, for: .normal)
        titleLabel?.font = .preferredFont(forTextStyle: .headline)
        layer.cornerRadius = 12
        contentEdgeInsets = UIEdgeInsets(top: 14, left: 24, bottom: 14, right: 24)
        originalBackgroundColor = backgroundColor
    }

    // Loading state
    private var activityIndicator: UIActivityIndicatorView?
    private var savedTitle: String?

    func showLoading() {
        savedTitle = title(for: .normal)
        setTitle(nil, for: .normal)
        isEnabled = false

        let indicator = UIActivityIndicatorView(style: .medium)
        indicator.color = .white
        indicator.translatesAutoresizingMaskIntoConstraints = false
        addSubview(indicator)
        NSLayoutConstraint.activate([
            indicator.centerXAnchor.constraint(equalTo: centerXAnchor),
            indicator.centerYAnchor.constraint(equalTo: centerYAnchor)
        ])
        indicator.startAnimating()
        activityIndicator = indicator
    }

    func hideLoading() {
        activityIndicator?.stopAnimating()
        activityIndicator?.removeFromSuperview()
        activityIndicator = nil
        setTitle(savedTitle, for: .normal)
        isEnabled = true
    }
}
```

### Empty State View

```swift
final class EmptyStateView: UIView {
    private let imageView: UIImageView = {
        let iv = UIImageView()
        iv.translatesAutoresizingMaskIntoConstraints = false
        iv.contentMode = .scaleAspectFit
        iv.tintColor = .secondaryLabel
        return iv
    }()

    private let titleLabel: UILabel = {
        let label = UILabel()
        label.translatesAutoresizingMaskIntoConstraints = false
        label.font = .preferredFont(forTextStyle: .headline)
        label.textAlignment = .center
        return label
    }()

    private let messageLabel: UILabel = {
        let label = UILabel()
        label.translatesAutoresizingMaskIntoConstraints = false
        label.font = .preferredFont(forTextStyle: .body)
        label.textColor = .secondaryLabel
        label.textAlignment = .center
        label.numberOfLines = 0
        return label
    }()

    override init(frame: CGRect) {
        super.init(frame: frame)
        setupUI()
    }

    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    private func setupUI() {
        let stack = UIStackView(arrangedSubviews: [imageView, titleLabel, messageLabel])
        stack.translatesAutoresizingMaskIntoConstraints = false
        stack.axis = .vertical
        stack.spacing = 12
        stack.alignment = .center
        addSubview(stack)

        NSLayoutConstraint.activate([
            imageView.widthAnchor.constraint(equalToConstant: 64),
            imageView.heightAnchor.constraint(equalToConstant: 64),
            stack.centerXAnchor.constraint(equalTo: centerXAnchor),
            stack.centerYAnchor.constraint(equalTo: centerYAnchor),
            stack.leadingAnchor.constraint(greaterThanOrEqualTo: leadingAnchor, constant: 32),
            stack.trailingAnchor.constraint(lessThanOrEqualTo: trailingAnchor, constant: -32)
        ])
    }

    func configure(image: UIImage?, title: String, message: String) {
        imageView.image = image
        titleLabel.text = title
        messageLabel.text = message
    }
}
```

## Animations

### UIView Spring Animations

```swift
extension UIView {
    func fadeIn(duration: TimeInterval = 0.3) {
        alpha = 0
        UIView.animate(withDuration: duration) { self.alpha = 1 }
    }

    func fadeOut(duration: TimeInterval = 0.3, completion: (() -> Void)? = nil) {
        UIView.animate(withDuration: duration, animations: { self.alpha = 0 }) { _ in
            completion?()
        }
    }

    func springBounce(scale: CGFloat = 0.9) {
        UIView.animate(
            withDuration: 0.15,
            animations: { self.transform = CGAffineTransform(scaleX: scale, y: scale) }
        ) { _ in
            UIView.animate(
                withDuration: 0.3,
                delay: 0,
                usingSpringWithDamping: 0.5,
                initialSpringVelocity: 3,
                options: .allowUserInteraction
            ) { self.transform = .identity }
        }
    }
}
```

### Animated Transitions

```swift
final class SlideTransition: NSObject, UIViewControllerAnimatedTransitioning {
    private let duration: TimeInterval
    private let isPresenting: Bool

    init(duration: TimeInterval = 0.3, isPresenting: Bool) {
        self.duration = duration
        self.isPresenting = isPresenting
    }

    func transitionDuration(using context: UIViewControllerContextTransitioning?) -> TimeInterval {
        duration
    }

    func animateTransition(using context: UIViewControllerContextTransitioning) {
        guard let toView = context.view(forKey: .to),
              let fromView = context.view(forKey: .from) else {
            context.completeTransition(false)
            return
        }

        let containerView = context.containerView
        let width = containerView.bounds.width

        if isPresenting {
            toView.frame.origin.x = width
            containerView.addSubview(toView)
        }

        UIView.animate(
            withDuration: duration,
            delay: 0,
            options: .curveEaseInOut,
            animations: {
                if self.isPresenting {
                    toView.frame.origin.x = 0
                } else {
                    fromView.frame.origin.x = width
                }
            }
        ) { finished in
            context.completeTransition(finished && !context.transitionWasCancelled)
        }
    }
}
```

### Property Animator

```swift
final class CardAnimator {
    private var animator: UIViewPropertyAnimator?

    func expand(view: UIView) {
        animator?.stopAnimation(true)
        animator = UIViewPropertyAnimator(duration: 0.5, dampingRatio: 0.7) {
            view.transform = CGAffineTransform(scaleX: 1.1, y: 1.1)
            view.layer.shadowOpacity = 0.3
            view.layer.shadowRadius = 10
        }
        animator?.startAnimation()
    }

    func collapse(view: UIView) {
        animator?.stopAnimation(true)
        animator = UIViewPropertyAnimator(duration: 0.3, curve: .easeOut) {
            view.transform = .identity
            view.layer.shadowOpacity = 0
        }
        animator?.startAnimation()
    }
}
```

## Core Data Integration

### Core Data Manager

```swift
import CoreData

final class CoreDataManager {
    static let shared = CoreDataManager()

    private init() {}

    lazy var persistentContainer: NSPersistentContainer = {
        let container = NSPersistentContainer(name: "MyApp")
        container.loadPersistentStores { _, error in
            if let error = error { fatalError("Core Data load failed: \(error)") }
        }
        container.viewContext.automaticallyMergesChangesFromParent = true
        return container
    }()

    var viewContext: NSManagedObjectContext { persistentContainer.viewContext }

    func newBackgroundContext() -> NSManagedObjectContext {
        persistentContainer.newBackgroundContext()
    }

    func saveContext() {
        let context = viewContext
        guard context.hasChanges else { return }
        do { try context.save() }
        catch { print("Core Data save failed: \(error)") }
    }

    func performBackground(_ block: @escaping (NSManagedObjectContext) -> Void) {
        persistentContainer.performBackgroundTask(block)
    }
}
```

### Fetched Results Controller

```swift
import CoreData

final class UserListDataProvider: NSObject, NSFetchedResultsControllerDelegate {
    private let fetchedResultsController: NSFetchedResultsController<UserEntity>
    var onUpdate: (() -> Void)?

    init(context: NSManagedObjectContext = CoreDataManager.shared.viewContext) {
        let request: NSFetchRequest<UserEntity> = UserEntity.fetchRequest()
        request.sortDescriptors = [NSSortDescriptor(key: "name", ascending: true)]
        request.fetchBatchSize = 20

        fetchedResultsController = NSFetchedResultsController(
            fetchRequest: request,
            managedObjectContext: context,
            sectionNameKeyPath: nil,
            cacheName: nil
        )

        super.init()
        fetchedResultsController.delegate = self
        try? fetchedResultsController.performFetch()
    }

    var numberOfUsers: Int {
        fetchedResultsController.fetchedObjects?.count ?? 0
    }

    func user(at indexPath: IndexPath) -> UserEntity {
        fetchedResultsController.object(at: indexPath)
    }

    func controllerDidChangeContent(
        _ controller: NSFetchedResultsController<NSFetchRequestResult>
    ) {
        onUpdate?()
    }
}
```

## Networking Layer

### Endpoint Definition

```swift
enum HTTPMethod: String {
    case get = "GET"
    case post = "POST"
    case put = "PUT"
    case delete = "DELETE"
}

struct Endpoint {
    let path: String
    let method: HTTPMethod
    let queryItems: [URLQueryItem]?

    init(path: String, method: HTTPMethod = .get, queryItems: [URLQueryItem]? = nil) {
        self.path = path
        self.method = method
        self.queryItems = queryItems
    }
}

extension Endpoint {
    static func users() -> Endpoint { Endpoint(path: "/users") }
    static func user(id: String) -> Endpoint { Endpoint(path: "/users/\(id)") }
    static func createUser() -> Endpoint { Endpoint(path: "/users", method: .post) }
    static func deleteUser(id: String) -> Endpoint { Endpoint(path: "/users/\(id)", method: .delete) }
}
```

### API Client

```swift
enum APIError: Error {
    case invalidURL
    case invalidResponse
    case httpError(statusCode: Int)
    case decodingError(Error)
}

protocol APIClientProtocol {
    func fetch<T: Decodable>(_ endpoint: Endpoint) async throws -> T
    func send<T: Decodable, U: Encodable>(_ endpoint: Endpoint, body: U) async throws -> T
}

final class APIClient: APIClientProtocol {
    private let session: URLSession
    private let decoder: JSONDecoder
    private let encoder: JSONEncoder
    private let baseURL: URL

    init(baseURL: URL, session: URLSession = .shared) {
        self.baseURL = baseURL
        self.session = session
        self.decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        decoder.dateDecodingStrategy = .iso8601
        self.encoder = JSONEncoder()
        encoder.keyEncodingStrategy = .convertToSnakeCase
    }

    func fetch<T: Decodable>(_ endpoint: Endpoint) async throws -> T {
        let request = try buildRequest(for: endpoint)
        return try await execute(request)
    }

    func send<T: Decodable, U: Encodable>(_ endpoint: Endpoint, body: U) async throws -> T {
        var request = try buildRequest(for: endpoint)
        request.httpBody = try encoder.encode(body)
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        return try await execute(request)
    }

    private func buildRequest(for endpoint: Endpoint) throws -> URLRequest {
        guard let url = URL(string: endpoint.path, relativeTo: baseURL) else {
            throw APIError.invalidURL
        }
        var request = URLRequest(url: url)
        request.httpMethod = endpoint.method.rawValue
        request.timeoutInterval = 30
        if let token = AuthManager.shared.accessToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        return request
    }

    private func execute<T: Decodable>(_ request: URLRequest) async throws -> T {
        let (data, response) = try await session.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }
        guard (200...299).contains(httpResponse.statusCode) else {
            throw APIError.httpError(statusCode: httpResponse.statusCode)
        }
        do { return try decoder.decode(T.self, from: data) }
        catch { throw APIError.decodingError(error) }
    }
}
```

### Service Layer

```swift
protocol UserServiceProtocol {
    func fetchUsers() async throws -> [User]
    func fetchUser(id: String) async throws -> User
    func deleteUser(id: String) async throws
    func searchUsers(query: String) async throws -> [User]
}

final class UserService: UserServiceProtocol {
    private let apiClient: APIClientProtocol

    init(apiClient: APIClientProtocol = APIClient(baseURL: AppConfig.apiBaseURL)) {
        self.apiClient = apiClient
    }

    func fetchUsers() async throws -> [User] {
        try await apiClient.fetch(.users())
    }

    func fetchUser(id: String) async throws -> User {
        try await apiClient.fetch(.user(id: id))
    }

    func deleteUser(id: String) async throws {
        let _: EmptyResponse = try await apiClient.fetch(.deleteUser(id: id))
    }

    func searchUsers(query: String) async throws -> [User] {
        let endpoint = Endpoint(
            path: "/users/search",
            queryItems: [URLQueryItem(name: "q", value: query)]
        )
        return try await apiClient.fetch(endpoint)
    }
}
```

## Testing Patterns

### Mock Service

```swift
final class MockUserService: UserServiceProtocol {
    var stubbedUsers: [User] = []
    var stubbedUser: User?
    var stubbedError: Error?

    var fetchUsersCalled = false
    var deletedUserID: String?

    func fetchUsers() async throws -> [User] {
        fetchUsersCalled = true
        if let error = stubbedError { throw error }
        return stubbedUsers
    }

    func fetchUser(id: String) async throws -> User {
        if let error = stubbedError { throw error }
        guard let user = stubbedUser else { throw APIError.invalidResponse }
        return user
    }

    func deleteUser(id: String) async throws {
        deletedUserID = id
        if let error = stubbedError { throw error }
    }

    func searchUsers(query: String) async throws -> [User] {
        if let error = stubbedError { throw error }
        return stubbedUsers.filter { $0.name.contains(query) }
    }
}
```

### View Model Tests

```swift
import XCTest
import Combine
@testable import MyApp

final class UserListViewModelTests: XCTestCase {
    var sut: UserListViewModel!
    var mockService: MockUserService!
    var cancellables: Set<AnyCancellable>!

    override func setUp() {
        super.setUp()
        mockService = MockUserService()
        sut = UserListViewModel(userService: mockService)
        cancellables = []
    }

    override func tearDown() {
        sut = nil
        mockService = nil
        cancellables = nil
        super.tearDown()
    }

    func test_loadUsers_setsUsers() {
        let expectation = expectation(description: "users loaded")
        mockService.stubbedUsers = [User.stub(name: "Alice"), User.stub(name: "Bob")]

        sut.$users
            .dropFirst()
            .sink { users in
                XCTAssertEqual(users.count, 2)
                XCTAssertEqual(users.first?.name, "Alice")
                expectation.fulfill()
            }
            .store(in: &cancellables)

        sut.loadUsers()
        waitForExpectations(timeout: 2)
    }

    func test_loadUsers_setsErrorOnFailure() {
        let expectation = expectation(description: "error received")
        mockService.stubbedError = APIError.invalidResponse

        sut.$error
            .compactMap { $0 }
            .sink { error in
                XCTAssertNotNil(error)
                expectation.fulfill()
            }
            .store(in: &cancellables)

        sut.loadUsers()
        waitForExpectations(timeout: 2)
    }

    func test_deleteUser_removesFromList() {
        let expectation = expectation(description: "user removed")
        let user = User.stub(name: "Alice")
        sut = UserListViewModel(userService: mockService)

        // Pre-populate
        mockService.stubbedUsers = [user]
        sut.loadUsers()

        DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) {
            self.sut.deleteUser(at: 0)

            DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) {
                XCTAssertTrue(self.sut.users.isEmpty)
                XCTAssertEqual(self.mockService.deletedUserID, user.id)
                expectation.fulfill()
            }
        }

        waitForExpectations(timeout: 3)
    }
}
```

### Coordinator Tests

```swift
import XCTest
@testable import MyApp

final class UserCoordinatorTests: XCTestCase {
    var sut: UserCoordinator!
    var navigationController: UINavigationController!

    override func setUp() {
        super.setUp()
        navigationController = UINavigationController()
        sut = UserCoordinator(navigationController: navigationController)
    }

    override func tearDown() {
        sut = nil
        navigationController = nil
        super.tearDown()
    }

    func test_start_pushesUserListViewController() {
        sut.start()
        XCTAssertTrue(navigationController.topViewController is UserListViewController)
    }

    func test_showUserDetail_pushesDetailViewController() {
        sut.start()
        let user = User.stub()
        sut.showUserDetail(user)
        XCTAssertTrue(navigationController.topViewController is UserDetailViewController)
    }

    func test_showAddUser_presentsModalNavController() {
        sut.start()
        let window = UIWindow()
        window.rootViewController = navigationController
        window.makeKeyAndVisible()

        sut.showAddUser()

        XCTAssertNotNil(navigationController.presentedViewController)
        XCTAssertTrue(
            navigationController.presentedViewController is UINavigationController
        )
    }
}
```

### Test Stubs

```swift
extension User {
    static func stub(
        id: String = UUID().uuidString,
        name: String = "Test User",
        email: String = "test@example.com",
        avatarURL: String = "https://example.com/avatar.png"
    ) -> User {
        User(id: id, name: name, email: email, avatarURL: avatarURL)
    }
}
```

## Multi-Section Collection Views

### Compositional Layout with Multiple Sections

```swift
enum HomeSection: Int, CaseIterable {
    case featured
    case categories
    case recent

    var title: String {
        switch self {
        case .featured: return "Featured"
        case .categories: return "Categories"
        case .recent: return "Recent"
        }
    }
}

private func createLayout() -> UICollectionViewLayout {
    UICollectionViewCompositionalLayout { [weak self] sectionIndex, _ in
        guard let section = HomeSection(rawValue: sectionIndex) else { return nil }
        switch section {
        case .featured: return self?.createFeaturedSection()
        case .categories: return self?.createCategoriesSection()
        case .recent: return self?.createRecentSection()
        }
    }
}

private func createFeaturedSection() -> NSCollectionLayoutSection {
    let itemSize = NSCollectionLayoutSize(
        widthDimension: .fractionalWidth(1.0), heightDimension: .fractionalHeight(1.0)
    )
    let item = NSCollectionLayoutItem(layoutSize: itemSize)
    let groupSize = NSCollectionLayoutSize(
        widthDimension: .fractionalWidth(0.9), heightDimension: .absolute(250)
    )
    let group = NSCollectionLayoutGroup.horizontal(layoutSize: groupSize, subitems: [item])
    let section = NSCollectionLayoutSection(group: group)
    section.orthogonalScrollingBehavior = .groupPagingCentered
    section.contentInsets = NSDirectionalEdgeInsets(top: 16, leading: 0, bottom: 16, trailing: 0)
    section.interGroupSpacing = 16
    section.boundarySupplementaryItems = [createSectionHeader()]
    return section
}

private func createCategoriesSection() -> NSCollectionLayoutSection {
    let itemSize = NSCollectionLayoutSize(
        widthDimension: .fractionalWidth(1.0), heightDimension: .fractionalHeight(1.0)
    )
    let item = NSCollectionLayoutItem(layoutSize: itemSize)
    let groupSize = NSCollectionLayoutSize(
        widthDimension: .absolute(100), heightDimension: .absolute(100)
    )
    let group = NSCollectionLayoutGroup.horizontal(layoutSize: groupSize, subitems: [item])
    let section = NSCollectionLayoutSection(group: group)
    section.orthogonalScrollingBehavior = .continuous
    section.contentInsets = NSDirectionalEdgeInsets(top: 8, leading: 16, bottom: 16, trailing: 16)
    section.interGroupSpacing = 12
    section.boundarySupplementaryItems = [createSectionHeader()]
    return section
}

private func createRecentSection() -> NSCollectionLayoutSection {
    let itemSize = NSCollectionLayoutSize(
        widthDimension: .fractionalWidth(1.0), heightDimension: .estimated(100)
    )
    let item = NSCollectionLayoutItem(layoutSize: itemSize)
    let groupSize = NSCollectionLayoutSize(
        widthDimension: .fractionalWidth(1.0), heightDimension: .estimated(100)
    )
    let group = NSCollectionLayoutGroup.vertical(layoutSize: groupSize, subitems: [item])
    let section = NSCollectionLayoutSection(group: group)
    section.contentInsets = NSDirectionalEdgeInsets(top: 8, leading: 16, bottom: 16, trailing: 16)
    section.interGroupSpacing = 12
    section.boundarySupplementaryItems = [createSectionHeader()]
    return section
}

private func createSectionHeader() -> NSCollectionLayoutBoundarySupplementaryItem {
    let headerSize = NSCollectionLayoutSize(
        widthDimension: .fractionalWidth(1.0), heightDimension: .estimated(44)
    )
    return NSCollectionLayoutBoundarySupplementaryItem(
        layoutSize: headerSize,
        elementKind: UICollectionView.elementKindSectionHeader,
        alignment: .top
    )
}
```

## SceneDelegate Setup

```swift
import UIKit

class SceneDelegate: UIResponder, UIWindowSceneDelegate {
    var window: UIWindow?
    var appCoordinator: AppCoordinator?

    func scene(
        _ scene: UIScene,
        willConnectTo session: UISceneSession,
        options connectionOptions: UIScene.ConnectionOptions
    ) {
        guard let windowScene = (scene as? UIWindowScene) else { return }

        let window = UIWindow(windowScene: windowScene)
        self.window = window

        let navigationController = UINavigationController()
        appCoordinator = AppCoordinator(navigationController: navigationController)

        window.rootViewController = navigationController
        window.makeKeyAndVisible()

        appCoordinator?.start()
    }

    func sceneDidEnterBackground(_ scene: UIScene) {
        CoreDataManager.shared.saveContext()
    }
}
```
