# Symfony Patterns Reference

## Contents

- [Doctrine Advanced Patterns](#doctrine-advanced-patterns)
- [DTOs with Validation](#dtos-with-validation)
- [Event System](#event-system)
- [Messenger (Async Processing)](#messenger-async-processing)
- [API Platform](#api-platform)
- [Security Configuration](#security-configuration)
- [Testing](#testing)
- [Deployment](#deployment)
- [Performance](#performance)

## Doctrine Advanced Patterns

### Entity with Relationships

```php
<?php

declare(strict_types=1);

namespace App\Entity;

use App\Repository\UserRepository;
use Doctrine\Common\Collections\ArrayCollection;
use Doctrine\Common\Collections\Collection;
use Doctrine\DBAL\Types\Types;
use Doctrine\ORM\Mapping as ORM;
use Symfony\Bridge\Doctrine\Validator\Constraints\UniqueEntity;
use Symfony\Component\Security\Core\User\PasswordAuthenticatedUserInterface;
use Symfony\Component\Security\Core\User\UserInterface;
use Symfony\Component\Validator\Constraints as Assert;

#[ORM\Entity(repositoryClass: UserRepository::class)]
#[ORM\Table(name: 'users')]
#[ORM\Index(columns: ['role', 'is_active'])]
#[ORM\HasLifecycleCallbacks]
#[UniqueEntity(fields: ['email'], message: 'This email is already registered.')]
class User implements UserInterface, PasswordAuthenticatedUserInterface
{
    #[ORM\Id]
    #[ORM\GeneratedValue]
    #[ORM\Column]
    private ?int $id = null;

    #[ORM\Column(length: 255)]
    #[Assert\NotBlank]
    #[Assert\Length(max: 255)]
    private string $name;

    #[ORM\Column(length: 180, unique: true)]
    #[Assert\NotBlank]
    #[Assert\Email]
    private string $email;

    #[ORM\Column]
    private string $password;

    #[ORM\Column(length: 50)]
    #[Assert\Choice(choices: ['admin', 'user', 'guest'])]
    private string $role = 'user';

    #[ORM\Column]
    private bool $isActive = true;

    #[ORM\Column(type: Types::DATETIME_IMMUTABLE)]
    private \DateTimeImmutable $createdAt;

    #[ORM\Column(type: Types::DATETIME_IMMUTABLE, nullable: true)]
    private ?\DateTimeImmutable $updatedAt = null;

    #[ORM\ManyToOne(targetEntity: Organization::class, inversedBy: 'users')]
    #[ORM\JoinColumn(nullable: true, onDelete: 'SET NULL')]
    private ?Organization $organization = null;

    #[ORM\OneToMany(mappedBy: 'author', targetEntity: Post::class, orphanRemoval: true)]
    private Collection $posts;

    public function __construct()
    {
        $this->posts = new ArrayCollection();
        $this->createdAt = new \DateTimeImmutable();
    }

    #[ORM\PreUpdate]
    public function setUpdatedAtValue(): void
    {
        $this->updatedAt = new \DateTimeImmutable();
    }

    public function getId(): ?int
    {
        return $this->id;
    }

    public function getRoles(): array
    {
        return ['ROLE_' . strtoupper($this->role)];
    }

    public function getUserIdentifier(): string
    {
        return $this->email;
    }

    public function eraseCredentials(): void
    {
        // Clear temporary sensitive data
    }

    // ... getters and fluent setters returning static
}
```

### Repository with Pagination

```php
<?php

declare(strict_types=1);

namespace App\Repository;

use App\Entity\User;
use Doctrine\Bundle\DoctrineBundle\Repository\ServiceEntityRepository;
use Doctrine\ORM\QueryBuilder;
use Doctrine\Persistence\ManagerRegistry;

/** @extends ServiceEntityRepository<User> */
class UserRepository extends ServiceEntityRepository
{
    public function __construct(ManagerRegistry $registry)
    {
        parent::__construct($registry, User::class);
    }

    public function save(User $user, bool $flush = true): void
    {
        $this->getEntityManager()->persist($user);
        if ($flush) {
            $this->getEntityManager()->flush();
        }
    }

    public function remove(User $user, bool $flush = true): void
    {
        $this->getEntityManager()->remove($user);
        if ($flush) {
            $this->getEntityManager()->flush();
        }
    }

    /** @return User[] */
    public function findActiveUsers(): array
    {
        return $this->createQueryBuilder('u')
            ->andWhere('u.isActive = :active')
            ->setParameter('active', true)
            ->orderBy('u.createdAt', 'DESC')
            ->getQuery()
            ->getResult();
    }

    public function findByRole(string $role): array
    {
        return $this->createQueryBuilder('u')
            ->andWhere('u.role = :role')
            ->andWhere('u.isActive = :active')
            ->setParameter('role', $role)
            ->setParameter('active', true)
            ->getQuery()
            ->getResult();
    }

    public function createPaginatedQueryBuilder(): QueryBuilder
    {
        return $this->createQueryBuilder('u')
            ->leftJoin('u.organization', 'o')
            ->addSelect('o')
            ->orderBy('u.createdAt', 'DESC');
    }
}
```

### Doctrine Query Best Practices

- Always use `setParameter()` for query values (prevents SQL injection)
- Use `leftJoin` + `addSelect` to avoid N+1 queries on relations
- Use `getOneOrNullResult()` for single-entity lookups
- Use `QueryBuilder` for dynamic queries; use DQL for static ones
- Avoid `findAll()` on large tables; always paginate
- Use `iterate()` or `toIterable()` for batch processing large datasets
- Index columns that appear in `WHERE`, `ORDER BY`, or `JOIN`

### Batch Processing

```php
public function batchUpdate(int $batchSize = 100): void
{
    $query = $this->createQueryBuilder('u')
        ->andWhere('u.isActive = true')
        ->getQuery();

    $count = 0;
    foreach ($query->toIterable() as $user) {
        $user->setUpdatedAtValue();
        $count++;

        if ($count % $batchSize === 0) {
            $this->getEntityManager()->flush();
            $this->getEntityManager()->clear();
        }
    }

    $this->getEntityManager()->flush();
    $this->getEntityManager()->clear();
}
```

## DTOs with Validation

```php
<?php

declare(strict_types=1);

namespace App\Dto;

use Symfony\Component\Validator\Constraints as Assert;

final readonly class CreateUserDto
{
    public function __construct(
        #[Assert\NotBlank]
        #[Assert\Length(max: 255)]
        public string $name,

        #[Assert\NotBlank]
        #[Assert\Email]
        public string $email,

        #[Assert\NotBlank]
        #[Assert\Length(min: 8)]
        #[Assert\Regex(
            pattern: '/^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[@$!%*?&]).{8,}$/',
            message: 'Password must contain uppercase, lowercase, number, and special character'
        )]
        public string $password,

        #[Assert\NotBlank]
        #[Assert\Choice(choices: ['admin', 'user', 'guest'])]
        public string $role = 'user',

        #[Assert\Positive]
        public ?int $organizationId = null,
    ) {}
}
```

### DTO Guidelines

- Use `final readonly class` for immutability
- Use constructor property promotion
- Apply all validation constraints on DTO properties
- Keep DTOs in `src/Dto/` directory
- One DTO per operation (CreateUserDto, UpdateUserDto, etc.)
- Use `#[MapRequestPayload]` in controllers for automatic deserialization + validation

## Event System

### Custom Event

```php
<?php

declare(strict_types=1);

namespace App\Event;

use App\Entity\User;
use Symfony\Contracts\EventDispatcher\Event;

class UserCreatedEvent extends Event
{
    public function __construct(
        public readonly User $user,
    ) {}
}
```

### Event Subscriber

```php
<?php

declare(strict_types=1);

namespace App\EventSubscriber;

use App\Event\UserCreatedEvent;
use App\Message\SendWelcomeEmail;
use Psr\Log\LoggerInterface;
use Symfony\Component\EventDispatcher\EventSubscriberInterface;
use Symfony\Component\Messenger\MessageBusInterface;

class UserEventSubscriber implements EventSubscriberInterface
{
    public function __construct(
        private readonly MessageBusInterface $messageBus,
        private readonly LoggerInterface $logger,
    ) {}

    public static function getSubscribedEvents(): array
    {
        return [
            UserCreatedEvent::class => 'onUserCreated',
        ];
    }

    public function onUserCreated(UserCreatedEvent $event): void
    {
        $this->logger->info('User created', ['user_id' => $event->user->getId()]);
        $this->messageBus->dispatch(new SendWelcomeEmail($event->user->getId()));
    }
}
```

### Event Best Practices

- Events carry data; subscribers act on it
- Keep subscribers focused on a single side effect
- Use Messenger for heavy work triggered by events (email, external APIs)
- Prefer `EventSubscriberInterface` over listener closures for testability
- Dispatch events from services, not controllers
- Name events after what happened (`UserCreatedEvent`, `OrderPlacedEvent`)

## Messenger (Async Processing)

### Message and Handler

```php
<?php
// src/Message/SendWelcomeEmail.php

declare(strict_types=1);

namespace App\Message;

class SendWelcomeEmail
{
    public function __construct(
        public readonly int $userId,
    ) {}
}
```

```php
<?php
// src/MessageHandler/SendWelcomeEmailHandler.php

declare(strict_types=1);

namespace App\MessageHandler;

use App\Message\SendWelcomeEmail;
use App\Repository\UserRepository;
use Symfony\Component\Mailer\MailerInterface;
use Symfony\Component\Messenger\Attribute\AsMessageHandler;
use Symfony\Component\Mime\Email;

#[AsMessageHandler]
class SendWelcomeEmailHandler
{
    public function __construct(
        private readonly UserRepository $userRepository,
        private readonly MailerInterface $mailer,
    ) {}

    public function __invoke(SendWelcomeEmail $message): void
    {
        $user = $this->userRepository->find($message->userId);
        if ($user === null) {
            return;
        }

        $email = (new Email())
            ->from('noreply@example.com')
            ->to($user->getEmail())
            ->subject('Welcome!')
            ->html('<p>Welcome to our platform!</p>');

        $this->mailer->send($email);
    }
}
```

### Messenger Configuration

```yaml
# config/packages/messenger.yaml
framework:
    messenger:
        failure_transport: failed

        transports:
            async:
                dsn: '%env(MESSENGER_TRANSPORT_DSN)%'
                retry_strategy:
                    max_retries: 3
                    delay: 1000
                    multiplier: 2
            failed:
                dsn: 'doctrine://default?queue_name=failed'

        routing:
            App\Message\SendWelcomeEmail: async
```

### Messenger Guidelines

- Messages are plain PHP objects (no logic, just data)
- Handlers use `__invoke()` method and `#[AsMessageHandler]` attribute
- Pass entity IDs in messages, not full entities (entities may be stale)
- Always handle null entity lookups gracefully in handlers
- Configure retry strategies for transient failures
- Use the `failed` transport to inspect and retry dead-letter messages
- Run workers: `php bin/console messenger:consume async`

## API Platform

### Resource Configuration

```php
<?php

declare(strict_types=1);

namespace App\Entity;

use ApiPlatform\Metadata\ApiResource;
use ApiPlatform\Metadata\Get;
use ApiPlatform\Metadata\GetCollection;
use ApiPlatform\Metadata\Post;
use ApiPlatform\Metadata\Put;
use ApiPlatform\Metadata\Delete;

#[ApiResource(
    operations: [
        new GetCollection(),
        new Get(),
        new Post(security: "is_granted('ROLE_ADMIN')"),
        new Put(security: "is_granted('ROLE_ADMIN') or object.getAuthor() == user"),
        new Delete(security: "is_granted('ROLE_ADMIN')"),
    ],
    normalizationContext: ['groups' => ['read']],
    denormalizationContext: ['groups' => ['write']],
    paginationItemsPerPage: 15,
)]
class Article
{
    // Entity with serialization groups on properties
}
```

### API Platform Guidelines

- Use PHP 8 attributes for all API Platform configuration
- Define security per operation (`security` parameter)
- Use serialization groups to control input/output
- Use custom state providers/processors for complex logic
- Use filters for collection filtering (`#[ApiFilter]`)
- Enable OpenAPI documentation (automatic)

## Security Configuration

### Full Security YAML

```yaml
# config/packages/security.yaml
security:
    password_hashers:
        Symfony\Component\Security\Core\User\PasswordAuthenticatedUserInterface: 'auto'

    providers:
        app_user_provider:
            entity:
                class: App\Entity\User
                property: email

    firewalls:
        dev:
            pattern: ^/(_(profiler|wdt)|css|images|js)/
            security: false

        login:
            pattern: ^/api/login
            stateless: true
            json_login:
                check_path: /api/login
                success_handler: lexik_jwt_authentication.handler.authentication_success
                failure_handler: lexik_jwt_authentication.handler.authentication_failure

        api:
            pattern: ^/api
            stateless: true
            jwt: ~

    access_control:
        - { path: ^/api/login, roles: PUBLIC_ACCESS }
        - { path: ^/api/docs, roles: PUBLIC_ACCESS }
        - { path: ^/api, roles: IS_AUTHENTICATED_FULLY }
```

### Security Checklist

- Hash passwords with `auto` algorithm (bcrypt or argon2id)
- API firewalls must be `stateless: true`
- Use JWT (`lexik/jwt-authentication-bundle`) for API authentication
- Protect all `/api` routes except login and documentation
- Use Voters for object-level permissions
- Use `#[IsGranted]` on controller actions for role-based access
- Never expose password hashes in API responses (use serialization groups)
- Rate-limit login endpoints to prevent brute force

## Testing

### Functional Test (API)

```php
<?php

declare(strict_types=1);

namespace App\Tests\Controller;

use App\Entity\User;
use Doctrine\ORM\EntityManagerInterface;
use Symfony\Bundle\FrameworkBundle\KernelBrowser;
use Symfony\Bundle\FrameworkBundle\Test\WebTestCase;

class UserControllerTest extends WebTestCase
{
    private KernelBrowser $client;
    private EntityManagerInterface $entityManager;

    protected function setUp(): void
    {
        $this->client = static::createClient();
        $this->entityManager = static::getContainer()->get(EntityManagerInterface::class);
    }

    public function testListUsersRequiresAuth(): void
    {
        $this->client->request('GET', '/api/v1/users');
        $this->assertResponseStatusCodeSame(401);
    }

    public function testCreateUserWithValidData(): void
    {
        $this->client->request(
            'POST',
            '/api/v1/users',
            [],
            [],
            [
                'HTTP_AUTHORIZATION' => 'Bearer ' . $this->getAdminToken(),
                'CONTENT_TYPE' => 'application/json',
            ],
            json_encode([
                'name' => 'John Doe',
                'email' => 'john@example.com',
                'password' => 'Password123!',
                'role' => 'user',
            ])
        );

        $this->assertResponseStatusCodeSame(201);
    }

    public function testCreateUserRejectsInvalidData(): void
    {
        $this->client->request(
            'POST',
            '/api/v1/users',
            [],
            [],
            [
                'HTTP_AUTHORIZATION' => 'Bearer ' . $this->getAdminToken(),
                'CONTENT_TYPE' => 'application/json',
            ],
            json_encode([
                'name' => '',
                'email' => 'invalid',
                'password' => 'short',
            ])
        );

        $this->assertResponseStatusCodeSame(422);
    }

    private function getAdminToken(): string
    {
        // Create admin and authenticate; return JWT token
        // Use fixtures or factories for test data setup
        return 'test-jwt-token';
    }
}
```

### Unit Test (Service)

```php
<?php

declare(strict_types=1);

namespace App\Tests\Service;

use App\Dto\CreateUserDto;
use App\Entity\User;
use App\Event\UserCreatedEvent;
use App\Repository\UserRepository;
use App\Service\UserService;
use Doctrine\ORM\EntityManagerInterface;
use PHPUnit\Framework\MockObject\MockObject;
use PHPUnit\Framework\TestCase;
use Symfony\Component\PasswordHasher\Hasher\UserPasswordHasherInterface;
use Symfony\Contracts\EventDispatcher\EventDispatcherInterface;

class UserServiceTest extends TestCase
{
    private UserService $service;
    private MockObject $entityManager;
    private MockObject $passwordHasher;
    private MockObject $eventDispatcher;

    protected function setUp(): void
    {
        $userRepository = $this->createMock(UserRepository::class);
        $this->entityManager = $this->createMock(EntityManagerInterface::class);
        $this->passwordHasher = $this->createMock(UserPasswordHasherInterface::class);
        $this->eventDispatcher = $this->createMock(EventDispatcherInterface::class);

        $this->service = new UserService(
            $userRepository,
            $this->entityManager,
            $this->passwordHasher,
            $this->eventDispatcher,
        );
    }

    public function testCreateUserPersistsAndDispatchesEvent(): void
    {
        $dto = new CreateUserDto(
            name: 'John Doe',
            email: 'john@example.com',
            password: 'Password123!',
        );

        $this->passwordHasher
            ->expects($this->once())
            ->method('hashPassword')
            ->willReturn('hashed_password');

        $this->entityManager
            ->expects($this->once())
            ->method('persist')
            ->with($this->isInstanceOf(User::class));

        $this->entityManager
            ->expects($this->once())
            ->method('flush');

        $this->eventDispatcher
            ->expects($this->once())
            ->method('dispatch')
            ->with($this->isInstanceOf(UserCreatedEvent::class));

        $user = $this->service->create($dto);

        $this->assertEquals('john@example.com', $user->getEmail());
    }
}
```

### Testing Guidelines

- Use `WebTestCase` for functional/integration tests
- Use `TestCase` for pure unit tests with mocks
- Use `doctrine/doctrine-fixtures-bundle` for test data
- Reset database between tests (use `DAMADoctrineTestBundle` for transaction rollback)
- Test authorization: verify 401/403 for unauthenticated/unauthorized requests
- Test validation: verify 422 for invalid input
- Test happy path and error paths for every endpoint
- Name tests: `testActionResultCondition` (`testCreateUserRejectsInvalidData`)

### Test Configuration

```xml
<!-- phpunit.xml.dist -->
<phpunit bootstrap="tests/bootstrap.php">
    <testsuites>
        <testsuite name="Project Test Suite">
            <directory>tests</directory>
        </testsuite>
    </testsuites>
    <php>
        <env name="APP_ENV" value="test" />
        <env name="KERNEL_CLASS" value="App\Kernel" />
    </php>
</phpunit>
```

## Deployment

### Production Checklist

```bash
# 1. Install dependencies (no dev)
composer install --no-dev --optimize-autoloader --classmap-authoritative

# 2. Clear and warm cache
APP_ENV=prod php bin/console cache:clear
APP_ENV=prod php bin/console cache:warmup

# 3. Run migrations
php bin/console doctrine:migrations:migrate --no-interaction

# 4. Validate schema
php bin/console doctrine:schema:validate

# 5. Compile assets (if using Webpack Encore)
npm run build
```

### Environment Configuration

```bash
# .env (committed — defaults only)
APP_ENV=prod
APP_SECRET=%env(APP_SECRET)%
DATABASE_URL=%env(DATABASE_URL)%
MESSENGER_TRANSPORT_DSN=%env(MESSENGER_TRANSPORT_DSN)%

# .env.local (not committed — real values)
APP_SECRET=your-real-secret-here
DATABASE_URL="postgresql://user:pass@host:5432/db?charset=utf8"
MESSENGER_TRANSPORT_DSN=doctrine://default
```

### Deployment Best Practices

- Always run `composer install --no-dev` in production
- Use `--optimize-autoloader --classmap-authoritative` for performance
- Run `cache:warmup` after deploy to avoid first-request latency
- Run migrations in deployment pipeline, not manually
- Use environment variables for all secrets (never hardcode)
- Set `APP_ENV=prod` and `APP_DEBUG=0` in production
- Use a process manager (systemd, supervisor) for Messenger workers
- Configure OPcache with `opcache.preload` for production
- Enable HTTP caching headers for static assets

## Performance

### OPcache Configuration

```ini
; php.ini (production)
opcache.enable=1
opcache.memory_consumption=256
opcache.max_accelerated_files=20000
opcache.validate_timestamps=0
opcache.preload=/path/to/project/config/preload.php
opcache.preload_user=www-data
```

### Doctrine Performance

- Enable second-level cache for read-heavy entities
- Use `EXTRA_LAZY` fetch mode for large collections
- Avoid hydrating full entities when you only need scalar values (use `SELECT` partial)
- Use `QueryBuilder` results caching for frequently executed queries
- Profile queries with Symfony Profiler in dev; set `logging: false` in prod

### HTTP Caching

```php
#[Route('/api/v1/articles/{id}', methods: ['GET'])]
public function show(Article $article): JsonResponse
{
    $response = $this->json($article, Response::HTTP_OK, [], ['groups' => 'read']);
    $response->setMaxAge(3600);        // Cache for 1 hour
    $response->setPublic();
    $response->headers->set('ETag', md5($response->getContent()));

    return $response;
}
```

### Performance Checklist

- Enable OPcache with preloading in production
- Use Doctrine second-level cache for read-heavy data
- Set HTTP cache headers (ETag, max-age) on read endpoints
- Use Messenger for anything > 200ms (email, PDF, external APIs)
- Paginate all collection endpoints
- Use `leftJoin` + `addSelect` to prevent N+1 in Doctrine
- Profile with Symfony Profiler and Blackfire in staging
