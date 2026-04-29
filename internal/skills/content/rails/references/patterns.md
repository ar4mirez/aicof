# Rails Advanced Patterns

Detailed patterns for ActiveRecord, Hotwire/Turbo, Action Cable, ActiveJob, API mode, testing, and deployment.

---

## ActiveRecord Advanced Patterns

### Associations with Options

```ruby
class User < ApplicationRecord
  has_many :posts, dependent: :destroy
  has_many :published_posts, -> { where(status: :published) }, class_name: "Post"
  has_many :comments, dependent: :destroy
  has_one :profile, dependent: :destroy
  has_and_belongs_to_many :roles

  # Polymorphic
  has_many :reactions, as: :reactable, dependent: :destroy

  # Through
  has_many :taggings, through: :posts
  has_many :tags, through: :taggings

  # Secure password
  has_secure_password

  # Enum with prefix
  enum :role, { user: "user", admin: "admin", moderator: "moderator" }, prefix: true

  def self.find_by_credentials(email, password)
    user = find_by(email: email.downcase)
    user&.authenticate(password) ? user : nil
  end
end
```

### Self-Referential Associations

```ruby
class Comment < ApplicationRecord
  belongs_to :user
  belongs_to :post, counter_cache: true

  belongs_to :parent, class_name: "Comment", optional: true
  has_many :replies, class_name: "Comment", foreign_key: :parent_id, dependent: :destroy

  validates :body, presence: true, length: { maximum: 1000 }

  scope :root_comments, -> { where(parent_id: nil) }
end
```

### Concerns

```ruby
# app/models/concerns/searchable.rb
module Searchable
  extend ActiveSupport::Concern

  included do
    scope :search, ->(query) {
      return all if query.blank?
      where("#{table_name}.title ILIKE :q OR #{table_name}.body ILIKE :q",
            q: "%#{sanitize_sql_like(query)}%")
    }
  end
end

# app/models/concerns/sluggable.rb
module Sluggable
  extend ActiveSupport::Concern

  included do
    before_validation :generate_slug, if: -> { slug.blank? && respond_to?(:title) }
    validates :slug, presence: true, uniqueness: true
  end

  private

  def generate_slug
    self.slug = title&.parameterize
  end
end

# Usage
class Post < ApplicationRecord
  include Searchable
  include Sluggable
end
```

### Query Objects

```ruby
# app/queries/posts_query.rb
class PostsQuery
  def initialize(relation = Post.all)
    @relation = relation
  end

  def call(params = {})
    result = @relation
    result = filter_by_status(result, params[:status])
    result = filter_by_category(result, params[:category_id])
    result = filter_by_date_range(result, params[:start_date], params[:end_date])
    result = search(result, params[:q])
    sort(result, params[:sort], params[:direction])
  end

  private

  def filter_by_status(relation, status)
    return relation if status.blank?
    case status.to_s
    when "published" then relation.where(published: true)
    when "draft" then relation.where(published: false)
    else relation
    end
  end

  def filter_by_category(relation, category_id)
    return relation if category_id.blank?
    relation.where(category_id: category_id)
  end

  def filter_by_date_range(relation, start_date, end_date)
    relation = relation.where("created_at >= ?", start_date) if start_date.present?
    relation = relation.where("created_at <= ?", end_date) if end_date.present?
    relation
  end

  def search(relation, query)
    return relation if query.blank?
    relation.where("title ILIKE ? OR body ILIKE ?", "%#{query}%", "%#{query}%")
  end

  def sort(relation, sort_by, direction)
    sort_by = %w[created_at title].include?(sort_by) ? sort_by : "created_at"
    direction = %w[asc desc].include?(direction) ? direction : "desc"
    relation.order(sort_by => direction)
  end
end

# Usage in controller
def index
  @posts = PostsQuery.new.call(params.permit(:status, :category_id, :q, :sort, :direction))
                         .page(params[:page]).per(20)
end
```

---

## Service Objects

### Result Object

```ruby
# app/services/service_result.rb
class ServiceResult
  attr_reader :data, :errors

  def initialize(success:, data: nil, errors: [])
    @success = success
    @data = data
    @errors = errors
  end

  def self.success(data = nil)
    new(success: true, data: data)
  end

  def self.failure(errors)
    new(success: false, errors: Array(errors))
  end

  def success?
    @success
  end

  def failure?
    !@success
  end
end
```

### Complex Service Example

```ruby
# app/services/users/create_service.rb
module Users
  class CreateService < ApplicationService
    def initialize(params:, created_by: nil)
      @params = params
      @created_by = created_by
    end

    def call
      user = User.new(user_params)

      ActiveRecord::Base.transaction do
        user.save!
        create_profile!(user)
        assign_default_role!(user)
        send_notifications(user)
      end

      ServiceResult.success(user)
    rescue ActiveRecord::RecordInvalid => e
      ServiceResult.failure(e.record.errors.full_messages)
    rescue StandardError => e
      Rails.logger.error("User creation failed: #{e.message}")
      ServiceResult.failure(["An unexpected error occurred"])
    end

    private

    attr_reader :params, :created_by

    def user_params
      params.slice(:email, :name, :password, :password_confirmation)
    end

    def create_profile!(user)
      user.create_profile!(bio: params[:bio], avatar_url: params[:avatar_url])
    end

    def assign_default_role!(user)
      default_role = Role.find_by!(name: "user")
      user.roles << default_role
    end

    def send_notifications(user)
      UserMailer.welcome(user).deliver_later
      AdminMailer.new_user(user, created_by).deliver_later if created_by
    end
  end
end

# Usage in controller
def create
  result = Users::CreateService.call(params: user_params, created_by: current_user)
  if result.success?
    redirect_to result.data, notice: "User created."
  else
    @errors = result.errors
    render :new, status: :unprocessable_entity
  end
end
```

---

## Hotwire / Turbo

### Turbo Frames

Turbo Frames allow parts of a page to be updated independently without full page reloads.

```erb
<%# app/views/posts/index.html.erb %>
<%= turbo_frame_tag "posts" do %>
  <% @posts.each do |post| %>
    <%= render post %>
  <% end %>
  <%= paginate @posts %>
<% end %>

<%# app/views/posts/_post.html.erb %>
<%= turbo_frame_tag dom_id(post) do %>
  <article>
    <h2><%= link_to post.title, post %></h2>
    <p><%= truncate(post.body, length: 200) %></p>
    <%= link_to "Edit", edit_post_path(post) %>
  </article>
<% end %>
```

### Turbo Streams

Turbo Streams update specific parts of the DOM in response to form submissions or WebSocket broadcasts.

```ruby
# app/controllers/comments_controller.rb
def create
  @comment = @post.comments.build(comment_params)
  @comment.user = current_user

  respond_to do |format|
    if @comment.save
      format.turbo_stream
      format.html { redirect_to @post }
    else
      format.html { render "posts/show", status: :unprocessable_entity }
    end
  end
end
```

```erb
<%# app/views/comments/create.turbo_stream.erb %>
<%= turbo_stream.append "comments" do %>
  <%= render @comment %>
<% end %>

<%= turbo_stream.update "comment_form" do %>
  <%= render "comments/form", post: @comment.post, comment: Comment.new %>
<% end %>

<%= turbo_stream.update "comments_count" do %>
  <%= @comment.post.comments.count %> comments
<% end %>
```

### Turbo Stream Actions

Available actions: `append`, `prepend`, `replace`, `update`, `remove`, `before`, `after`.

```ruby
# Broadcasting from model (real-time updates)
class Comment < ApplicationRecord
  after_create_commit -> {
    broadcast_append_to post, target: "comments", partial: "comments/comment"
  }
  after_destroy_commit -> {
    broadcast_remove_to post
  }
end
```

### Stimulus Controllers

```javascript
// app/javascript/controllers/toggle_controller.js
import { Controller } from "@hotwired/stimulus"

export default class extends Controller {
  static targets = ["content"]
  static values = { open: { type: Boolean, default: false } }

  toggle() {
    this.openValue = !this.openValue
  }

  openValueChanged() {
    this.contentTarget.classList.toggle("hidden", !this.openValue)
  }
}
```

```erb
<div data-controller="toggle">
  <button data-action="toggle#toggle">Toggle</button>
  <div data-toggle-target="content" class="hidden">
    Content here
  </div>
</div>
```

---

## Action Cable

### Channel Setup

```ruby
# app/channels/application_cable/connection.rb
module ApplicationCable
  class Connection < ActionCable::Connection::Base
    identified_by :current_user

    def connect
      self.current_user = find_verified_user
    end

    private

    def find_verified_user
      user = User.find_by(id: cookies.encrypted[:user_id])
      user || reject_unauthorized_connection
    end
  end
end

# app/channels/chat_channel.rb
class ChatChannel < ApplicationCable::Channel
  def subscribed
    @room = Room.find(params[:room_id])
    stream_for @room
  end

  def receive(data)
    message = @room.messages.create!(
      user: current_user,
      body: data["body"]
    )
    ChatChannel.broadcast_to(@room, {
      message: render_message(message)
    })
  end

  private

  def render_message(message)
    ApplicationController.renderer.render(
      partial: "messages/message",
      locals: { message: message }
    )
  end
end
```

### Client-Side Subscription

```javascript
// app/javascript/channels/chat_channel.js
import consumer from "./consumer"

const chatChannel = consumer.subscriptions.create(
  { channel: "ChatChannel", room_id: roomId },
  {
    received(data) {
      const messages = document.getElementById("messages")
      messages.insertAdjacentHTML("beforeend", data.message)
    },

    send(body) {
      this.perform("receive", { body })
    }
  }
)
```

---

## ActiveJob and Background Processing

### Job Definition

```ruby
# app/jobs/application_job.rb
class ApplicationJob < ActiveJob::Base
  retry_on ActiveRecord::Deadlocked, wait: 5.seconds, attempts: 3
  retry_on Net::OpenTimeout, wait: :polynomially_longer, attempts: 5
  discard_on ActiveJob::DeserializationError
end

# app/jobs/process_image_job.rb
class ProcessImageJob < ApplicationJob
  queue_as :default

  def perform(post_id)
    post = Post.find(post_id)
    return unless post.featured_image.attached?

    post.featured_image.variant(resize_to_limit: [800, 600]).processed
    post.featured_image.variant(resize_to_limit: [400, 300]).processed
  end
end

# app/jobs/send_weekly_digest_job.rb
class SendWeeklyDigestJob < ApplicationJob
  queue_as :mailers

  def perform
    User.active.find_each do |user|
      posts = Post.published
                  .where("published_at > ?", 1.week.ago)
                  .order(published_at: :desc)
                  .limit(10)

      next if posts.empty?
      DigestMailer.weekly(user, posts).deliver_now
    end
  end
end
```

### Sidekiq Configuration

```yaml
# config/sidekiq.yml
:concurrency: 5
:queues:
  - [critical, 3]
  - [default, 2]
  - [mailers, 1]
  - [low, 1]
```

```ruby
# config/initializers/sidekiq.rb
Sidekiq.configure_server do |config|
  config.redis = { url: ENV.fetch("REDIS_URL", "redis://localhost:6379/1") }
end

Sidekiq.configure_client do |config|
  config.redis = { url: ENV.fetch("REDIS_URL", "redis://localhost:6379/1") }
end
```

### Scheduling Recurring Jobs

```ruby
# Using sidekiq-cron or solid_queue (Rails 7.1+)
# config/recurring.yml (solid_queue)
weekly_digest:
  class: SendWeeklyDigestJob
  schedule: every Monday at 9am
  queue: mailers
```

---

## API Mode

### API Base Controller

```ruby
# app/controllers/api/v1/base_controller.rb
module Api
  module V1
    class BaseController < ActionController::API
      include ActionController::HttpAuthentication::Token::ControllerMethods

      before_action :authenticate_api_user!

      rescue_from ActiveRecord::RecordNotFound, with: :not_found
      rescue_from ActiveRecord::RecordInvalid, with: :unprocessable_entity
      rescue_from ActionController::ParameterMissing, with: :bad_request

      private

      def authenticate_api_user!
        authenticate_or_request_with_http_token do |token, _options|
          @current_user = User.find_by(api_token: token)
        end
      end

      def current_user
        @current_user
      end

      def not_found(exception)
        render json: { error: exception.message }, status: :not_found
      end

      def unprocessable_entity(exception)
        render json: { errors: exception.record.errors.full_messages },
               status: :unprocessable_entity
      end

      def bad_request(exception)
        render json: { error: exception.message }, status: :bad_request
      end
    end
  end
end
```

### API Resource Controller

```ruby
# app/controllers/api/v1/posts_controller.rb
module Api
  module V1
    class PostsController < BaseController
      before_action :set_post, only: [:show, :update, :destroy]

      def index
        posts = Post.published
                    .includes(:user)
                    .order(created_at: :desc)
                    .page(params[:page])
                    .per(params[:per_page] || 20)

        render json: {
          posts: posts.map { |p| post_json(p) },
          meta: pagination_meta(posts)
        }
      end

      def show
        render json: post_json(@post, include_body: true)
      end

      def create
        post = current_user.posts.create!(post_params)
        render json: post_json(post), status: :created
      end

      def update
        @post.update!(post_params)
        render json: post_json(@post)
      end

      def destroy
        @post.destroy!
        head :no_content
      end

      private

      def set_post
        @post = Post.find(params[:id])
      end

      def post_params
        params.require(:post).permit(:title, :body, :category_id, :published)
      end

      def post_json(post, include_body: false)
        json = {
          id: post.id, title: post.title, published: post.published,
          created_at: post.created_at,
          user: { id: post.user.id, name: post.user.name }
        }
        json[:body] = post.body if include_body
        json
      end

      def pagination_meta(collection)
        {
          current_page: collection.current_page,
          total_pages: collection.total_pages,
          total_count: collection.total_count
        }
      end
    end
  end
end
```

### JSON Serialization Alternatives

For more complex APIs, consider using serialization libraries:

```ruby
# Using jbuilder
# app/views/api/v1/posts/show.json.jbuilder
json.post do
  json.extract! @post, :id, :title, :body, :published, :created_at
  json.user do
    json.extract! @post.user, :id, :name
  end
end

# Using ActiveModelSerializers or Blueprinter
# app/serializers/post_serializer.rb
class PostSerializer < ActiveModel::Serializer
  attributes :id, :title, :published, :created_at
  belongs_to :user
end
```

---

## Controller Concerns

### Authentication

```ruby
# app/controllers/concerns/authenticatable.rb
module Authenticatable
  extend ActiveSupport::Concern

  included do
    helper_method :current_user, :user_signed_in?
  end

  def current_user
    @current_user ||= User.find_by(id: session[:user_id]) if session[:user_id]
  end

  def user_signed_in?
    current_user.present?
  end

  def authenticate_user!
    unless user_signed_in?
      store_location
      redirect_to login_path, alert: "Please sign in to continue."
    end
  end

  def store_location
    session[:return_to] = request.fullpath if request.get?
  end

  def redirect_back_or(default)
    redirect_to(session.delete(:return_to) || default)
  end
end
```

### Pagination Helper

```ruby
# app/controllers/concerns/paginatable.rb
module Paginatable
  extend ActiveSupport::Concern

  private

  def page_number
    [params[:page].to_i, 1].max
  end

  def per_page
    [(params[:per_page] || 20).to_i, 100].min
  end
end
```

---

## Testing

### Controller Tests

```ruby
# test/controllers/posts_controller_test.rb
require "test_helper"

class PostsControllerTest < ActionDispatch::IntegrationTest
  setup do
    @user = users(:john)
    @post = posts(:first_post)
  end

  test "should get index" do
    get posts_url
    assert_response :success
  end

  test "should create post when authenticated" do
    sign_in @user
    assert_difference("Post.count") do
      post posts_url, params: { post: { title: "New", body: "Content" } }
    end
    assert_redirected_to post_url(Post.last)
  end

  test "should redirect to login when not authenticated" do
    get new_post_url
    assert_redirected_to login_url
  end

  test "should not update another user's post" do
    other_user = users(:jane)
    sign_in other_user
    patch post_url(@post), params: { post: { title: "Hacked" } }
    assert_redirected_to posts_url
    assert_equal "Not authorized.", flash[:alert]
  end

  private

  def sign_in(user)
    post login_url, params: { email: user.email, password: "password" }
  end
end
```

### System Tests

```ruby
# test/system/posts_test.rb
require "application_system_test_case"

class PostsTest < ApplicationSystemTestCase
  setup do
    @user = users(:john)
    @post = posts(:first_post)
  end

  test "visiting the index" do
    visit posts_url
    assert_selector "h1", text: "Posts"
  end

  test "creating a post" do
    sign_in @user
    visit new_post_url

    fill_in "Title", with: "New Post Title"
    fill_in "Body", with: "This is the post content."
    select "Technology", from: "Category"
    click_on "Create Post"

    assert_text "Post created."
    assert_text "New Post Title"
  end

  private

  def sign_in(user)
    visit login_url
    fill_in "Email", with: user.email
    fill_in "Password", with: "password"
    click_on "Sign In"
    assert_text "Signed in successfully"
  end
end
```

### Service Tests

```ruby
# test/services/posts/publish_service_test.rb
require "test_helper"

class Posts::PublishServiceTest < ActiveSupport::TestCase
  setup do
    @user = users(:john)
    @post = posts(:draft_post)
    @post.update!(user: @user)
  end

  test "publishes post successfully" do
    result = Posts::PublishService.call(post: @post, user: @user)

    assert result.success?
    assert @post.reload.published?
    assert_not_nil @post.published_at
  end

  test "fails when user is not the author" do
    other_user = users(:jane)
    result = Posts::PublishService.call(post: @post, user: other_user)

    assert result.failure?
    assert_includes result.errors, "Not authorized"
    assert_not @post.reload.published?
  end
end
```

### API Tests

```ruby
# test/controllers/api/v1/posts_controller_test.rb
require "test_helper"

class Api::V1::PostsControllerTest < ActionDispatch::IntegrationTest
  setup do
    @user = users(:john)
    @token = @user.api_token
  end

  test "returns paginated posts" do
    get api_v1_posts_url,
        headers: { "Authorization" => "Bearer #{@token}" }

    assert_response :success
    json = JSON.parse(response.body)
    assert json.key?("posts")
    assert json.key?("meta")
  end

  test "creates post with valid params" do
    assert_difference("Post.count") do
      post api_v1_posts_url,
           params: { post: { title: "API Post", body: "Content" } },
           headers: { "Authorization" => "Bearer #{@token}" },
           as: :json
    end

    assert_response :created
  end

  test "returns 401 without token" do
    get api_v1_posts_url
    assert_response :unauthorized
  end
end
```

---

## Caching

### Fragment Caching

```erb
<%# Russian doll caching %>
<% cache @post do %>
  <article>
    <h2><%= @post.title %></h2>
    <% cache @post.user do %>
      <p>By <%= @post.user.name %></p>
    <% end %>
    <p><%= @post.body %></p>
  </article>
<% end %>

<%# Collection caching %>
<%= render partial: "post", collection: @posts, cached: true %>
```

### Low-Level Caching

```ruby
Rails.cache.fetch("user_#{user.id}_posts_count", expires_in: 1.hour) do
  user.posts.count
end

# Cache with key based on record
Rails.cache.fetch([@post, "comments_count"], expires_in: 30.minutes) do
  @post.comments.count
end
```

---

## Error Handling

### Custom Error Pages

```ruby
# config/application.rb
config.exceptions_app = routes

# config/routes.rb
match "/404", to: "errors#not_found", via: :all
match "/500", to: "errors#internal_server_error", via: :all

# app/controllers/errors_controller.rb
class ErrorsController < ApplicationController
  def not_found
    render status: :not_found
  end

  def internal_server_error
    render status: :internal_server_error
  end
end
```

---

## Deployment

### Production Configuration

```ruby
# config/environments/production.rb
Rails.application.configure do
  config.force_ssl = true
  config.log_level = :info
  config.log_tags = [:request_id]

  # Asset delivery
  config.public_file_server.enabled = ENV["RAILS_SERVE_STATIC_FILES"].present?
  config.assets.compile = false

  # Caching
  config.cache_store = :redis_cache_store, { url: ENV["REDIS_URL"] }
  config.action_controller.perform_caching = true

  # ActiveJob
  config.active_job.queue_adapter = :sidekiq

  # Mailer
  config.action_mailer.delivery_method = :smtp
  config.action_mailer.default_url_options = { host: ENV["APP_HOST"] }
end
```

### Database Configuration

```yaml
# config/database.yml
production:
  adapter: postgresql
  encoding: unicode
  pool: <%= ENV.fetch("RAILS_MAX_THREADS") { 5 } %>
  url: <%= ENV["DATABASE_URL"] %>
```

### Dockerfile

```dockerfile
FROM ruby:3.2-slim

RUN apt-get update -qq && apt-get install -y \
  build-essential libpq-dev nodejs

WORKDIR /app

COPY Gemfile Gemfile.lock ./
RUN bundle config set --local deployment true && \
    bundle config set --local without 'development test' && \
    bundle install

COPY . .

RUN SECRET_KEY_BASE=placeholder rails assets:precompile

EXPOSE 3000

CMD ["bundle", "exec", "puma", "-C", "config/puma.rb"]
```

### Health Check

```ruby
# app/controllers/health_controller.rb
class HealthController < ApplicationController
  skip_before_action :authenticate_user!

  def show
    ActiveRecord::Base.connection.execute("SELECT 1")
    Redis.current.ping if defined?(Redis)

    render json: { status: "ok", timestamp: Time.current.iso8601 }
  rescue StandardError => e
    render json: { status: "error", message: e.message }, status: :service_unavailable
  end
end
```

---

## Gemfile Recommendations

```ruby
source "https://rubygems.org"
ruby "3.2.2"

# Core
gem "rails", "~> 7.1"
gem "pg", "~> 1.5"
gem "puma", "~> 6.0"
gem "redis", "~> 5.0"

# Frontend
gem "sprockets-rails"
gem "importmap-rails"
gem "turbo-rails"
gem "stimulus-rails"
gem "tailwindcss-rails"

# Authentication
gem "bcrypt", "~> 3.1"

# Background jobs
gem "sidekiq", "~> 7.0"

# Pagination
gem "kaminari", "~> 1.2"

# JSON
gem "jbuilder"

# Performance
gem "bootsnap", require: false

group :development, :test do
  gem "debug"
  gem "rspec-rails", "~> 6.0"     # Optional: use instead of Minitest
  gem "factory_bot_rails"
  gem "faker"
end

group :development do
  gem "web-console"
  gem "rack-mini-profiler"
end

group :test do
  gem "capybara"
  gem "selenium-webdriver"
  gem "shoulda-matchers"
  gem "webmock"
  gem "simplecov", require: false
end
```
