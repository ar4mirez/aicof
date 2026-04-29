# PHP Patterns Reference

## Contents

- [Enum Patterns](#enum-patterns)
- [Readonly Class Patterns](#readonly-class-patterns)
- [Dependency Injection](#dependency-injection)
- [Repository Pattern](#repository-pattern)
- [Value Objects](#value-objects)
- [Command/Handler Pattern](#commandhandler-pattern)
- [Type Declaration Examples](#type-declaration-examples)
- [Error Handling](#error-handling)
- [Modern PHP Features](#modern-php-features)
- [Testing](#testing)
- [Tooling Configurations](#tooling-configurations)

## Enum Patterns

### Backed Enum with Interface

```php
<?php

declare(strict_types=1);

interface HasLabel
{
    public function label(): string;
}

enum Permission: string implements HasLabel
{
    case Read = 'read';
    case Write = 'write';
    case Delete = 'delete';
    case Admin = 'admin';

    public function label(): string
    {
        return match ($this) {
            self::Read => 'Can read resources',
            self::Write => 'Can create and update resources',
            self::Delete => 'Can delete resources',
            self::Admin => 'Full administrative access',
        };
    }

    public function includes(self $other): bool
    {
        if ($this === self::Admin) {
            return true;
        }

        return $this === $other;
    }

    /** @return list<self> */
    public static function forRole(string $role): array
    {
        return match ($role) {
            'viewer' => [self::Read],
            'editor' => [self::Read, self::Write],
            'admin' => self::cases(),
            default => [],
        };
    }
}
```

### Enum as State Machine

```php
<?php

declare(strict_types=1);

enum InvoiceStatus: string
{
    case Draft = 'draft';
    case Sent = 'sent';
    case Paid = 'paid';
    case Overdue = 'overdue';
    case Voided = 'voided';

    /** @return list<self> */
    public function allowedTransitions(): array
    {
        return match ($this) {
            self::Draft => [self::Sent, self::Voided],
            self::Sent => [self::Paid, self::Overdue, self::Voided],
            self::Overdue => [self::Paid, self::Voided],
            self::Paid, self::Voided => [],
        };
    }

    public function transitionTo(self $next): self
    {
        if (!in_array($next, $this->allowedTransitions(), true)) {
            throw new \LogicException(
                sprintf('Cannot transition from "%s" to "%s"', $this->value, $next->value),
            );
        }

        return $next;
    }

    public function isFinal(): bool
    {
        return $this->allowedTransitions() === [];
    }
}
```

## Readonly Class Patterns

### Value Object with Validation

```php
<?php

declare(strict_types=1);

readonly class EmailAddress
{
    private function __construct(
        public string $value,
    ) {}

    public static function fromString(string $email): self
    {
        $normalized = strtolower(trim($email));

        if (!filter_var($normalized, FILTER_VALIDATE_EMAIL)) {
            throw new \InvalidArgumentException(
                sprintf('Invalid email address: "%s"', $email),
            );
        }

        return new self($normalized);
    }

    public function domain(): string
    {
        return substr($this->value, strpos($this->value, '@') + 1);
    }

    public function equals(self $other): bool
    {
        return $this->value === $other->value;
    }

    public function __toString(): string
    {
        return $this->value;
    }
}
```

### Immutable Data Transfer Object

```php
<?php

declare(strict_types=1);

readonly class CreateUserRequest
{
    public function __construct(
        public string $name,
        public EmailAddress $email,
        public UserRole $role = UserRole::Member,
        public ?string $department = null,
    ) {}

    /** @param array<string, mixed> $data */
    public static function fromArray(array $data): self
    {
        return new self(
            name: $data['name'] ?? throw new \InvalidArgumentException('Name is required'),
            email: EmailAddress::fromString($data['email'] ?? ''),
            role: UserRole::tryFrom($data['role'] ?? '') ?? UserRole::Member,
            department: $data['department'] ?? null,
        );
    }

    public function withRole(UserRole $role): self
    {
        return new self(
            name: $this->name,
            email: $this->email,
            role: $role,
            department: $this->department,
        );
    }
}
```

### Readonly Collection Wrapper

```php
<?php

declare(strict_types=1);

/** @template T */
readonly class TypedCollection
{
    /** @param list<T> $items */
    private function __construct(
        private array $items,
    ) {}

    /** @param list<T> $items */
    public static function of(array $items): self
    {
        return new self($items);
    }

    /** @return list<T> */
    public function all(): array
    {
        return $this->items;
    }

    public function count(): int
    {
        return count($this->items);
    }

    public function isEmpty(): bool
    {
        return $this->items === [];
    }

    /**
     * @template U
     * @param callable(T): U $callback
     * @return self<U>
     */
    public function map(callable $callback): self
    {
        return new self(array_map($callback, $this->items));
    }

    /**
     * @param callable(T): bool $predicate
     * @return self<T>
     */
    public function filter(callable $predicate): self
    {
        return new self(array_values(array_filter($this->items, $predicate)));
    }
}
```

## Dependency Injection

### Constructor Injection with Interfaces

```php
<?php

declare(strict_types=1);

// Define contracts in the domain layer
interface UserRepositoryInterface
{
    public function findById(UserId $id): ?User;
    public function save(User $user): void;
    public function existsByEmail(EmailAddress $email): bool;
}

interface EventDispatcherInterface
{
    public function dispatch(object $event): void;
}

interface LoggerInterface
{
    public function info(string $message, array $context = []): void;
    public function error(string $message, array $context = []): void;
}

// Service depends only on interfaces, never on implementations
final class UserService
{
    public function __construct(
        private readonly UserRepositoryInterface $users,
        private readonly EventDispatcherInterface $events,
        private readonly LoggerInterface $logger,
    ) {}

    public function deactivate(UserId $id): void
    {
        $user = $this->users->findById($id)
            ?? throw NotFoundException::forResource('User', $id->toString());

        $user->deactivate();
        $this->users->save($user);

        $this->events->dispatch(new UserDeactivated($id));
        $this->logger->info('User deactivated', ['user_id' => $id->toString()]);
    }
}
```

### Service Container Registration

```php
<?php

declare(strict_types=1);

// Framework-agnostic container setup example
use Psr\Container\ContainerInterface;

return static function (ContainerInterface $container): void {
    // Bind interface to implementation
    $container->set(
        UserRepositoryInterface::class,
        static fn (ContainerInterface $c): DoctrineUserRepository => new DoctrineUserRepository(
            entityManager: $c->get(EntityManagerInterface::class),
        ),
    );

    // Service with multiple dependencies
    $container->set(
        UserService::class,
        static fn (ContainerInterface $c): UserService => new UserService(
            users: $c->get(UserRepositoryInterface::class),
            events: $c->get(EventDispatcherInterface::class),
            logger: $c->get(LoggerInterface::class),
        ),
    );
};
```

## Repository Pattern

### PDO-Based Repository

```php
<?php

declare(strict_types=1);

final class PdoUserRepository implements UserRepositoryInterface
{
    public function __construct(
        private readonly \PDO $pdo,
    ) {}

    public function findById(UserId $id): ?User
    {
        $stmt = $this->pdo->prepare(
            'SELECT id, name, email, role, created_at FROM users WHERE id = :id',
        );
        $stmt->execute(['id' => $id->toString()]);

        $row = $stmt->fetch(\PDO::FETCH_ASSOC);

        if ($row === false) {
            return null;
        }

        return $this->hydrate($row);
    }

    public function save(User $user): void
    {
        $stmt = $this->pdo->prepare(
            'INSERT INTO users (id, name, email, role, created_at)
             VALUES (:id, :name, :email, :role, :created_at)
             ON CONFLICT (id) DO UPDATE SET
                 name = EXCLUDED.name,
                 email = EXCLUDED.email,
                 role = EXCLUDED.role',
        );

        $stmt->execute([
            'id' => $user->id->toString(),
            'name' => $user->name,
            'email' => $user->email->value,
            'role' => $user->role->value,
            'created_at' => $user->createdAt->format('Y-m-d H:i:s'),
        ]);
    }

    public function existsByEmail(EmailAddress $email): bool
    {
        $stmt = $this->pdo->prepare('SELECT 1 FROM users WHERE email = :email LIMIT 1');
        $stmt->execute(['email' => $email->value]);

        return $stmt->fetch() !== false;
    }

    /** @param array<string, string> $row */
    private function hydrate(array $row): User
    {
        return new User(
            id: UserId::fromString($row['id']),
            name: $row['name'],
            email: EmailAddress::fromString($row['email']),
            role: UserRole::from($row['role']),
            createdAt: new \DateTimeImmutable($row['created_at']),
        );
    }
}
```

## Value Objects

### Identifier Value Object

```php
<?php

declare(strict_types=1);

readonly class UserId
{
    private function __construct(
        private string $value,
    ) {}

    public static function generate(): self
    {
        return new self(bin2hex(random_bytes(16)));
    }

    public static function fromString(string $id): self
    {
        if ($id === '' || strlen($id) !== 32) {
            throw new \InvalidArgumentException('Invalid user ID format');
        }

        return new self($id);
    }

    public function toString(): string
    {
        return $this->value;
    }

    public function equals(self $other): bool
    {
        return $this->value === $other->value;
    }
}
```

## Command/Handler Pattern

```php
<?php

declare(strict_types=1);

// Command: immutable data describing intent
readonly class RegisterUserCommand
{
    public function __construct(
        public string $name,
        public string $email,
        public string $role = 'member',
    ) {}
}

// Handler: single responsibility, one public method
final class RegisterUserHandler
{
    public function __construct(
        private readonly UserRepositoryInterface $users,
        private readonly EventDispatcherInterface $events,
    ) {}

    public function handle(RegisterUserCommand $command): UserId
    {
        $email = EmailAddress::fromString($command->email);

        if ($this->users->existsByEmail($email)) {
            throw ValidationException::duplicateEmail($command->email);
        }

        $user = User::register(
            name: $command->name,
            email: $email,
            role: UserRole::from($command->role),
        );

        $this->users->save($user);
        $this->events->dispatch(new UserRegistered($user->id));

        return $user->id;
    }
}
```

## Type Declaration Examples

### Union, Intersection, and Never Types

```php
<?php

declare(strict_types=1);

// Union types
function formatValue(string|int|float $value): string
{
    return match (true) {
        is_string($value) => $value,
        is_int($value) => number_format($value),
        is_float($value) => number_format($value, 2),
    };
}

// Intersection types (PHP 8.1+)
function countAndIterate(Countable&Iterator $collection): void
{
    echo count($collection);
    foreach ($collection as $item) {
        // process item
    }
}

// Never return type
function throwNotFound(string $resource, string $id): never
{
    throw new NotFoundException(
        sprintf('%s with ID "%s" not found', $resource, $id),
    );
}
```

## Error Handling

### Domain Exception Hierarchy

```php
<?php

declare(strict_types=1);

namespace App\Domain\Exception;

abstract class DomainException extends \RuntimeException
{
    public function __construct(
        string $message,
        public readonly string $errorCode = 'UNKNOWN',
        int $code = 0,
        ?\Throwable $previous = null,
    ) {
        parent::__construct($message, $code, $previous);
    }
}

final class NotFoundException extends DomainException
{
    public static function forResource(string $resource, string $id): self
    {
        return new self(
            message: sprintf('%s with ID "%s" not found', $resource, $id),
            errorCode: 'NOT_FOUND',
        );
    }
}

final class ValidationException extends DomainException
{
    /** @param array<string, string[]> $violations */
    public static function withViolations(array $violations): self
    {
        return new self(
            message: 'Validation failed: ' . json_encode($violations, JSON_THROW_ON_ERROR),
            errorCode: 'VALIDATION_ERROR',
        );
    }

    public static function duplicateEmail(string $email): self
    {
        return new self(
            message: sprintf('Email "%s" is already registered', $email),
            errorCode: 'DUPLICATE_EMAIL',
        );
    }
}
```

## Modern PHP Features

### Backed Enum with State Transitions (OrderStatus)

```php
<?php

declare(strict_types=1);

enum OrderStatus: string
{
    case Pending = 'pending';
    case Confirmed = 'confirmed';
    case Shipped = 'shipped';
    case Delivered = 'delivered';
    case Cancelled = 'cancelled';

    public function label(): string
    {
        return match ($this) {
            self::Pending => 'Order Pending',
            self::Confirmed => 'Order Confirmed',
            self::Shipped => 'In Transit',
            self::Delivered => 'Delivered',
            self::Cancelled => 'Cancelled',
        };
    }

    public function canTransitionTo(self $next): bool
    {
        return match ($this) {
            self::Pending => in_array($next, [self::Confirmed, self::Cancelled], true),
            self::Confirmed => in_array($next, [self::Shipped, self::Cancelled], true),
            self::Shipped => $next === self::Delivered,
            self::Delivered, self::Cancelled => false,
        };
    }
}
```

### Readonly Classes

```php
<?php

declare(strict_types=1);

readonly class Money
{
    public function __construct(
        public int $amount,
        public string $currency,
    ) {}

    public function add(self $other): self
    {
        if ($this->currency !== $other->currency) {
            throw new \InvalidArgumentException('Cannot add different currencies');
        }

        return new self($this->amount + $other->amount, $this->currency);
    }

    public function isPositive(): bool
    {
        return $this->amount > 0;
    }
}
```

### Named Arguments

```php
<?php

declare(strict_types=1);

// Named arguments improve readability for functions with many parameters
$user = new User(
    name: 'Ada Lovelace',
    email: Email::fromString('ada@example.com'),
    role: UserRole::Admin,
    isActive: true,
);

// Particularly useful with optional parameters
$response = $httpClient->request(
    method: 'POST',
    url: '/api/users',
    body: $payload,
    timeout: 30,
    retries: 3,
);
```

### Match Expressions

```php
<?php

declare(strict_types=1);

// Prefer match over switch -- strict comparison, expression-based, no fallthrough
function httpStatusMessage(int $code): string
{
    return match (true) {
        $code >= 200 && $code < 300 => 'Success',
        $code >= 300 && $code < 400 => 'Redirection',
        $code >= 400 && $code < 500 => 'Client Error',
        $code >= 500 => 'Server Error',
        default => 'Unknown',
    };
}

// Simple value matching
function toHumanSize(int $bytes): string
{
    return match (true) {
        $bytes < 1024 => $bytes . ' B',
        $bytes < 1048576 => round($bytes / 1024, 1) . ' KB',
        $bytes < 1073741824 => round($bytes / 1048576, 1) . ' MB',
        default => round($bytes / 1073741824, 1) . ' GB',
    };
}
```

### Fibers (PHP 8.1+)

```php
<?php

declare(strict_types=1);

// Fibers enable cooperative multitasking (foundation for async frameworks)
function asyncFetch(string $url): Fiber
{
    return new Fiber(function () use ($url): string {
        // Simulate async I/O -- suspend while waiting
        Fiber::suspend('waiting');

        // In practice, an event loop resumes the fiber when data is ready
        $response = file_get_contents($url);

        if ($response === false) {
            throw new \RuntimeException("Failed to fetch: {$url}");
        }

        return $response;
    });
}

// Usage with a scheduler
$fiber = asyncFetch('https://api.example.com/data');
$fiber->start();              // Runs until Fiber::suspend()
// ... do other work ...
$result = $fiber->resume();   // Resume and get return value
```

## Testing

### Mocking with Mockery

```php
<?php

declare(strict_types=1);

namespace Tests\Unit\Application\User;

use App\Application\User\RegisterUserHandler;
use App\Domain\User\User;
use App\Domain\User\UserRepositoryInterface;
use App\Domain\Event\EventDispatcherInterface;
use Mockery;
use Mockery\Adapter\Phpunit\MockeryPHPUnitIntegration;
use PHPUnit\Framework\Attributes\Test;
use PHPUnit\Framework\TestCase;

final class RegisterUserHandlerTest extends TestCase
{
    use MockeryPHPUnitIntegration;

    #[Test]
    public function it_registers_a_new_user(): void
    {
        $repository = Mockery::mock(UserRepositoryInterface::class);
        $dispatcher = Mockery::mock(EventDispatcherInterface::class);

        $repository->expects('existsByEmail')->once()->andReturnFalse();
        $repository->expects('save')->once()->with(Mockery::type(User::class));
        $dispatcher->expects('dispatch')->once();

        $handler = new RegisterUserHandler($repository, $dispatcher);
        $user = $handler->handle(name: 'Ada Lovelace', email: 'ada@example.com');

        self::assertSame('Ada Lovelace', $user->name);
    }
}
```

## Tooling Configurations

### composer.json

```json
{
    "require": {
        "php": ">=8.1"
    },
    "require-dev": {
        "phpunit/phpunit": "^10.0",
        "phpstan/phpstan": "^1.10",
        "friendsofphp/php-cs-fixer": "^3.0",
        "mockery/mockery": "^1.6",
        "vimeo/psalm": "^5.0"
    },
    "autoload": {
        "psr-4": {
            "App\\": "src/"
        }
    },
    "autoload-dev": {
        "psr-4": {
            "Tests\\": "tests/"
        }
    },
    "config": {
        "sort-packages": true,
        "allow-plugins": {}
    }
}
```

### PHPStan Configuration

```neon
# phpstan.neon
parameters:
    level: 8
    paths:
        - src
    excludePaths:
        - src/Kernel.php
    treatPhpDocTypesAsCertain: false
    reportUnmatchedIgnoredErrors: true
    checkMissingIterableValueType: true
    checkGenericClassInNonGenericObjectType: true
```

### PHP-CS-Fixer Configuration

```php
<?php
// .php-cs-fixer.php
use PhpCsFixer\Config;
use PhpCsFixer\Finder;

$finder = Finder::create()
    ->in(__DIR__ . '/src')
    ->in(__DIR__ . '/tests');

return (new Config())
    ->setRiskyAllowed(true)
    ->setRules([
        '@PSR12' => true,
        'strict_param' => true,
        'declare_strict_types' => true,
        'array_syntax' => ['syntax' => 'short'],
        'no_unused_imports' => true,
        'ordered_imports' => ['sort_algorithm' => 'alpha'],
        'trailing_comma_in_multiline' => ['elements' => ['arguments', 'arrays', 'parameters']],
        'native_function_invocation' => ['include' => ['@all']],
        'global_namespace_import' => ['import_classes' => true, 'import_functions' => true],
    ])
    ->setFinder($finder);
```
