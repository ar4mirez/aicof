# Hanami Advanced Patterns

Detailed patterns, validation, interactors, testing, assets, and deployment for Hanami 2+ applications.

---

## ROM Advanced Patterns

### Associations & Eager Loading

```ruby
# lib/myapp/persistence/relations/posts.rb
module MyApp
  module Persistence
    module Relations
      class Posts < ROM::Relation[:sql]
        schema(:posts, infer: true) do
          associations do
            belongs_to :user
            has_many :comments
            has_many :taggings
            has_many :tags, through: :taggings
          end
        end

        def published      = where(published: true)
        def recent          = order { created_at.desc }
        def by_user(uid)   = where(user_id: uid)
        def by_tag(tag)    = join(:taggings).join(:tags).where(tags__name: tag)
      end
    end
  end
end

# lib/myapp/persistence/relations/comments.rb
module MyApp
  module Persistence
    module Relations
      class Comments < ROM::Relation[:sql]
        schema(:comments, infer: true) do
          associations do
            belongs_to :user
            belongs_to :post
          end
        end

        def by_post(post_id) = where(post_id: post_id)
        def recent            = order { created_at.desc }
      end
    end
  end
end
```

### Advanced Repository Patterns

```ruby
# lib/myapp/repositories/post_repo.rb
module MyApp
  module Repositories
    class PostRepo < ROM::Repository[:posts]
      include Deps[container: "persistence.rom"]

      commands :create, update: :by_pk, delete: :by_pk
      struct_namespace MyApp::Entities

      # Eager load author for a single post
      def with_author(id)
        posts.by_pk(id).combine(:user).one
      end

      # Eager load posts with their authors (batch, avoids N+1)
      def recent_with_authors(limit: 10)
        posts.published.recent.limit(limit).combine(:user).to_a
      end

      # Eager load nested: post -> comments -> comment authors
      def with_comments(id)
        posts.by_pk(id).combine(comments: :user).one
      end

      # Aggregate with counting (subquery)
      def popular(min_comments: 5)
        posts
          .published
          .qualified
          .left_join(:comments, post_id: :id)
          .group(:id)
          .having { count(comments[:id]) >= min_comments }
          .to_a
      end

      # Transactional multi-table write
      def create_with_tags(attrs, tag_ids)
        posts.transaction do
          post = create(attrs)
          tag_ids.each do |tag_id|
            taggings.create(post_id: post.id, tag_id: tag_id)
          end
          post
        end
      end

      # Search with ILIKE (PostgreSQL)
      def search(query)
        posts.where { title.ilike("%#{query}%") | body.ilike("%#{query}%") }.to_a
      end
    end
  end
end
```

### Custom Mappers & Projections

```ruby
# Select only specific columns (projection)
def user_summaries
  users.select(:id, :name, :email).to_a
end

# Map to custom struct
def user_with_post_count
  users
    .qualified
    .left_join(:posts, user_id: :id)
    .select_append { int::count(posts[:id]).as(:post_count) }
    .group(:id)
    .to_a
end
```

### Entities with Behavior

```ruby
# lib/myapp/entities/user.rb
module MyApp
  module Entities
    class User < ROM::Struct
      def full_name
        "#{first_name} #{last_name}".strip
      end

      def admin?
        role == "admin"
      end

      def can_publish?(post)
        admin? || post.user_id == id
      end
    end
  end
end

# lib/myapp/entities/post.rb
module MyApp
  module Entities
    class Post < ROM::Struct
      def published? = published == true
      def draft?     = !published?

      def reading_time
        words = body.split.size
        (words / 200.0).ceil
      end

      def excerpt(length: 150)
        body.length > length ? "#{body[0...length]}..." : body
      end
    end
  end
end
```

### Migrations Best Practices

```ruby
# db/migrate/20240201000001_create_posts.rb
ROM::SQL.migration do
  change do
    create_table :posts do
      primary_key :id
      foreign_key :user_id, :users, null: false, on_delete: :cascade
      column :title, String, null: false
      column :slug, String, null: false, unique: true
      column :body, :text, null: false
      column :published, TrueClass, default: false
      column :published_at, DateTime
      column :created_at, DateTime, null: false
      column :updated_at, DateTime, null: false
    end

    add_index :posts, :user_id
    add_index :posts, :slug, unique: true
    add_index :posts, [:published, :published_at]
  end
end

# db/migrate/20240201000002_add_role_to_users.rb
# Always include up AND down for reversibility
ROM::SQL.migration do
  up do
    add_column :users, :role, String, default: "user", null: false
    add_index :users, :role
  end

  down do
    drop_index :users, :role
    drop_column :users, :role
  end
end
```

---

## Dry-Validation Contracts

### Standalone Validation Contracts

```ruby
# lib/myapp/contracts/user_contract.rb
require "dry/validation"

module MyApp
  module Contracts
    class UserContract < Dry::Validation::Contract
      params do
        required(:email).filled(:string)
        required(:name).filled(:string, min_size?: 2, max_size?: 100)
        required(:password).filled(:string, min_size?: 8)
        optional(:bio).maybe(:string, max_size?: 500)
        optional(:role).filled(:string, included_in?: %w[user admin moderator])
      end

      rule(:email) do
        unless /\A[\w+\-.]+@[a-z\d\-]+(\.[a-z\d\-]+)*\.[a-z]+\z/i.match?(value)
          key.failure("has invalid format")
        end
      end

      rule(:password) do
        unless value.match?(/[A-Z]/) && value.match?(/[0-9]/)
          key.failure("must contain at least one uppercase letter and one number")
        end
      end
    end
  end
end
```

### Using Contracts in Operations

```ruby
# lib/myapp/operations/users/create.rb
require "dry/monads"

module MyApp
  module Operations
    module Users
      class Create
        include Dry::Monads[:result, :do]
        include Deps[
          "repositories.user_repo",
          "contracts.user_contract",
          "services.password_hasher"
        ]

        def call(params)
          validated = yield validate(params)
          yield check_uniqueness(validated[:email])
          user = yield persist(validated)
          Success(user)
        end

        private

        def validate(params)
          result = user_contract.call(params)
          result.success? ? Success(result.to_h) : Failure(result.errors.to_h)
        end

        def check_uniqueness(email)
          user_repo.find_by_email(email) ? Failure(email: ["already taken"]) : Success()
        end

        def persist(validated)
          user = user_repo.create(
            email: validated[:email].downcase.strip,
            name: validated[:name].strip,
            password_digest: password_hasher.hash(validated[:password]),
            created_at: Time.now,
            updated_at: Time.now
          )
          Success(user)
        rescue => e
          Hanami.logger.error(e)
          Failure(base: ["An unexpected error occurred"])
        end
      end
    end
  end
end
```

### Custom Types with Dry-Types

```ruby
# lib/myapp/types.rb
require "dry/types"

module MyApp
  module Types
    include Dry.Types()

    Email = String.constrained(format: /\A[\w+\-.]+@[a-z\d\-]+(\.[a-z\d\-]+)*\.[a-z]+\z/i)
    Slug = String.constrained(format: /\A[a-z0-9]+(?:-[a-z0-9]+)*\z/)
    Role = String.enum("user", "admin", "moderator")
    PositiveInt = Integer.constrained(gt: 0)
    Pagination = Hash.schema(page: PositiveInt, per_page: PositiveInt.constrained(lteq: 100))
  end
end
```

---

## Interactor Pipelines

### Monadic Do Notation

```ruby
# lib/myapp/operations/posts/publish.rb
require "dry/monads"

module MyApp
  module Operations
    module Posts
      class Publish
        include Dry::Monads[:result, :do]
        include Deps[
          "repositories.post_repo",
          "services.notifier"
        ]

        def call(post_id, publisher:)
          post = yield find_post(post_id)
          yield authorize(post, publisher)
          yield check_publishable(post)
          published = yield perform_publish(post)
          yield notify_subscribers(published)
          Success(published)
        end

        private

        def find_post(id)
          post = post_repo.find(id)
          post ? Success(post) : Failure(:not_found)
        end

        def authorize(post, publisher)
          publisher.can_publish?(post) ? Success() : Failure(:unauthorized)
        end

        def check_publishable(post)
          post.draft? ? Success() : Failure(:already_published)
        end

        def perform_publish(post)
          updated = post_repo.update(post.id,
            published: true,
            published_at: Time.now,
            updated_at: Time.now
          )
          Success(updated)
        end

        def notify_subscribers(post)
          notifier.post_published(post)
          Success()
        rescue => e
          Hanami.logger.warn("Notification failed: #{e.message}")
          Success() # Non-critical, don't fail the operation
        end
      end
    end
  end
end
```

### Composing Operations

```ruby
# lib/myapp/operations/registration/complete.rb
# Chain multiple operations together
module MyApp
  module Operations
    module Registration
      class Complete
        include Dry::Monads[:result, :do]
        include Deps[
          "operations.users.create",
          "operations.profiles.create",
          "services.mailer"
        ]

        def call(params)
          user = yield create_user.call(params[:user])
          yield create_profile.call(user_id: user.id, **params[:profile])
          yield send_welcome_email(user)
          Success(user)
        end

        private

        def send_welcome_email(user)
          mailer.deliver(:welcome, to: user.email, name: user.name)
          Success()
        rescue => e
          Hanami.logger.warn("Welcome email failed: #{e.message}")
          Success() # Non-critical
        end
      end
    end
  end
end
```

### Error Mapping in Actions

```ruby
# Mapping operation failures to HTTP responses
module MyApp
  module Actions
    module Posts
      class Publish < MyApp::Action
        include Deps["operations.posts.publish"]

        def handle(request, response)
          require_authentication!

          result = publish.call(request.params[:id], publisher: current_user)

          case result
          in Dry::Monads::Success(post)
            response.flash[:success] = "Post published"
            response.redirect_to routes.path(:posts_show, id: post.id)
          in Dry::Monads::Failure(:not_found)
            halt 404
          in Dry::Monads::Failure(:unauthorized)
            halt 403
          in Dry::Monads::Failure(:already_published)
            response.flash[:error] = "Post is already published"
            response.redirect_to routes.path(:posts_show, id: request.params[:id])
          in Dry::Monads::Failure(errors)
            response.render(view, errors: errors)
          end
        end
      end
    end
  end
end
```

---

## Testing Patterns

### RSpec Setup

```ruby
# spec/spec_helper.rb
ENV["HANAMI_ENV"] ||= "test"

require "hanami/prepare"
require "database_cleaner/sequel"

RSpec.configure do |config|
  config.before(:suite) do
    DatabaseCleaner.strategy = :transaction
    DatabaseCleaner.clean_with(:truncation)
  end

  config.around(:each) do |example|
    DatabaseCleaner.cleaning { example.run }
  end
end

# spec/support/requests.rb
module RequestHelpers
  def app = Hanami.app

  def json_response
    JSON.parse(last_response.body, symbolize_names: true)
  end

  def post_json(path, body = {}, headers = {})
    post path, body.to_json, headers.merge("CONTENT_TYPE" => "application/json")
  end

  def auth_headers(user)
    token = MyApp::App["services.jwt_encoder"].encode(user_id: user.id)
    { "HTTP_AUTHORIZATION" => "Bearer #{token}" }
  end
end

RSpec.configure do |config|
  config.include RequestHelpers, type: :request
  config.include Rack::Test::Methods, type: :request
end
```

### Factory Bot Setup

```ruby
# spec/support/factories.rb
require "factory_bot"

RSpec.configure do |config|
  config.include FactoryBot::Syntax::Methods

  config.before(:suite) do
    FactoryBot.find_definitions
  end
end

# spec/factories/users.rb
FactoryBot.define do
  factory :user, class: "MyApp::Entities::User" do
    sequence(:email) { |n| "user#{n}@example.com" }
    name { "Test User" }
    password_digest { BCrypt::Password.create("password123") }
    role { "user" }
    active { true }
    created_at { Time.now }
    updated_at { Time.now }

    trait :admin do
      role { "admin" }
    end

    trait :inactive do
      active { false }
    end
  end

  factory :post, class: "MyApp::Entities::Post" do
    association :user
    sequence(:title) { |n| "Post #{n}" }
    body { "Post body content" }
    published { false }
    created_at { Time.now }
    updated_at { Time.now }

    trait :published do
      published { true }
      published_at { Time.now }
    end
  end
end
```

### Operation Tests (Unit)

```ruby
# spec/operations/users/create_spec.rb
RSpec.describe MyApp::Operations::Users::Create do
  subject(:operation) { described_class.new }
  let(:user_repo) { MyApp::App["repositories.user_repo"] }

  let(:valid_params) do
    { email: "test@example.com", name: "Test User", password: "Password1" }
  end

  it "creates a user with valid params" do
    result = operation.call(valid_params)

    expect(result).to be_success
    expect(result.value!.email).to eq("test@example.com")
  end

  it "fails with duplicate email" do
    operation.call(valid_params)
    result = operation.call(valid_params)

    expect(result).to be_failure
    expect(result.failure[:email]).to include("already taken")
  end

  it "normalizes email to lowercase" do
    result = operation.call(valid_params.merge(email: "TEST@Example.COM"))

    expect(result).to be_success
    expect(result.value!.email).to eq("test@example.com")
  end

  it "fails with invalid email format" do
    result = operation.call(valid_params.merge(email: "invalid"))

    expect(result).to be_failure
    expect(result.failure[:email]).to be_present
  end
end
```

### Action Tests (Integration)

```ruby
# spec/actions/users/index_spec.rb
RSpec.describe MyApp::Actions::Users::Index, type: :request do
  let(:user_repo) { MyApp::App["repositories.user_repo"] }

  before do
    user_repo.create(
      email: "test@example.com", name: "Test User",
      password_digest: BCrypt::Password.create("password"),
      created_at: Time.now, updated_at: Time.now
    )
  end

  it "returns 200 with users" do
    get "/users"

    expect(last_response.status).to eq(200)
    expect(last_response.body).to include("Test User")
  end
end

# spec/actions/api/v1/users/create_spec.rb
RSpec.describe API::Actions::V1::Users::Create, type: :request do
  let(:valid_params) { { email: "new@example.com", name: "New", password: "Password1" } }

  it "creates with valid params" do
    post_json "/api/v1/users", valid_params

    expect(last_response.status).to eq(201)
    expect(json_response[:user][:email]).to eq("new@example.com")
  end

  it "rejects invalid params" do
    post_json "/api/v1/users", { email: "" }

    expect(last_response.status).to eq(422)
    expect(json_response[:errors]).to be_present
  end
end
```

### Repository Tests

```ruby
# spec/repositories/user_repo_spec.rb
RSpec.describe MyApp::Repositories::UserRepo do
  subject(:repo) { described_class.new }

  describe "#find_by_email" do
    it "returns user for matching email" do
      repo.create(email: "test@example.com", name: "Test",
                  password_digest: "hash", created_at: Time.now, updated_at: Time.now)

      user = repo.find_by_email("test@example.com")
      expect(user).to be_present
      expect(user.email).to eq("test@example.com")
    end

    it "returns nil for non-matching email" do
      expect(repo.find_by_email("none@example.com")).to be_nil
    end

    it "performs case-insensitive lookup" do
      repo.create(email: "test@example.com", name: "Test",
                  password_digest: "hash", created_at: Time.now, updated_at: Time.now)

      expect(repo.find_by_email("TEST@Example.COM")).to be_present
    end
  end

  describe "#all_paginated" do
    before do
      5.times do |i|
        repo.create(email: "user#{i}@example.com", name: "User #{i}",
                    password_digest: "hash", active: true,
                    created_at: Time.now, updated_at: Time.now)
      end
    end

    it "returns paginated results" do
      page1 = repo.all_paginated(page: 1, per_page: 2)
      page2 = repo.all_paginated(page: 2, per_page: 2)

      expect(page1.size).to eq(2)
      expect(page2.size).to eq(2)
      expect(page1).not_to eq(page2)
    end
  end
end
```

### Contract Tests

```ruby
# spec/contracts/user_contract_spec.rb
RSpec.describe MyApp::Contracts::UserContract do
  subject(:contract) { described_class.new }

  let(:valid_params) do
    { email: "test@example.com", name: "Test User", password: "Password1" }
  end

  it "succeeds with valid params" do
    result = contract.call(valid_params)
    expect(result).to be_success
  end

  it "fails without email" do
    result = contract.call(valid_params.except(:email))
    expect(result.errors[:email]).to include("is missing")
  end

  it "fails with weak password" do
    result = contract.call(valid_params.merge(password: "short"))
    expect(result.errors[:password]).to be_present
  end

  it "fails with invalid email format" do
    result = contract.call(valid_params.merge(email: "not-an-email"))
    expect(result.errors[:email]).to include("has invalid format")
  end
end
```

---

## Services

### Password Hasher

```ruby
# lib/myapp/services/password_hasher.rb
require "bcrypt"

module MyApp
  module Services
    class PasswordHasher
      COST = ENV.fetch("BCRYPT_COST", 12).to_i

      def hash(password)
        BCrypt::Password.create(password, cost: COST)
      end

      def verify(password, digest)
        BCrypt::Password.new(digest) == password
      rescue BCrypt::Errors::InvalidHash
        false
      end
    end
  end
end
```

### JWT Encoder

```ruby
# lib/myapp/services/jwt_encoder.rb
require "jwt"

module MyApp
  module Services
    class JWTEncoder
      include Deps["settings"]

      ALGORITHM = "HS256"
      DEFAULT_EXPIRATION = 24 * 60 * 60 # 24 hours

      def encode(payload, expiration: DEFAULT_EXPIRATION)
        payload[:exp] = (Time.now + expiration).to_i
        payload[:iat] = Time.now.to_i
        JWT.encode(payload, settings.jwt_secret, ALGORITHM)
      end

      def decode(token)
        JWT.decode(token, settings.jwt_secret, true, algorithm: ALGORITHM).first
      rescue JWT::DecodeError, JWT::ExpiredSignature
        nil
      end
    end
  end
end
```

---

## Asset Management

### Hanami Assets Setup

```ruby
# config/app.rb
module MyApp
  class App < Hanami::App
    config.assets.compile = true # Development only
  end
end
```

### Directory Structure

```
app/
├── assets/
│   ├── css/
│   │   ├── app.css           # Main stylesheet
│   │   └── components/       # Component styles
│   ├── js/
│   │   ├── app.js            # Main entry point
│   │   └── components/       # JS modules
│   └── images/
│       └── logo.svg
```

### Using Assets in Templates

```erb
<!-- app/templates/layouts/app.html.erb -->
<!DOCTYPE html>
<html>
<head>
  <%= stylesheet "app" %>
  <%= javascript "app" %>
  <%= favicon "favicon.ico" %>
</head>
<body>
  <%= yield %>
</body>
</html>
```

---

## Deployment

### Production Configuration

```ruby
# config/app.rb
module MyApp
  class App < Hanami::App
    environment(:production) do
      config.logger.level = :info
      config.logger.stream = "log/production.log"
      config.assets.compile = false
      config.assets.fingerprint = true
    end
  end
end
```

### Puma Configuration

```ruby
# config/puma.rb
workers ENV.fetch("WEB_CONCURRENCY", 2).to_i
threads_count = ENV.fetch("RAILS_MAX_THREADS", 5).to_i
threads threads_count, threads_count

preload_app!

port ENV.fetch("PORT", 3000)
environment ENV.fetch("HANAMI_ENV", "production")

on_worker_boot do
  # Reconnect database on worker fork
end
```

### Dockerfile

```dockerfile
FROM ruby:3.2-slim

RUN apt-get update -qq && apt-get install -y build-essential libpq-dev

WORKDIR /app

COPY Gemfile Gemfile.lock ./
RUN bundle config set --local deployment true && \
    bundle config set --local without 'development test' && \
    bundle install

COPY . .

# Precompile assets
RUN HANAMI_ENV=production bundle exec hanami assets compile

EXPOSE 3000
CMD ["bundle", "exec", "puma", "-C", "config/puma.rb"]
```

### Environment Variables (.env.example)

```bash
# Database
DATABASE_URL=postgres://user:pass@localhost:5432/myapp_production

# Security
SESSION_SECRET=generate-with-bundle-exec-hanami-generate-secret
JWT_SECRET=generate-a-secure-random-string

# Application
HANAMI_ENV=production
PORT=3000
WEB_CONCURRENCY=2
LOG_LEVEL=info

# Optional
REDIS_URL=redis://localhost:6379/0
```

### Health Check Endpoint

```ruby
# config/routes.rb -- Add before other routes
get "/health", to: ->(_, response) {
  response.status = 200
  response.body = { status: "ok", timestamp: Time.now.iso8601 }.to_json
}
```

---

## Middleware & Rack Integration

### Custom Middleware

```ruby
# lib/myapp/middleware/request_id.rb
module MyApp
  module Middleware
    class RequestId
      def initialize(app)
        @app = app
      end

      def call(env)
        env["X-Request-Id"] = SecureRandom.uuid
        status, headers, body = @app.call(env)
        headers["X-Request-Id"] = env["X-Request-Id"]
        [status, headers, body]
      end
    end
  end
end

# config/app.rb
config.middleware.use MyApp::Middleware::RequestId
```

### CORS for API Slices

```ruby
# config/app.rb
require "rack/cors"

config.middleware.use Rack::Cors do
  allow do
    origins ENV.fetch("CORS_ORIGINS", "http://localhost:3001")
    resource "/api/*",
      headers: :any,
      methods: [:get, :post, :put, :patch, :delete, :options],
      credentials: true,
      max_age: 600
  end
end
```

---

## Best Practices Summary

### Architecture Checklist

- [ ] Each slice represents a single bounded context
- [ ] Actions delegate to operations (no business logic in actions)
- [ ] Operations use dry-monads Result (Success/Failure)
- [ ] Operations use do notation for multi-step pipelines
- [ ] Repositories handle all data access (never bypass to relations)
- [ ] Services wrap external dependencies (email, storage, APIs)
- [ ] Providers register services in the DI container

### Testing Checklist

- [ ] Operations: unit tests with mocked dependencies
- [ ] Actions: request specs verifying HTTP status and response body
- [ ] Repositories: integration tests with real database
- [ ] Contracts: unit tests for all validation rules
- [ ] Coverage >80% for operations and repositories
- [ ] Database cleaned between tests (transaction strategy)
- [ ] Factories defined for all entities

### Security Checklist

- [ ] All inputs validated via action params or dry-validation contracts
- [ ] Passwords hashed with bcrypt (cost >= 12 in production)
- [ ] JWT tokens have expiration and are verified on every request
- [ ] CSP headers configured
- [ ] Secrets stored in environment variables, never in code
- [ ] SQL injection prevented by ROM parameterized queries
- [ ] CORS configured with explicit origins in production
