# Sinatra Patterns Reference

## Contents

- [Database Integration (ActiveRecord)](#database-integration-activerecord)
- [Database Integration (Sequel)](#database-integration-sequel)
- [Authentication: Sessions](#authentication-sessions)
- [Authentication: JWT](#authentication-jwt)
- [Rack Middleware](#rack-middleware)
- [Testing Patterns](#testing-patterns)
- [Deployment](#deployment)

## Database Integration (ActiveRecord)

### Database Configuration

```ruby
# config/database.rb
require 'active_record'
require 'yaml'
require 'erb'

def db_config
  config_file = File.join(__dir__, 'database.yml')
  YAML.safe_load(ERB.new(File.read(config_file)).result, aliases: true)
end

def setup_database
  env = ENV.fetch('RACK_ENV', 'development')
  ActiveRecord::Base.establish_connection(db_config[env])
  ActiveRecord::Base.logger = Logger.new($stdout) if env == 'development'
end

setup_database
```

```yaml
# config/database.yml
default: &default
  adapter: postgresql
  encoding: unicode
  pool: 5

development:
  <<: *default
  database: myapp_development

test:
  <<: *default
  database: myapp_test

production:
  <<: *default
  url: <%= ENV['DATABASE_URL'] %>
```

### Rake Tasks for Migrations

```ruby
# Rakefile
require 'bundler/setup'
require 'active_record'
require_relative 'config/database'

namespace :db do
  desc 'Create database'
  task :create do
    config = db_config[ENV.fetch('RACK_ENV', 'development')]
    ActiveRecord::Base.establish_connection(config.merge('database' => 'postgres'))
    ActiveRecord::Base.connection.create_database(config['database'])
    puts "Database created: #{config['database']}"
  end

  desc 'Drop database'
  task :drop do
    config = db_config[ENV.fetch('RACK_ENV', 'development')]
    ActiveRecord::Base.establish_connection(config.merge('database' => 'postgres'))
    ActiveRecord::Base.connection.drop_database(config['database'])
    puts "Database dropped: #{config['database']}"
  end

  desc 'Run migrations'
  task :migrate do
    setup_database
    ActiveRecord::MigrationContext.new('db/migrations').migrate
    puts 'Migrations complete'
  end

  desc 'Rollback migration'
  task :rollback do
    setup_database
    ActiveRecord::MigrationContext.new('db/migrations').rollback
    puts 'Rollback complete'
  end
end
```

### Migration Example

```ruby
# db/migrations/001_create_users.rb
class CreateUsers < ActiveRecord::Migration[7.0]
  def change
    create_table :users do |t|
      t.string :email, null: false
      t.string :name, null: false
      t.string :password_digest, null: false
      t.boolean :admin, default: false
      t.boolean :active, default: true
      t.timestamps
    end

    add_index :users, :email, unique: true
  end
end
```

### Model with Validations

```ruby
# app/models/user.rb
require 'active_record'
require 'bcrypt'

class User < ActiveRecord::Base
  has_secure_password

  has_many :posts, dependent: :destroy

  validates :email, presence: true,
                    uniqueness: { case_sensitive: false },
                    format: { with: URI::MailTo::EMAIL_REGEXP }
  validates :name, presence: true, length: { minimum: 2, maximum: 100 }
  validates :password, length: { minimum: 8 }, if: -> { password.present? }

  before_save :downcase_email

  scope :active, -> { where(active: true) }
  scope :admins, -> { where(admin: true) }

  def to_h
    {
      id: id,
      email: email,
      name: name,
      admin: admin,
      created_at: created_at.iso8601
    }
  end

  private

  def downcase_email
    self.email = email.downcase.strip
  end
end
```

## Database Integration (Sequel)

Sequel is a lightweight alternative to ActiveRecord, often preferred in Sinatra projects.

### Setup

```ruby
# config/database.rb
require 'sequel'

DB = Sequel.connect(
  ENV.fetch('DATABASE_URL', 'postgres://localhost/myapp_development'),
  max_connections: ENV.fetch('DB_POOL', 5).to_i,
  logger: ENV['RACK_ENV'] == 'development' ? Logger.new($stdout) : nil
)

# Enable Sequel model plugin
Sequel::Model.plugin :json_serializer
Sequel::Model.plugin :validation_helpers
Sequel::Model.plugin :timestamps, update_on_create: true
```

### Model

```ruby
# app/models/user.rb
class User < Sequel::Model
  plugin :secure_password

  one_to_many :posts

  def validate
    super
    validates_presence [:email, :name]
    validates_unique :email
    validates_format URI::MailTo::EMAIL_REGEXP, :email
    validates_min_length 8, :password if new? || password
  end

  def before_save
    self.email = email.downcase.strip
    super
  end

  def to_api
    { id: id, email: email, name: name, created_at: created_at.iso8601 }
  end
end
```

### Sequel Migration

```ruby
# db/migrations/001_create_users.rb
Sequel.migration do
  change do
    create_table(:users) do
      primary_key :id
      String :email, null: false, unique: true
      String :name, null: false
      String :password_digest, null: false
      TrueClass :admin, default: false
      DateTime :created_at
      DateTime :updated_at
    end
  end
end
```

Run migrations with: `bundle exec sequel -m db/migrations postgres://localhost/myapp_development`

## Authentication: Sessions

### Session-Based Auth Routes

```ruby
# app/routes/auth.rb
module Routes
  module Auth
    def self.registered(app)
      app.namespace '/api/v1/auth' do
        post '/login' do
          user = User.find_by(email: json_params[:email])

          if user&.authenticate(json_params[:password])
            session[:user_id] = user.id
            json user: user.to_h
          else
            halt 401, json(error: 'Invalid credentials')
          end
        end

        post '/logout' do
          session.clear
          json message: 'Logged out successfully'
        end

        get '/me' do
          require_login!
          json user: current_user.to_h
        end

        post '/register' do
          user = User.create!(
            email: json_params[:email],
            password: json_params[:password],
            name: json_params[:name]
          )
          session[:user_id] = user.id
          status 201
          json user: user.to_h
        rescue ActiveRecord::RecordInvalid => e
          status 422
          json errors: e.record.errors.full_messages
        end
      end
    end
  end
end
```

### Session Configuration

```ruby
configure do
  enable :sessions
  set :session_secret, ENV.fetch('SESSION_SECRET')
  set :sessions, {
    httponly: true,
    secure: production?,
    same_site: :lax,
    expire_after: 24 * 60 * 60  # 24 hours
  }
end
```

## Authentication: JWT

### JWT Helper Module

```ruby
# app/helpers/jwt_helper.rb
require 'jwt'

module JwtHelper
  JWT_SECRET = ENV.fetch('JWT_SECRET')
  JWT_ALGORITHM = 'HS256'
  TOKEN_EXPIRY = 24 * 3600  # 24 hours

  def generate_token(user)
    payload = {
      user_id: user.id,
      exp: Time.now.to_i + TOKEN_EXPIRY,
      iat: Time.now.to_i
    }
    JWT.encode(payload, JWT_SECRET, JWT_ALGORITHM)
  end

  def decode_token(token)
    JWT.decode(token, JWT_SECRET, true, algorithm: JWT_ALGORITHM).first
  rescue JWT::ExpiredSignature
    halt 401, json(error: 'Token expired')
  rescue JWT::DecodeError
    halt 401, json(error: 'Invalid token')
  end

  def authenticate_token!
    header = request.env['HTTP_AUTHORIZATION']
    halt 401, json(error: 'Missing Authorization header') unless header

    token = header.sub(/^Bearer\s+/, '')
    payload = decode_token(token)

    @current_user = User.find_by(id: payload['user_id'])
    halt 401, json(error: 'User not found') unless @current_user
  end
end
```

### JWT Auth Filter

```ruby
class App < Sinatra::Base
  helpers JwtHelper

  # Protect API routes with JWT
  before '/api/v1/*' do
    pass if request.path_info =~ %r{/api/v1/auth/(login|register)}
    authenticate_token!
  end
end
```

## Rack Middleware

### Request Logger

```ruby
# app/middleware/request_logger.rb
class RequestLogger
  def initialize(app)
    @app = app
    @logger = Logger.new($stdout)
  end

  def call(env)
    start_time = Time.now
    status, headers, response = @app.call(env)
    duration = ((Time.now - start_time) * 1000).round(2)

    @logger.info(
      "#{env['REQUEST_METHOD']} #{env['PATH_INFO']} " \
      "#{status} #{duration}ms"
    )

    [status, headers, response]
  end
end
```

### CORS Middleware

```ruby
# app/middleware/cors.rb
class CORS
  ALLOWED_METHODS = 'GET, POST, PUT, PATCH, DELETE, OPTIONS'
  ALLOWED_HEADERS = 'Content-Type, Authorization'

  def initialize(app, origins: '*')
    @app = app
    @origins = origins
  end

  def call(env)
    if env['REQUEST_METHOD'] == 'OPTIONS'
      [204, cors_headers, []]
    else
      status, headers, response = @app.call(env)
      [status, headers.merge(cors_headers), response]
    end
  end

  private

  def cors_headers
    {
      'Access-Control-Allow-Origin' => @origins,
      'Access-Control-Allow-Methods' => ALLOWED_METHODS,
      'Access-Control-Allow-Headers' => ALLOWED_HEADERS,
      'Access-Control-Max-Age' => '86400'
    }
  end
end
```

### Rate Limiter

```ruby
# app/middleware/rate_limiter.rb
class RateLimiter
  def initialize(app, requests_per_minute: 60)
    @app = app
    @limit = requests_per_minute
    @store = Hash.new { |h, k| h[k] = [] }
  end

  def call(env)
    client_ip = env['REMOTE_ADDR']
    now = Time.now.to_i
    window = now - 60

    @store[client_ip].reject! { |t| t < window }

    if @store[client_ip].size >= @limit
      [429, { 'Content-Type' => 'application/json' },
       ['{"error":"Rate limit exceeded"}']]
    else
      @store[client_ip] << now
      @app.call(env)
    end
  end
end
```

### Registering Middleware in config.ru

```ruby
# config.ru
require 'bundler/setup'
Bundler.require(:default, ENV.fetch('RACK_ENV', 'development'))

require_relative 'app/middleware/request_logger'
require_relative 'app/middleware/cors'
require_relative 'app/middleware/rate_limiter'
require_relative 'app/main'

use RequestLogger
use CORS, origins: ENV.fetch('CORS_ORIGINS', '*')
use RateLimiter, requests_per_minute: 100

run App
```

## Testing Patterns

### RSpec Configuration

```ruby
# spec/spec_helper.rb
ENV['RACK_ENV'] = 'test'

require 'bundler/setup'
Bundler.require(:default, :test)

require 'rack/test'
require 'database_cleaner/active_record'

require_relative '../config/database'
require_relative '../app/main'

RSpec.configure do |config|
  config.include Rack::Test::Methods

  config.before(:suite) do
    DatabaseCleaner.strategy = :transaction
    DatabaseCleaner.clean_with(:truncation)
  end

  config.around(:each) do |example|
    DatabaseCleaner.cleaning { example.run }
  end

  def app
    App
  end

  def json_response
    JSON.parse(last_response.body, symbolize_names: true)
  end

  def post_json(path, body = {})
    post path, body.to_json, { 'CONTENT_TYPE' => 'application/json' }
  end

  def put_json(path, body = {})
    put path, body.to_json, { 'CONTENT_TYPE' => 'application/json' }
  end

  def auth_header(user)
    token = App.new.helpers.generate_token(user)
    { 'HTTP_AUTHORIZATION' => "Bearer #{token}" }
  end
end
```

### Route Tests

```ruby
# spec/routes/users_spec.rb
require 'spec_helper'

RSpec.describe 'Users API' do
  let!(:user) do
    User.create!(email: 'test@example.com', name: 'Test', password: 'password123')
  end

  describe 'GET /api/v1/users' do
    it 'returns all users' do
      get '/api/v1/users'

      expect(last_response.status).to eq(200)
      expect(json_response[:users]).to be_an(Array)
      expect(json_response[:users].length).to eq(1)
    end
  end

  describe 'GET /api/v1/users/:id' do
    it 'returns a user by id' do
      get "/api/v1/users/#{user.id}"

      expect(last_response.status).to eq(200)
      expect(json_response[:user][:email]).to eq('test@example.com')
    end

    it 'returns 404 for non-existent user' do
      get '/api/v1/users/999'
      expect(last_response.status).to eq(404)
    end
  end

  describe 'POST /api/v1/users' do
    let(:valid_params) do
      { email: 'new@example.com', name: 'New User', password: 'password123' }
    end

    it 'creates a user with valid params' do
      post_json '/api/v1/users', valid_params

      expect(last_response.status).to eq(201)
      expect(json_response[:user][:email]).to eq('new@example.com')
    end

    it 'returns 422 with invalid params' do
      post_json '/api/v1/users', { email: 'invalid' }

      expect(last_response.status).to eq(422)
      expect(json_response[:errors]).to be_present
    end
  end
end
```

### Auth Tests

```ruby
# spec/routes/auth_spec.rb
require 'spec_helper'

RSpec.describe 'Auth API' do
  let!(:user) do
    User.create!(email: 'test@example.com', name: 'Test', password: 'password123')
  end

  describe 'POST /api/v1/auth/login' do
    it 'returns user with valid credentials' do
      post_json '/api/v1/auth/login',
                { email: 'test@example.com', password: 'password123' }

      expect(last_response.status).to eq(200)
      expect(json_response[:user]).to be_present
    end

    it 'returns 401 with invalid credentials' do
      post_json '/api/v1/auth/login',
                { email: 'test@example.com', password: 'wrong' }

      expect(last_response.status).to eq(401)
    end
  end

  describe 'GET /api/v1/auth/me' do
    it 'returns 401 when not authenticated' do
      get '/api/v1/auth/me'
      expect(last_response.status).to eq(401)
    end
  end
end
```

### Testing Middleware

```ruby
# spec/middleware/cors_spec.rb
require 'spec_helper'

RSpec.describe CORS do
  let(:inner_app) { ->(_env) { [200, {}, ['OK']] } }
  let(:app) { CORS.new(inner_app, origins: 'https://example.com') }

  it 'adds CORS headers to responses' do
    get '/'

    expect(last_response.headers['Access-Control-Allow-Origin'])
      .to eq('https://example.com')
  end

  it 'responds to OPTIONS preflight with 204' do
    options '/'

    expect(last_response.status).to eq(204)
    expect(last_response.headers['Access-Control-Allow-Methods'])
      .to include('GET')
  end
end
```

## Deployment

### Gemfile

```ruby
source 'https://rubygems.org'

ruby '3.2.2'

# Framework
gem 'sinatra', '~> 3.0'
gem 'sinatra-contrib'
gem 'puma', '~> 6.0'

# Database
gem 'activerecord', '~> 7.0'
gem 'pg', '~> 1.5'
gem 'bcrypt', '~> 3.1'

# Auth & Utils
gem 'jwt', '~> 2.7'
gem 'rake', '~> 13.0'
gem 'dotenv', '~> 2.8'

group :development do
  gem 'pry'
end

group :test do
  gem 'rspec', '~> 3.12'
  gem 'rack-test'
  gem 'database_cleaner-active_record'
  gem 'factory_bot'
  gem 'faker'
  gem 'simplecov', require: false
end
```

### Puma Configuration

```ruby
# config/puma.rb
workers ENV.fetch('WEB_CONCURRENCY', 2)
threads_count = ENV.fetch('MAX_THREADS', 5)
threads threads_count, threads_count

preload_app!

port ENV.fetch('PORT', 4567)
environment ENV.fetch('RACK_ENV', 'development')

on_worker_boot do
  ActiveRecord::Base.establish_connection
end
```

### Environment Boot File

```ruby
# config/environment.rb
require 'bundler/setup'
require 'dotenv'

Dotenv.load(".env.#{ENV.fetch('RACK_ENV', 'development')}", '.env')

Bundler.require(:default, ENV.fetch('RACK_ENV', 'development'))

require_relative 'database'

Dir[File.join(__dir__, '..', 'app', 'models', '*.rb')].each { |f| require f }
Dir[File.join(__dir__, '..', 'app', 'services', '*.rb')].each { |f| require f }
```

### Procfile (Heroku)

```
web: bundle exec puma -C config/puma.rb
release: bundle exec rake db:migrate
```

### Dockerfile

```dockerfile
FROM ruby:3.2-slim

RUN apt-get update && apt-get install -y \
    build-essential libpq-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY Gemfile Gemfile.lock ./
RUN bundle install --without development test

COPY . .

EXPOSE 4567

CMD ["bundle", "exec", "puma", "-C", "config/puma.rb"]
```

### docker-compose.yml

```yaml
version: '3.8'

services:
  web:
    build: .
    ports:
      - "4567:4567"
    environment:
      - DATABASE_URL=postgres://postgres:postgres@db:5432/myapp
      - RACK_ENV=production
      - SESSION_SECRET=${SESSION_SECRET}
      - JWT_SECRET=${JWT_SECRET}
    depends_on:
      - db

  db:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: myapp
      POSTGRES_PASSWORD: postgres
    volumes:
      - pgdata:/var/lib/postgresql/data

volumes:
  pgdata:
```

### .env.example

```bash
RACK_ENV=development
PORT=4567
DATABASE_URL=postgres://localhost/myapp_development
SESSION_SECRET=change-me-in-production
JWT_SECRET=change-me-in-production
CORS_ORIGINS=http://localhost:3000
```

### Service Object Pattern

```ruby
# app/services/user_service.rb
class UserService
  class << self
    def create(params)
      user = User.new(
        email: params[:email],
        name: params[:name],
        password: params[:password]
      )

      if user.save
        { success: true, user: user }
      else
        { success: false, errors: user.errors.full_messages }
      end
    end

    def update(user, params)
      allowed = params.slice(:name, :email)

      if user.update(allowed)
        { success: true, user: user }
      else
        { success: false, errors: user.errors.full_messages }
      end
    end

    def authenticate(email, password)
      user = User.find_by(email: email&.downcase&.strip)
      return nil unless user&.authenticate(password)

      user
    end
  end
end
```

### Using Service Objects in Routes

```ruby
# app/routes/users.rb
module Routes
  module Users
    def self.registered(app)
      app.namespace '/api/v1/users' do
        post do
          result = UserService.create(json_params)

          if result[:success]
            status 201
            json user: result[:user].to_h
          else
            status 422
            json errors: result[:errors]
          end
        end
      end
    end
  end
end
```
