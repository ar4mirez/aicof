# Ruby Patterns Reference

## Contents

- [Enumerable Patterns](#enumerable-patterns)
- [Metaprogramming](#metaprogramming)
- [RSpec Matchers & Helpers](#rspec-matchers--helpers)

## Enumerable Patterns

### Transformation & Filtering

```ruby
# filter_map to transform and compact in one pass (Ruby 2.7+)
users.filter_map { |u| u.email if u.active? }

# tally for counting occurrences (Ruby 2.7+)
%w[apple banana apple cherry banana apple].tally
# => {"apple"=>3, "banana"=>2, "cherry"=>1}

# flat_map to flatten one level
teams.flat_map(&:members)

# group_by for categorization
orders.group_by(&:status)
# => { pending: [...], completed: [...] }

# chunk_while for grouping consecutive elements
[1, 1, 2, 2, 3, 1, 1].chunk_while { |a, b| a == b }.to_a
# => [[1, 1], [2, 2], [3], [1, 1]]
```

### Lazy Evaluation

```ruby
# Process large files without loading into memory
File.open("access.log").each_line.lazy
  .select { |line| line.match?(/ERROR/) }
  .map { |line| parse_log_entry(line) }
  .take(100)
  .to_a

# Infinite sequences
primes = (2..Float::INFINITY).lazy.select { |n| prime?(n) }
primes.first(20)

# Batched processing
large_dataset.lazy
  .select(&:valid?)
  .map(&:transform)
  .each_slice(100) { |batch| bulk_insert(batch) }
```

## Metaprogramming

### Safe define_method with Validation

```ruby
class StrictStruct
  def self.attribute(name, type:)
    define_method(name) { instance_variable_get(:"@#{name}") }

    define_method(:"#{name}=") do |value|
      raise TypeError, "#{name} must be #{type}, got #{value.class}" unless value.is_a?(type)

      instance_variable_set(:"@#{name}", value)
    end
  end
end

class User < StrictStruct
  attribute :name,  type: String
  attribute :age,   type: Integer
end
```

### Decorator with Module#prepend

```ruby
module Logging
  def call(params)
    logger.info("#{self.class}#call started with #{params.keys}")
    result = super
    logger.info("#{self.class}#call completed: #{result.success?}")
    result
  rescue StandardError => e
    logger.error("#{self.class}#call failed: #{e.message}")
    raise
  end
end

class OrderService
  prepend Logging

  def call(params)
    # business logic
  end
end
```

## RSpec Matchers & Helpers

### Built-in Matchers

```ruby
# Equality
expect(result).to eq(expected)           # value equality (==)
expect(result).to eql(expected)          # type + value (eql?)
expect(result).to be(expected)           # identity (equal?)

# Collections
expect(list).to include(item)
expect(list).to contain_exactly(a, b, c) # any order
expect(hash).to include(key: value)
expect(list).to all(be_positive)

# Changes
expect { action }.to change { user.balance }.by(-50)
expect { action }.to change(counter, :count).from(0).to(1)

# Errors
expect { action }.to raise_error(CustomError, /message/)
expect { action }.not_to raise_error

# Types and predicates
expect(obj).to be_a(String)
expect(obj).to respond_to(:call)
expect(user).to be_active              # calls user.active?
```

### Custom Matchers

```ruby
RSpec::Matchers.define :be_valid_email do
  match do |actual|
    actual.match?(/\A[\w+\-.]+@[a-z\d\-]+(\.[a-z\d\-]+)*\.[a-z]+\z/i)
  end

  failure_message do |actual|
    "expected '#{actual}' to be a valid email address"
  end
end

expect(user.email).to be_valid_email
```

### Shared Examples

```ruby
RSpec.shared_examples "soft deletable" do
  describe "#soft_delete" do
    it "sets deleted_at timestamp" do
      expect { subject.soft_delete }
        .to change(subject, :deleted_at).from(nil)
    end

    it "excludes from default scope" do
      subject.soft_delete
      expect(described_class.all).not_to include(subject)
    end
  end
end

# Usage
RSpec.describe User do
  it_behaves_like "soft deletable"
end
```
