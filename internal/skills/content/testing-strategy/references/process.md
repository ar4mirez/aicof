# Testing Strategy - Extended Process Guide

Detailed testing examples, patterns, and comprehensive implementation guidance.

---

## Table of Contents

1. [Detailed Edge Case Examples](#detailed-edge-case-examples)
2. [Flaky Test Patterns](#flaky-test-patterns)
3. [Test Pyramid Examples](#test-pyramid-examples)
4. [Coverage Analysis Deep Dive](#coverage-analysis-deep-dive)
5. [Sprint Planning Details](#sprint-planning-details)

---

## Detailed Edge Case Examples

### Complete Edge Case Template

```typescript
describe('validateUser', () => {
  // Happy path
  it('should validate correct user data', () => {
    const validUser = {
      username: 'john_doe',
      email: 'john@example.com',
      age: 25
    };
    expect(validateUser(validUser)).toBe(true);
  });

  // Null/Undefined
  it('should reject null input', () => {
    expect(() => validateUser(null)).toThrow('Invalid input');
  });

  it('should reject undefined input', () => {
    expect(() => validateUser(undefined)).toThrow('Invalid input');
  });

  // Empty
  it('should reject empty object', () => {
    expect(() => validateUser({})).toThrow('Missing required fields');
  });

  it('should reject empty string username', () => {
    const user = { username: '', email: 'john@example.com', age: 25 };
    expect(() => validateUser(user)).toThrow('Username cannot be empty');
  });

  it('should reject empty string email', () => {
    const user = { username: 'john_doe', email: '', age: 25 };
    expect(() => validateUser(user)).toThrow('Email cannot be empty');
  });

  // Boundary
  it('should reject username under minimum length', () => {
    const user = { username: 'ab', email: 'john@example.com', age: 25 };
    expect(() => validateUser(user)).toThrow('Username too short');
  });

  it('should reject username over maximum length', () => {
    const longUsername = 'a'.repeat(51);
    const user = { username: longUsername, email: 'john@example.com', age: 25 };
    expect(() => validateUser(user)).toThrow('Username too long');
  });

  it('should accept username at minimum length', () => {
    const user = { username: 'abc', email: 'john@example.com', age: 25 };
    expect(validateUser(user)).toBe(true);
  });

  it('should accept username at maximum length', () => {
    const maxUsername = 'a'.repeat(50);
    const user = { username: maxUsername, email: 'john@example.com', age: 25 };
    expect(validateUser(user)).toBe(true);
  });

  it('should reject age under minimum', () => {
    const user = { username: 'john_doe', email: 'john@example.com', age: 12 };
    expect(() => validateUser(user)).toThrow('Age must be at least 13');
  });

  it('should reject age over maximum', () => {
    const user = { username: 'john_doe', email: 'john@example.com', age: 151 };
    expect(() => validateUser(user)).toThrow('Age must be less than 150');
  });

  it('should accept age at minimum boundary', () => {
    const user = { username: 'john_doe', email: 'john@example.com', age: 13 };
    expect(validateUser(user)).toBe(true);
  });

  it('should accept age at maximum boundary', () => {
    const user = { username: 'john_doe', email: 'john@example.com', age: 150 };
    expect(validateUser(user)).toBe(true);
  });

  // Invalid type
  it('should reject non-object input', () => {
    expect(() => validateUser('string')).toThrow('Invalid input type');
  });

  it('should reject number input', () => {
    expect(() => validateUser(123)).toThrow('Invalid input type');
  });

  it('should reject array input', () => {
    expect(() => validateUser([])).toThrow('Invalid input type');
  });

  it('should reject string age', () => {
    const user = { username: 'john_doe', email: 'john@example.com', age: '25' };
    expect(() => validateUser(user)).toThrow('Age must be a number');
  });

  // Format validation
  it('should reject invalid email format', () => {
    const user = { username: 'john_doe', email: 'invalid-email', age: 25 };
    expect(() => validateUser(user)).toThrow('Invalid email format');
  });

  it('should reject username with invalid characters', () => {
    const user = { username: 'john@doe', email: 'john@example.com', age: 25 };
    expect(() => validateUser(user)).toThrow('Username contains invalid characters');
  });

  // Missing fields
  it('should reject missing username', () => {
    const user = { email: 'john@example.com', age: 25 };
    expect(() => validateUser(user)).toThrow('Username is required');
  });

  it('should reject missing email', () => {
    const user = { username: 'john_doe', age: 25 };
    expect(() => validateUser(user)).toThrow('Email is required');
  });

  it('should reject missing age', () => {
    const user = { username: 'john_doe', email: 'john@example.com' };
    expect(() => validateUser(user)).toThrow('Age is required');
  });

  // Extra fields (if strict validation)
  it('should ignore extra fields', () => {
    const user = {
      username: 'john_doe',
      email: 'john@example.com',
      age: 25,
      extraField: 'ignored'
    };
    expect(validateUser(user)).toBe(true);
  });
});
```

---

## Flaky Test Patterns

### Bad: Timing-dependent (setTimeout)

```typescript
// FLAKY - Avoid this pattern
describe('debounce function', () => {
  it('should debounce calls', () => {
    let counter = 0;
    const debouncedFn = debounce(() => counter++, 100);

    debouncedFn();
    debouncedFn();
    debouncedFn();

    setTimeout(() => {
      expect(counter).toBe(1);
    }, 150);
  });
});
```

### Good: Proper async handling

```typescript
// STABLE - Use this pattern
describe('debounce function', () => {
  it('should debounce calls', async () => {
    let counter = 0;
    const debouncedFn = debounce(() => counter++, 100);

    debouncedFn();
    debouncedFn();
    debouncedFn();

    await new Promise(resolve => setTimeout(resolve, 150));
    expect(counter).toBe(1);
  });
});
```

### Bad: Shared state between tests

```typescript
// FLAKY - Tests affect each other
let counter = 0;

describe('counter tests', () => {
  it('should increment', () => {
    counter++;
    expect(counter).toBe(1);
  });

  it('should be one', () => {
    expect(counter).toBe(1); // FAILS if previous test ran
  });

  it('should increment again', () => {
    counter++;
    expect(counter).toBe(2); // FAILS if tests run in different order
  });
});
```

### Good: Isolated state

```typescript
// STABLE - Each test is independent
describe('counter tests', () => {
  let counter;

  beforeEach(() => {
    counter = 0;
  });

  it('should increment', () => {
    counter++;
    expect(counter).toBe(1);
  });

  it('should start at zero', () => {
    expect(counter).toBe(0);
  });

  it('should increment from zero', () => {
    counter++;
    expect(counter).toBe(1);
  });
});
```

### Bad: Date/time dependencies

```typescript
// FLAKY - Fails at different times
describe('isToday', () => {
  it('should return true for today', () => {
    const today = new Date();
    expect(isToday(today)).toBe(true);
  });

  it('should return false for yesterday', () => {
    const yesterday = new Date();
    yesterday.setDate(yesterday.getDate() - 1);
    expect(isToday(yesterday)).toBe(false); // FAILS at midnight
  });
});
```

### Good: Mocked dates

```typescript
// STABLE - Consistent time
describe('isToday', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    jest.setSystemTime(new Date('2024-01-15T12:00:00Z'));
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('should return true for today', () => {
    const today = new Date('2024-01-15T12:00:00Z');
    expect(isToday(today)).toBe(true);
  });

  it('should return false for yesterday', () => {
    const yesterday = new Date('2024-01-14T12:00:00Z');
    expect(isToday(yesterday)).toBe(false);
  });
});
```

### Bad: External service calls

```typescript
// FLAKY - Depends on network/service
describe('fetchUser', () => {
  it('should fetch user from API', async () => {
    const user = await fetchUser(123);
    expect(user.id).toBe(123);
    expect(user.name).toBe('John Doe'); // FAILS if API changes
  });
});
```

### Good: Mocked external calls

```typescript
// STABLE - Controlled responses
describe('fetchUser', () => {
  beforeEach(() => {
    jest.mock('./api');
  });

  it('should fetch user from API', async () => {
    const mockUser = { id: 123, name: 'John Doe' };
    api.getUser.mockResolvedValue(mockUser);

    const user = await fetchUser(123);
    expect(user.id).toBe(123);
    expect(user.name).toBe('John Doe');
  });

  it('should handle API errors', async () => {
    api.getUser.mockRejectedValue(new Error('Network error'));

    await expect(fetchUser(123)).rejects.toThrow('Network error');
  });
});
```

### Bad: Race conditions

```typescript
// FLAKY - Results vary
describe('concurrent operations', () => {
  it('should handle parallel updates', async () => {
    const promises = [
      updateCounter(1),
      updateCounter(1),
      updateCounter(1)
    ];
    await Promise.all(promises);

    const counter = await getCounter();
    expect(counter).toBe(3); // Sometimes 1 or 2 due to race condition
  });
});
```

### Good: Sequential or properly synchronized

```typescript
// STABLE - Predictable results
describe('concurrent operations', () => {
  it('should handle parallel updates with proper locking', async () => {
    const promises = [
      updateCounterWithLock(1),
      updateCounterWithLock(1),
      updateCounterWithLock(1)
    ];
    await Promise.all(promises);

    const counter = await getCounter();
    expect(counter).toBe(3);
  });

  it('should handle sequential updates', async () => {
    await updateCounter(1);
    await updateCounter(1);
    await updateCounter(1);

    const counter = await getCounter();
    expect(counter).toBe(3);
  });
});
```

---

## Test Pyramid Examples

### Example: E-commerce Application

#### Unit Tests (70%)

```typescript
// Pure business logic
describe('calculateOrderTotal', () => {
  it('should sum item prices', () => {
    const items = [
      { price: 10, quantity: 2 },
      { price: 20, quantity: 1 }
    ];
    expect(calculateOrderTotal(items)).toBe(40);
  });

  it('should apply discount', () => {
    const items = [{ price: 100, quantity: 1 }];
    const discount = 0.1; // 10%
    expect(calculateOrderTotal(items, discount)).toBe(90);
  });

  it('should handle empty cart', () => {
    expect(calculateOrderTotal([])).toBe(0);
  });
});

// Utility functions
describe('formatPrice', () => {
  it('should format USD currency', () => {
    expect(formatPrice(1234.56, 'USD')).toBe('$1,234.56');
  });

  it('should format EUR currency', () => {
    expect(formatPrice(1234.56, 'EUR')).toBe('€1,234.56');
  });

  it('should handle zero', () => {
    expect(formatPrice(0, 'USD')).toBe('$0.00');
  });
});

// State management
describe('cartReducer', () => {
  it('should add item to cart', () => {
    const state = { items: [] };
    const action = { type: 'ADD_ITEM', payload: { id: 1, price: 10 } };
    const newState = cartReducer(state, action);
    expect(newState.items).toHaveLength(1);
    expect(newState.items[0].id).toBe(1);
  });

  it('should remove item from cart', () => {
    const state = { items: [{ id: 1, price: 10 }] };
    const action = { type: 'REMOVE_ITEM', payload: { id: 1 } };
    const newState = cartReducer(state, action);
    expect(newState.items).toHaveLength(0);
  });

  it('should update item quantity', () => {
    const state = { items: [{ id: 1, price: 10, quantity: 1 }] };
    const action = { type: 'UPDATE_QUANTITY', payload: { id: 1, quantity: 3 } };
    const newState = cartReducer(state, action);
    expect(newState.items[0].quantity).toBe(3);
  });
});
```

#### Integration Tests (20%)

```typescript
// API endpoints
describe('POST /api/orders', () => {
  beforeEach(async () => {
    await db.clearOrders();
  });

  it('should create order with valid data', async () => {
    const orderData = {
      userId: 123,
      items: [{ productId: 1, quantity: 2 }],
      shippingAddress: { street: '123 Main St', city: 'City' }
    };

    const response = await request(app)
      .post('/api/orders')
      .send(orderData);

    expect(response.status).toBe(201);
    expect(response.body.order.id).toBeDefined();
    expect(response.body.order.total).toBeGreaterThan(0);
  });

  it('should reject order with invalid product', async () => {
    const orderData = {
      userId: 123,
      items: [{ productId: 999, quantity: 2 }],
      shippingAddress: { street: '123 Main St', city: 'City' }
    };

    const response = await request(app)
      .post('/api/orders')
      .send(orderData);

    expect(response.status).toBe(404);
    expect(response.body.error).toContain('Product not found');
  });

  it('should reject order without authentication', async () => {
    const response = await request(app)
      .post('/api/orders')
      .send({ items: [] });

    expect(response.status).toBe(401);
  });
});

// Database operations
describe('OrderRepository', () => {
  let repository;

  beforeEach(async () => {
    await db.migrate();
    repository = new OrderRepository(db);
  });

  afterEach(async () => {
    await db.clearAll();
  });

  it('should save order to database', async () => {
    const order = {
      userId: 123,
      items: [{ productId: 1, quantity: 2, price: 10 }],
      total: 20
    };

    const savedOrder = await repository.save(order);
    expect(savedOrder.id).toBeDefined();

    const retrieved = await repository.findById(savedOrder.id);
    expect(retrieved.userId).toBe(123);
    expect(retrieved.total).toBe(20);
  });

  it('should update order status', async () => {
    const order = await repository.save({ userId: 123, status: 'pending' });

    await repository.updateStatus(order.id, 'completed');

    const updated = await repository.findById(order.id);
    expect(updated.status).toBe('completed');
  });
});

// Service interactions
describe('PaymentService', () => {
  beforeEach(() => {
    jest.mock('./paymentGateway');
  });

  it('should process payment successfully', async () => {
    const paymentData = {
      amount: 100,
      currency: 'USD',
      cardToken: 'tok_123'
    };

    paymentGateway.charge.mockResolvedValue({
      success: true,
      transactionId: 'txn_123'
    });

    const result = await paymentService.processPayment(paymentData);

    expect(result.success).toBe(true);
    expect(result.transactionId).toBe('txn_123');
  });

  it('should handle payment failure', async () => {
    paymentGateway.charge.mockResolvedValue({
      success: false,
      error: 'Insufficient funds'
    });

    const result = await paymentService.processPayment({
      amount: 100,
      currency: 'USD',
      cardToken: 'tok_123'
    });

    expect(result.success).toBe(false);
    expect(result.error).toContain('Insufficient funds');
  });
});
```

#### E2E Tests (10%)

```typescript
// Critical user journey
describe('Complete checkout flow', () => {
  beforeEach(async () => {
    await browser.url('/');
    await setupTestData();
  });

  it('should complete purchase from cart to confirmation', async () => {
    // 1. Browse products
    await browser.url('/products');
    await expect($('h1')).toHaveText('Products');

    // 2. Add items to cart
    await $('#product-1 .add-to-cart').click();
    await $('#product-2 .add-to-cart').click();

    // 3. View cart
    await $('#cart-icon').click();
    await expect($$('.cart-item')).toHaveLength(2);

    // 4. Proceed to checkout
    await $('#checkout-button').click();
    await expect($('h1')).toHaveText('Checkout');

    // 5. Fill shipping information
    await $('#shipping-street').setValue('123 Main St');
    await $('#shipping-city').setValue('City');
    await $('#shipping-zip').setValue('12345');
    await $('#continue-to-payment').click();

    // 6. Enter payment details
    await $('#card-number').setValue('4242424242424242');
    await $('#card-expiry').setValue('12/25');
    await $('#card-cvc').setValue('123');
    await $('#place-order').click();

    // 7. Verify confirmation
    await browser.waitUntil(async () => {
      return (await $('h1').getText()) === 'Order Confirmed';
    }, 5000);

    const orderNumber = await $('#order-number').getText();
    expect(orderNumber).toMatch(/^ORD-\d+$/);
  });

  it('should handle payment failure gracefully', async () => {
    // Add items and go to checkout
    await $('#product-1 .add-to-cart').click();
    await $('#cart-icon').click();
    await $('#checkout-button').click();

    // Fill shipping
    await $('#shipping-street').setValue('123 Main St');
    await $('#continue-to-payment').click();

    // Use card that triggers failure
    await $('#card-number').setValue('4000000000000002');
    await $('#place-order').click();

    // Verify error message
    await expect($('.error-message')).toHaveText('Payment declined');

    // Verify cart still has items
    await $('#cart-icon').click();
    await expect($$('.cart-item')).toHaveLength(1);
  });
});

// Authentication flow
describe('User registration and login', () => {
  it('should register new user and log in', async () => {
    // Navigate to registration
    await browser.url('/register');

    // Fill registration form
    const timestamp = Date.now();
    await $('#username').setValue(`user${timestamp}`);
    await $('#email').setValue(`user${timestamp}@example.com`);
    await $('#password').setValue('SecurePassword123!');
    await $('#confirm-password').setValue('SecurePassword123!');
    await $('#register-button').click();

    // Verify email confirmation page
    await expect($('h1')).toHaveText('Verify Email');

    // Simulate email verification (test helper)
    const verificationToken = await getVerificationToken(`user${timestamp}@example.com`);
    await browser.url(`/verify?token=${verificationToken}`);

    // Verify redirect to login
    await expect($('h1')).toHaveText('Login');

    // Log in
    await $('#email').setValue(`user${timestamp}@example.com`);
    await $('#password').setValue('SecurePassword123!');
    await $('#login-button').click();

    // Verify logged in state
    await expect($('#user-menu')).toBeDisplayed();
    await expect($('#user-menu')).toHaveText(`user${timestamp}`);
  });
});
```

---

## Coverage Analysis Deep Dive

### Analyzing Coverage Reports

#### Example Coverage Report Analysis

```
File                     | Stmts | Branch | Funcs | Lines | Uncovered Line #s
-------------------------|-------|--------|-------|-------|-------------------
src/auth/login.ts        |  45%  |  30%   |  50%  |  45%  | 15-23, 45-67, 89
src/auth/session.ts      |  52%  |  40%   |  60%  |  52%  | 34-45, 78-90
src/auth/middleware.ts   |  88%  |  80%   |  100% |  88%  | 123-125
src/models/user.ts       |  72%  |  65%   |  80%  |  72%  | 56-60, 78
src/api/payments.ts      |  28%  |  15%   |  30%  |  28%  | 12-89, 100-145
src/api/users.ts         |  48%  |  35%   |  50%  |  48%  | 23-45, 67-89
```

#### Priority Analysis

**Critical Priority (P0):**
- `src/api/payments.ts` - 28% coverage, handles money (CRITICAL)
  - Uncovered: Lines 12-89 (payment processing), 100-145 (refunds)
  - Risk: Payment failures, incorrect charges

**High Priority (P1):**
- `src/auth/login.ts` - 45% coverage, authentication (HIGH)
  - Uncovered: Lines 15-23 (password validation), 45-67 (2FA), 89 (lockout)
  - Risk: Security vulnerabilities

**Medium Priority (P2):**
- `src/auth/session.ts` - 52% coverage
  - Uncovered: Lines 34-45 (session refresh), 78-90 (expiration)
  - Risk: Session hijacking, unauthorized access

**Low Priority (P3):**
- `src/models/user.ts` - 72% coverage (acceptable for model)
  - Uncovered: Lines 56-60 (helper method), 78 (toString)
  - Risk: Low

**Already Good:**
- `src/auth/middleware.ts` - 88% coverage (meets target)

### Coverage Improvement Plan

```markdown
## Week 1: Critical Path (Payments)

### Focus: src/api/payments.ts
**Current**: 28% → **Target**: 90%

### Test Cases to Add:

1. **Payment Processing (Lines 12-45)**
   - [ ] Valid payment with card
   - [ ] Valid payment with PayPal
   - [ ] Invalid card number
   - [ ] Expired card
   - [ ] Insufficient funds
   - [ ] Network timeout
   - [ ] Currency mismatch

2. **Payment Validation (Lines 46-67)**
   - [ ] Amount validation (min/max)
   - [ ] Currency validation
   - [ ] Card token validation
   - [ ] User authorization

3. **Error Handling (Lines 68-89)**
   - [ ] Gateway error
   - [ ] Database error
   - [ ] Retry logic
   - [ ] Rollback on failure

4. **Refunds (Lines 100-145)**
   - [ ] Full refund
   - [ ] Partial refund
   - [ ] Refund validation
   - [ ] Already refunded error
   - [ ] Refund too late error

**Estimated Effort**: 12 hours
**Expected Coverage**: 28% → 92%
```

---

## Sprint Planning Details

### Example: 2-Week Sprint

```markdown
## Sprint Goal
Increase overall coverage from 55% to 65% by focusing on critical paths (authentication and payments).

---

## Sprint Breakdown

### Week 1: Authentication Module

#### Day 1-2: Unit Tests (8 hours)
- [ ] src/auth/login.ts - password validation (2 hours)
  - Valid passwords
  - Invalid passwords (too short, no special chars, etc.)
  - Password hashing
- [ ] src/auth/login.ts - 2FA implementation (2 hours)
  - TOTP generation
  - TOTP validation
  - Backup codes
- [ ] src/auth/login.ts - account lockout (2 hours)
  - Failed attempt tracking
  - Lockout threshold
  - Lockout duration
  - Lockout reset
- [ ] src/auth/session.ts - session management (2 hours)
  - Session creation
  - Session validation
  - Session refresh
  - Session expiration

#### Day 3-4: Integration Tests (8 hours)
- [ ] POST /api/auth/login (2 hours)
  - Successful login
  - Invalid credentials
  - Locked account
  - Unverified email
- [ ] POST /api/auth/logout (1 hour)
  - Successful logout
  - Invalid session
- [ ] POST /api/auth/refresh (2 hours)
  - Valid refresh token
  - Expired refresh token
  - Revoked refresh token
- [ ] Middleware authentication (3 hours)
  - Valid token
  - Invalid token
  - Expired token
  - Missing token
  - Revoked token

#### Day 5: E2E Tests (4 hours)
- [ ] Complete login flow (2 hours)
  - Navigate to login
  - Enter credentials
  - 2FA verification
  - Redirect to dashboard
- [ ] Failed login attempts (2 hours)
  - Invalid credentials
  - Account lockout
  - Error messages

**Week 1 Outcome**:
- Auth coverage: 45% → 90%
- Tests added: ~25
- Overall coverage: 55% → 60%

---

### Week 2: Payment Module

#### Day 1-2: Unit Tests (8 hours)
- [ ] src/api/payments.ts - payment processing (4 hours)
  - Valid payments
  - Invalid card
  - Expired card
  - Insufficient funds
  - Amount validation
  - Currency validation
- [ ] src/api/payments.ts - refund logic (4 hours)
  - Full refund
  - Partial refund
  - Refund validation
  - Error cases

#### Day 3-4: Integration Tests (8 hours)
- [ ] POST /api/payments (4 hours)
  - Successful payment
  - Payment failure
  - Invalid amount
  - Invalid currency
  - Unauthorized user
- [ ] POST /api/payments/:id/refund (4 hours)
  - Successful refund
  - Invalid payment ID
  - Already refunded
  - Partial refund
  - Refund too late

#### Day 5: E2E Tests (4 hours)
- [ ] Complete checkout flow (3 hours)
  - Add items to cart
  - Enter shipping info
  - Enter payment info
  - Confirm order
  - Verify confirmation
- [ ] Payment failure flow (1 hour)
  - Enter invalid card
  - Verify error message
  - Retry with valid card

**Week 2 Outcome**:
- Payment coverage: 28% → 92%
- Tests added: ~20
- Overall coverage: 60% → 65%

---

## Sprint Retrospective

### What Went Well
- Achieved coverage target (65%)
- Critical paths now well-tested
- Found 3 bugs during test writing

### What Could Be Improved
- E2E tests took longer than expected
- Some flaky tests in integration suite
- Need better test data factories

### Action Items
- [ ] Fix flaky integration tests
- [ ] Create test data factories
- [ ] Document testing patterns
```

---

## Summary

This extended guide provides:
- Complete edge case examples for thorough testing
- Detailed flaky test patterns (bad vs. good)
- Real-world test pyramid examples
- In-depth coverage analysis techniques
- Comprehensive sprint planning templates

Use these patterns to achieve and maintain >80% coverage on business logic and >60% overall coverage.
