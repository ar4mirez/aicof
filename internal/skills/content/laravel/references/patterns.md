# Laravel Patterns Reference

## Contents

- [Service Layer](#service-layer)
- [Eloquent Advanced Patterns](#eloquent-advanced-patterns)
- [API Resources](#api-resources)
- [Jobs and Queues](#jobs-and-queues)
- [Events and Listeners](#events-and-listeners)
- [Livewire Components](#livewire-components)
- [Authorization Policies](#authorization-policies)
- [Model Factories](#model-factories)
- [Notifications](#notifications)
- [Caching](#caching)
- [Deployment](#deployment)
- [Common Pitfalls](#common-pitfalls)

## Service Layer

Extract business logic out of controllers into dedicated service classes.

```php
<?php

declare(strict_types=1);

namespace App\Services;

use App\Events\PostPublished;
use App\Models\Post;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Str;

class PostService
{
    /**
     * Create a new post within a transaction.
     *
     * @param array<string, mixed> $data
     */
    public function create(array $data): Post
    {
        return DB::transaction(function () use ($data): Post {
            $post = Post::create([
                'title' => $data['title'],
                'slug' => $data['slug'] ?? Str::slug($data['title']),
                'body' => $data['body'],
                'status' => $data['status'] ?? 'draft',
                'user_id' => $data['user_id'],
            ]);

            if (isset($data['tags'])) {
                $post->tags()->sync($data['tags']);
            }

            return $post;
        });
    }

    /**
     * Publish a post and dispatch event.
     */
    public function publish(Post $post): Post
    {
        $post->update([
            'status' => 'published',
            'published_at' => now(),
        ]);

        event(new PostPublished($post));

        return $post->fresh();
    }

    /**
     * Update with selective field handling.
     *
     * @param array<string, mixed> $data
     */
    public function update(Post $post, array $data): Post
    {
        $post->update(collect($data)->only([
            'title', 'slug', 'body', 'status',
        ])->toArray());

        if (isset($data['tags'])) {
            $post->tags()->sync($data['tags']);
        }

        return $post->fresh();
    }

    public function delete(Post $post): bool
    {
        return $post->delete();
    }
}
```

### Service Layer Rules

- One service per domain model or bounded context
- Wrap multi-step operations in `DB::transaction()`
- Dispatch events from services, not controllers
- Accept validated arrays (from Form Requests), return models
- Use constructor injection for dependencies

## Eloquent Advanced Patterns

### Query Scopes with Composition

```php
// Combine scopes for readable, reusable queries
$posts = Post::query()
    ->published()
    ->byAuthor($userId)
    ->withCategory('tech')
    ->recent()
    ->paginate(15);
```

### Eager Loading (Preventing N+1)

```php
// Bad: N+1 queries (1 query for posts + N queries for authors)
$posts = Post::all();
foreach ($posts as $post) {
    echo $post->author->name; // Triggers query each iteration
}

// Good: Eager load (2 queries total)
$posts = Post::with('author')->get();

// Good: Nested eager loading
$posts = Post::with(['author', 'comments.user'])->get();

// Good: Constrained eager loading
$posts = Post::with(['comments' => function ($query) {
    $query->where('approved', true)->latest()->limit(5);
}])->get();
```

### Eloquent Accessors and Mutators (Laravel 11 Style)

```php
use Illuminate\Database\Eloquent\Casts\Attribute;

class User extends Model
{
    protected function fullName(): Attribute
    {
        return Attribute::make(
            get: fn () => "{$this->first_name} {$this->last_name}",
        );
    }

    protected function email(): Attribute
    {
        return Attribute::make(
            set: fn (string $value) => strtolower($value),
        );
    }
}
```

### Polymorphic Relationships

```php
// Comment can belong to Post or Video
class Comment extends Model
{
    public function commentable(): MorphTo
    {
        return $this->morphTo();
    }
}

class Post extends Model
{
    public function comments(): MorphMany
    {
        return $this->morphMany(Comment::class, 'commentable');
    }
}
```

### Query Optimization

```php
// Use chunk for large datasets (avoids memory exhaustion)
Post::where('status', 'draft')
    ->chunkById(200, function ($posts) {
        foreach ($posts as $post) {
            $post->update(['status' => 'archived']);
        }
    });

// Use cursor for streaming (very low memory)
foreach (Post::cursor() as $post) {
    // Process one at a time
}

// Select only needed columns
$emails = User::query()
    ->select('id', 'email')
    ->where('is_active', true)
    ->pluck('email');
```

## API Resources

### Resource with Conditional Data

```php
<?php

declare(strict_types=1);

namespace App\Http\Resources;

use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\JsonResource;

class PostResource extends JsonResource
{
    public function toArray(Request $request): array
    {
        return [
            'id' => $this->id,
            'title' => $this->title,
            'slug' => $this->slug,
            'excerpt' => $this->excerpt,
            'status' => $this->status,
            'created_at' => $this->created_at->toIso8601String(),

            // Include body only on show (not index)
            'body' => $this->when(
                $request->routeIs('posts.show'),
                $this->body,
            ),

            // Conditional relationships
            'author' => UserResource::make($this->whenLoaded('author')),
            'comments' => CommentResource::collection($this->whenLoaded('comments')),

            // Conditional counts
            'comments_count' => $this->when(
                $this->comments_count !== null,
                $this->comments_count,
            ),
        ];
    }
}
```

### Resource Collection with Meta

```php
<?php

declare(strict_types=1);

namespace App\Http\Resources;

use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\ResourceCollection;

class PostCollection extends ResourceCollection
{
    public function toArray(Request $request): array
    {
        return [
            'data' => $this->collection,
            'meta' => [
                'total' => $this->total(),
                'per_page' => $this->perPage(),
                'current_page' => $this->currentPage(),
            ],
        ];
    }
}
```

## Jobs and Queues

### Job with Retry and Failure Handling

```php
<?php

declare(strict_types=1);

namespace App\Jobs;

use App\Models\User;
use App\Services\ReportService;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldBeUnique;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;

class GenerateUserReport implements ShouldQueue, ShouldBeUnique
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    public int $tries = 3;
    public int $backoff = 60;        // Seconds between retries
    public int $timeout = 300;       // Max execution time
    public int $maxExceptions = 2;   // Release after N exceptions

    public function __construct(
        public readonly User $user,
        public readonly string $reportType,
    ) {}

    public function handle(ReportService $reportService): void
    {
        $reportService->generate($this->user, $this->reportType);
    }

    /** Unique lock key to prevent duplicate jobs. */
    public function uniqueId(): string
    {
        return "{$this->user->id}-{$this->reportType}";
    }

    /** Handle job failure after all retries exhausted. */
    public function failed(\Throwable $exception): void
    {
        logger()->error('Report generation failed', [
            'user_id' => $this->user->id,
            'type' => $this->reportType,
            'error' => $exception->getMessage(),
        ]);
    }
}
```

### Dispatching Jobs

```php
// Dispatch to default queue
GenerateUserReport::dispatch($user, 'monthly');

// Dispatch to specific queue
GenerateUserReport::dispatch($user, 'monthly')
    ->onQueue('reports');

// Delay dispatch
GenerateUserReport::dispatch($user, 'monthly')
    ->delay(now()->addMinutes(5));

// Job chaining (run sequentially)
Bus::chain([
    new FetchData($user),
    new ProcessData($user),
    new SendNotification($user),
])->dispatch();

// Job batching (run in parallel with progress tracking)
Bus::batch([
    new ProcessChunk($chunk1),
    new ProcessChunk($chunk2),
    new ProcessChunk($chunk3),
])->then(function (Batch $batch) {
    // All jobs completed
})->catch(function (Batch $batch, \Throwable $e) {
    // First failure
})->dispatch();
```

### Queue Rules

- Always implement `ShouldQueue` for async processing
- Set `$tries`, `$backoff`, and `$timeout` explicitly
- Implement `failed()` for error handling
- Use `ShouldBeUnique` to prevent duplicate processing
- Use job batching for bulk operations with progress tracking
- Use Horizon for queue monitoring in production

## Events and Listeners

### Event Definition

```php
<?php

declare(strict_types=1);

namespace App\Events;

use App\Models\Post;
use Illuminate\Broadcasting\InteractsWithSockets;
use Illuminate\Foundation\Events\Dispatchable;
use Illuminate\Queue\SerializesModels;

class PostPublished
{
    use Dispatchable, InteractsWithSockets, SerializesModels;

    public function __construct(
        public readonly Post $post,
    ) {}
}
```

### Queued Listener

```php
<?php

declare(strict_types=1);

namespace App\Listeners;

use App\Events\PostPublished;
use App\Notifications\NewPostNotification;
use Illuminate\Contracts\Queue\ShouldQueue;

class NotifySubscribers implements ShouldQueue
{
    public string $queue = 'notifications';

    public function handle(PostPublished $event): void
    {
        $subscribers = $event->post->author
            ->subscribers()
            ->get();

        foreach ($subscribers as $subscriber) {
            $subscriber->notify(new NewPostNotification($event->post));
        }
    }

    /** Determine if listener should be queued. */
    public function shouldQueue(PostPublished $event): bool
    {
        return $event->post->status === 'published';
    }
}
```

### Event Rules

- Use readonly constructor promotion for event properties
- Make listeners implement `ShouldQueue` for async processing
- Use `shouldQueue()` for conditional queueing
- Register events in `EventServiceProvider` or use auto-discovery
- Keep events simple (data containers), logic goes in listeners

## Livewire Components

### Component with State

```php
<?php

declare(strict_types=1);

namespace App\Livewire;

use App\Models\Post;
use Livewire\Component;
use Livewire\WithPagination;

class PostList extends Component
{
    use WithPagination;

    public string $search = '';
    public string $sortBy = 'created_at';
    public string $sortDir = 'desc';

    // Reset pagination when search changes
    public function updatedSearch(): void
    {
        $this->resetPage();
    }

    public function sort(string $column): void
    {
        if ($this->sortBy === $column) {
            $this->sortDir = $this->sortDir === 'asc' ? 'desc' : 'asc';
        } else {
            $this->sortBy = $column;
            $this->sortDir = 'asc';
        }
    }

    public function render()
    {
        $posts = Post::query()
            ->with('author')
            ->when($this->search, fn ($q) => $q->where('title', 'like', "%{$this->search}%"))
            ->orderBy($this->sortBy, $this->sortDir)
            ->paginate(10);

        return view('livewire.post-list', compact('posts'));
    }
}
```

### Livewire Rules

- Use `WithPagination` for paginated lists
- Reset pagination on filter changes with `updatedPropertyName()`
- Use `wire:model.live` sparingly (debounce with `wire:model.live.debounce.300ms`)
- Validate with `$this->validate()` using `$rules` property
- Use `$this->dispatch()` for cross-component communication

## Authorization Policies

```php
<?php

declare(strict_types=1);

namespace App\Policies;

use App\Models\Post;
use App\Models\User;

class PostPolicy
{
    /** Any authenticated user can view published posts. */
    public function view(?User $user, Post $post): bool
    {
        return $post->status === 'published'
            || $user?->id === $post->user_id;
    }

    /** Only the author can update their post. */
    public function update(User $user, Post $post): bool
    {
        return $user->id === $post->user_id;
    }

    /** Only admins or the author can delete. */
    public function delete(User $user, Post $post): bool
    {
        return $user->role === 'admin'
            || $user->id === $post->user_id;
    }
}
```

Register in `AuthServiceProvider` or use auto-discovery (Laravel 11+).

## Model Factories

### Factory with States and Relationships

```php
<?php

declare(strict_types=1);

namespace Database\Factories;

use App\Models\User;
use Illuminate\Database\Eloquent\Factories\Factory;
use Illuminate\Support\Str;

class PostFactory extends Factory
{
    public function definition(): array
    {
        $title = fake()->sentence();

        return [
            'user_id' => User::factory(),
            'title' => $title,
            'slug' => Str::slug($title),
            'body' => fake()->paragraphs(3, true),
            'status' => 'draft',
            'is_featured' => false,
        ];
    }

    public function published(): static
    {
        return $this->state(fn () => [
            'status' => 'published',
            'published_at' => now()->subDays(rand(1, 30)),
        ]);
    }

    public function featured(): static
    {
        return $this->state(fn () => [
            'is_featured' => true,
        ]);
    }
}

// Usage in tests:
// Post::factory()->published()->featured()->create();
// Post::factory()->count(5)->for($user)->create();
```

## Notifications

### Multi-Channel Notification

```php
<?php

declare(strict_types=1);

namespace App\Notifications;

use App\Models\Post;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Notifications\Messages\MailMessage;
use Illuminate\Notifications\Notification;

class NewPostNotification extends Notification implements ShouldQueue
{
    use Queueable;

    public function __construct(
        public readonly Post $post,
    ) {}

    /** @return array<string> */
    public function via(object $notifiable): array
    {
        return ['mail', 'database'];
    }

    public function toMail(object $notifiable): MailMessage
    {
        return (new MailMessage())
            ->subject("New post: {$this->post->title}")
            ->line("{$this->post->author->name} published a new post.")
            ->action('Read Post', url("/posts/{$this->post->slug}"));
    }

    /** @return array<string, mixed> */
    public function toArray(object $notifiable): array
    {
        return [
            'post_id' => $this->post->id,
            'title' => $this->post->title,
            'author' => $this->post->author->name,
        ];
    }
}
```

## Caching

```php
use Illuminate\Support\Facades\Cache;

// Cache with expiration
$posts = Cache::remember('featured-posts', now()->addHour(), function () {
    return Post::query()
        ->published()
        ->featured()
        ->with('author')
        ->latest()
        ->take(10)
        ->get();
});

// Cache tags (Redis/Memcached only)
Cache::tags(['posts'])->remember('post-' . $id, 3600, fn () => Post::find($id));

// Invalidate cache on model events
class Post extends Model
{
    protected static function booted(): void
    {
        static::saved(fn () => Cache::tags(['posts'])->flush());
        static::deleted(fn () => Cache::tags(['posts'])->flush());
    }
}
```

### Caching Rules

- Use `Cache::remember()` with appropriate TTL (time-to-live)
- Invalidate cache when underlying data changes
- Use cache tags for grouped invalidation (Redis required)
- Never cache user-specific data in shared cache keys
- Use `Cache::lock()` for atomic operations

## Deployment

### Production Optimization

```bash
# Optimize autoloader
composer install --optimize-autoloader --no-dev

# Cache configuration, routes, and views
php artisan optimize

# Individual cache commands
php artisan config:cache    # Config to single file
php artisan route:cache     # Routes to single file
php artisan view:cache      # Compile Blade templates
php artisan event:cache     # Event/listener mappings

# Run migrations
php artisan migrate --force
```

### Environment Configuration

```bash
# .env.example (commit this, never .env)
APP_NAME=MyApp
APP_ENV=production
APP_DEBUG=false
APP_URL=https://myapp.com

DB_CONNECTION=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_DATABASE=myapp
DB_USERNAME=
DB_PASSWORD=

CACHE_DRIVER=redis
QUEUE_CONNECTION=redis
SESSION_DRIVER=redis

MAIL_MAILER=ses
```

### Deployment Checklist

- `APP_DEBUG=false` in production (never expose stack traces)
- `APP_ENV=production`
- Run `php artisan optimize` after deployment
- Use Redis for cache, sessions, and queues in production
- Run `php artisan migrate --force` (auto-confirms migrations)
- Configure queue workers with Supervisor or Horizon
- Set up health check endpoint for load balancer
- Enable HTTPS and force via `APP_URL` and `TrustProxies` middleware

## Common Pitfalls

### Do

- Use `$fillable` whitelist on all models
- Eager load relationships to avoid N+1
- Use Form Requests for all input validation
- Use database transactions for multi-step writes
- Queue emails, notifications, and heavy processing
- Use `declare(strict_types=1)` in every file
- Use model factories in tests (not raw SQL inserts)
- Add `down()` method to every migration

### Don't

- Don't put business logic in controllers or models
- Don't use `DB::raw()` without parameter binding
- Don't use `$guarded = []` (disables mass assignment protection)
- Don't use `{!! !!}` in Blade unless HTML is explicitly trusted
- Don't commit `.env` files to version control
- Don't use `*` in select queries in production code
- Don't use synchronous mail/notification sending in request cycle
- Don't modify deployed migrations; create new ones instead
- Don't ignore the N+1 query detector in development (`barryvdh/laravel-debugbar`)
