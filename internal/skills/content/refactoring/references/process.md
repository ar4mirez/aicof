# Refactoring Process Reference

Detailed code examples and extended refactoring patterns for the refactoring skill.

---

## Common Refactoring Patterns (Detailed)

### Extract Method Pattern

**Before**:
```typescript
function processOrder(order: Order) {
  // 120 lines of code

  // Validation logic (20 lines)
  if (!order.id) {
    throw new Error('Order ID required');
  }
  if (!order.items || order.items.length === 0) {
    throw new Error('Order must have items');
  }
  if (!order.customerId) {
    throw new Error('Customer ID required');
  }
  // ... more validation

  // Calculation logic (25 lines)
  let subtotal = 0;
  for (const item of order.items) {
    subtotal += item.price * item.quantity;
  }

  let tax = subtotal * 0.08;
  let shipping = calculateShipping(order);
  let discount = 0;

  if (order.couponCode) {
    discount = applyCoupon(order.couponCode, subtotal);
  }

  let total = subtotal + tax + shipping - discount;
  // ... more calculations

  // Payment logic (30 lines)
  const paymentMethod = getPaymentMethod(order.customerId);

  try {
    const charge = await paymentGateway.charge({
      amount: total,
      method: paymentMethod,
      customerId: order.customerId,
    });

    order.paymentId = charge.id;
    order.status = 'paid';
  } catch (error) {
    order.status = 'payment_failed';
    throw new PaymentError(error);
  }
  // ... more payment handling

  // Notification logic (20 lines)
  const customer = await getCustomer(order.customerId);
  const emailData = {
    to: customer.email,
    subject: 'Order Confirmation',
    template: 'order-confirmation',
    data: {
      orderId: order.id,
      total: total,
      items: order.items,
    },
  };

  await emailService.send(emailData);
  // ... more notification logic
}
```

**After**:
```typescript
function processOrder(order: Order) {
  validateOrder(order);
  const totals = calculateTotals(order);
  processPayment(order, totals);
  sendConfirmation(order);
}

function validateOrder(order: Order) {
  if (!order.id) {
    throw new Error('Order ID required');
  }
  if (!order.items || order.items.length === 0) {
    throw new Error('Order must have items');
  }
  if (!order.customerId) {
    throw new Error('Customer ID required');
  }
  // ... more validation
}

function calculateTotals(order: Order): OrderTotals {
  let subtotal = 0;
  for (const item of order.items) {
    subtotal += item.price * item.quantity;
  }

  const tax = subtotal * 0.08;
  const shipping = calculateShipping(order);
  const discount = order.couponCode
    ? applyCoupon(order.couponCode, subtotal)
    : 0;

  const total = subtotal + tax + shipping - discount;

  return { subtotal, tax, shipping, discount, total };
}

function processPayment(order: Order, totals: OrderTotals) {
  const paymentMethod = getPaymentMethod(order.customerId);

  try {
    const charge = await paymentGateway.charge({
      amount: totals.total,
      method: paymentMethod,
      customerId: order.customerId,
    });

    order.paymentId = charge.id;
    order.status = 'paid';
  } catch (error) {
    order.status = 'payment_failed';
    throw new PaymentError(error);
  }
}

function sendConfirmation(order: Order) {
  const customer = await getCustomer(order.customerId);
  const emailData = {
    to: customer.email,
    subject: 'Order Confirmation',
    template: 'order-confirmation',
    data: {
      orderId: order.id,
      total: order.total,
      items: order.items,
    },
  };

  await emailService.send(emailData);
}
```

**Benefits**:
- Main function is now self-documenting
- Each function has single responsibility
- Easy to test each part independently
- Easy to modify without affecting others

---

### Extract Class Pattern

**Before**: User class with 400 lines
```typescript
class User {
  id: string;
  email: string;
  password: string;
  firstName: string;
  lastName: string;
  avatar: string;
  bio: string;
  notificationPreferences: object;
  billingAddress: object;

  // Authentication methods (100 lines)
  async login(password: string) {
    const isValid = await bcrypt.compare(password, this.password);
    if (!isValid) throw new Error('Invalid password');

    const token = jwt.sign({ id: this.id }, SECRET);
    await this.updateLastLogin();
    await this.trackLoginEvent();

    return token;
  }

  async updatePassword(newPassword: string) {
    this.password = await bcrypt.hash(newPassword, 10);
    await this.save();
    await this.sendPasswordChangeEmail();
  }

  async resetPassword(token: string) {
    // ... password reset logic
  }

  async verifyEmail(token: string) {
    // ... email verification logic
  }

  // Profile methods (100 lines)
  async updateProfile(data: ProfileData) {
    this.firstName = data.firstName;
    this.lastName = data.lastName;
    this.bio = data.bio;
    await this.save();
  }

  async uploadAvatar(file: File) {
    const url = await uploadToS3(file);
    this.avatar = url;
    await this.save();
  }

  async getPublicProfile() {
    return {
      id: this.id,
      firstName: this.firstName,
      lastName: this.lastName,
      avatar: this.avatar,
      bio: this.bio,
    };
  }

  // Notification methods (100 lines)
  async sendNotification(type: string, data: any) {
    if (!this.notificationPreferences[type]) return;

    // ... notification sending logic
  }

  async updateNotificationPreferences(prefs: object) {
    this.notificationPreferences = prefs;
    await this.save();
  }

  // Billing methods (100 lines)
  async addPaymentMethod(method: PaymentMethod) {
    // ... payment method logic
  }

  async chargeCard(amount: number) {
    // ... charging logic
  }

  async updateBillingAddress(address: Address) {
    this.billingAddress = address;
    await this.save();
  }
}
```

**After**: Separated by responsibility
```typescript
class User {
  id: string;
  email: string;
  firstName: string;
  lastName: string;

  auth: UserAuthentication;
  profile: UserProfile;
  notifications: UserNotifications;
  billing: UserBilling;

  constructor(data: UserData) {
    this.id = data.id;
    this.email = data.email;
    this.firstName = data.firstName;
    this.lastName = data.lastName;

    this.auth = new UserAuthentication(this);
    this.profile = new UserProfile(this);
    this.notifications = new UserNotifications(this);
    this.billing = new UserBilling(this);
  }
}

class UserAuthentication {
  constructor(private user: User) {}

  async login(password: string) {
    const isValid = await bcrypt.compare(password, this.user.password);
    if (!isValid) throw new Error('Invalid password');

    const token = jwt.sign({ id: this.user.id }, SECRET);
    await this.updateLastLogin();
    await this.trackLoginEvent();

    return token;
  }

  async updatePassword(newPassword: string) {
    this.user.password = await bcrypt.hash(newPassword, 10);
    await this.user.save();
    await this.sendPasswordChangeEmail();
  }

  async resetPassword(token: string) {
    // ... password reset logic
  }

  async verifyEmail(token: string) {
    // ... email verification logic
  }
}

class UserProfile {
  constructor(private user: User) {}

  async update(data: ProfileData) {
    this.user.firstName = data.firstName;
    this.user.lastName = data.lastName;
    this.user.bio = data.bio;
    await this.user.save();
  }

  async uploadAvatar(file: File) {
    const url = await uploadToS3(file);
    this.user.avatar = url;
    await this.user.save();
  }

  getPublic() {
    return {
      id: this.user.id,
      firstName: this.user.firstName,
      lastName: this.user.lastName,
      avatar: this.user.avatar,
      bio: this.user.bio,
    };
  }
}

class UserNotifications {
  constructor(private user: User) {}

  async send(type: string, data: any) {
    if (!this.user.notificationPreferences[type]) return;

    // ... notification sending logic
  }

  async updatePreferences(prefs: object) {
    this.user.notificationPreferences = prefs;
    await this.user.save();
  }
}

class UserBilling {
  constructor(private user: User) {}

  async addPaymentMethod(method: PaymentMethod) {
    // ... payment method logic
  }

  async charge(amount: number) {
    // ... charging logic
  }

  async updateAddress(address: Address) {
    this.user.billingAddress = address;
    await this.user.save();
  }
}
```

**Benefits**:
- Each class has single responsibility
- User class is now a lightweight coordinator
- Each subsystem can be tested independently
- Easy to replace implementations (e.g., different notification providers)

---

### Replace Conditional with Polymorphism Pattern

**Before**: Complex switch statement
```typescript
function calculateShipping(order: Order) {
  switch (order.shippingType) {
    case 'standard':
      return order.weight * 5;

    case 'express':
      const baseExpress = order.weight * 10;
      const expressExtra = 15;
      return baseExpress + expressExtra;

    case 'overnight':
      const baseOvernight = order.weight * 15;
      const overnightExtra = 30;
      if (order.isInternational) {
        return baseOvernight + overnightExtra + 50;
      }
      return baseOvernight + overnightExtra;

    case 'pickup':
      return 0;

    default:
      throw new Error('Unknown shipping type');
  }
}

function getEstimatedDeliveryDays(order: Order) {
  switch (order.shippingType) {
    case 'standard':
      return 5;
    case 'express':
      return 2;
    case 'overnight':
      return 1;
    case 'pickup':
      return 0;
    default:
      throw new Error('Unknown shipping type');
  }
}

function canTrack(order: Order) {
  switch (order.shippingType) {
    case 'standard':
      return true;
    case 'express':
      return true;
    case 'overnight':
      return true;
    case 'pickup':
      return false;
    default:
      throw new Error('Unknown shipping type');
  }
}
```

**After**: Strategy pattern with polymorphism
```typescript
interface ShippingStrategy {
  calculate(order: Order): number;
  getEstimatedDeliveryDays(): number;
  canTrack(): boolean;
  getName(): string;
}

class StandardShipping implements ShippingStrategy {
  calculate(order: Order): number {
    return order.weight * 5;
  }

  getEstimatedDeliveryDays(): number {
    return 5;
  }

  canTrack(): boolean {
    return true;
  }

  getName(): string {
    return 'Standard Shipping';
  }
}

class ExpressShipping implements ShippingStrategy {
  calculate(order: Order): number {
    return order.weight * 10 + 15;
  }

  getEstimatedDeliveryDays(): number {
    return 2;
  }

  canTrack(): boolean {
    return true;
  }

  getName(): string {
    return 'Express Shipping';
  }
}

class OvernightShipping implements ShippingStrategy {
  calculate(order: Order): number {
    let cost = order.weight * 15 + 30;
    if (order.isInternational) {
      cost += 50;
    }
    return cost;
  }

  getEstimatedDeliveryDays(): number {
    return 1;
  }

  canTrack(): boolean {
    return true;
  }

  getName(): string {
    return 'Overnight Shipping';
  }
}

class PickupShipping implements ShippingStrategy {
  calculate(order: Order): number {
    return 0;
  }

  getEstimatedDeliveryDays(): number {
    return 0;
  }

  canTrack(): boolean {
    return false;
  }

  getName(): string {
    return 'In-Store Pickup';
  }
}

// Factory to create strategies
class ShippingFactory {
  static create(type: string): ShippingStrategy {
    switch (type) {
      case 'standard':
        return new StandardShipping();
      case 'express':
        return new ExpressShipping();
      case 'overnight':
        return new OvernightShipping();
      case 'pickup':
        return new PickupShipping();
      default:
        throw new Error('Unknown shipping type');
    }
  }
}

// Usage
function processOrder(order: Order) {
  const strategy = ShippingFactory.create(order.shippingType);

  const cost = strategy.calculate(order);
  const deliveryDays = strategy.getEstimatedDeliveryDays();
  const trackable = strategy.canTrack();

  console.log(`${strategy.getName()}: $${cost}, ${deliveryDays} days`);
}
```

**Benefits**:
- Each shipping type is now a self-contained class
- Easy to add new shipping types (just add a new class)
- No switch statements scattered throughout code
- Each strategy can have complex logic without polluting other strategies
- Open/Closed Principle: open for extension, closed for modification

---

## Extract Parameter Object Pattern

**Before**: Long parameter list
```typescript
function createUser(
  firstName: string,
  lastName: string,
  email: string,
  password: string,
  phoneNumber: string,
  address: string,
  city: string,
  state: string,
  zipCode: string,
  country: string,
  dateOfBirth: Date,
  acceptedTerms: boolean
) {
  // ... implementation
}

// Calling this function is error-prone
createUser(
  'John',
  'Doe',
  'john@example.com',
  'password123',
  '555-1234',
  '123 Main St',
  'Springfield',
  'IL',
  '62701',
  'USA',
  new Date('1990-01-01'),
  true
);
```

**After**: Parameter object
```typescript
interface UserData {
  firstName: string;
  lastName: string;
  email: string;
  password: string;
  phoneNumber: string;
  address: Address;
  dateOfBirth: Date;
  acceptedTerms: boolean;
}

interface Address {
  street: string;
  city: string;
  state: string;
  zipCode: string;
  country: string;
}

function createUser(data: UserData) {
  // ... implementation
}

// Much clearer and type-safe
createUser({
  firstName: 'John',
  lastName: 'Doe',
  email: 'john@example.com',
  password: 'password123',
  phoneNumber: '555-1234',
  address: {
    street: '123 Main St',
    city: 'Springfield',
    state: 'IL',
    zipCode: '62701',
    country: 'USA',
  },
  dateOfBirth: new Date('1990-01-01'),
  acceptedTerms: true,
});
```

**Benefits**:
- Named parameters prevent mistakes
- Easy to add optional parameters
- Better IDE autocomplete support
- Can validate the entire object at once
- Easier to refactor (add/remove fields)

---

## Inline Variable/Method Pattern

**Before**: Over-abstraction
```typescript
function calculateDiscount(order: Order) {
  const basePrice = getBasePrice(order);
  const quantityDiscount = getQuantityDiscount(order);
  const seasonalDiscount = getSeasonalDiscount(order);
  const totalDiscount = quantityDiscount + seasonalDiscount;
  return basePrice - totalDiscount;
}

function getBasePrice(order: Order) {
  return order.subtotal;
}

function getQuantityDiscount(order: Order) {
  return order.quantity > 10 ? 10 : 0;
}

function getSeasonalDiscount(order: Order) {
  return 5;
}
```

**After**: Inlined for clarity (when abstraction adds no value)
```typescript
function calculateDiscount(order: Order) {
  const quantityDiscount = order.quantity > 10 ? 10 : 0;
  const seasonalDiscount = 5;
  return order.subtotal - (quantityDiscount + seasonalDiscount);
}
```

**When to Inline**:
- Method/variable is used only once
- Method body is as clear as the name
- Over-abstraction makes code harder to follow

---

## Move Method Pattern

**Before**: Feature envy (method uses another class more than its own)
```typescript
class Order {
  customerId: string;
  items: OrderItem[];

  calculateLoyaltyPoints(): number {
    const customer = Customer.find(this.customerId);
    let points = 0;

    // Using customer's data more than order's data
    if (customer.membershipLevel === 'gold') {
      points = this.total * 2;
    } else if (customer.membershipLevel === 'silver') {
      points = this.total * 1.5;
    } else {
      points = this.total;
    }

    if (customer.yearsAsMember > 5) {
      points *= 1.1;
    }

    return points;
  }
}
```

**After**: Method moved to appropriate class
```typescript
class Order {
  customerId: string;
  items: OrderItem[];

  calculateLoyaltyPoints(): number {
    const customer = Customer.find(this.customerId);
    return customer.calculateLoyaltyPoints(this.total);
  }
}

class Customer {
  membershipLevel: string;
  yearsAsMember: number;

  calculateLoyaltyPoints(orderTotal: number): number {
    let multiplier = 1;

    if (this.membershipLevel === 'gold') {
      multiplier = 2;
    } else if (this.membershipLevel === 'silver') {
      multiplier = 1.5;
    }

    if (this.yearsAsMember > 5) {
      multiplier *= 1.1;
    }

    return orderTotal * multiplier;
  }
}
```

**Benefits**:
- Method is now in the class that has the data it needs
- Easier to test Customer's loyalty logic independently
- Follows "Tell, Don't Ask" principle

---

## Additional Refactoring Patterns

### Replace Magic Number with Named Constant

**Before**:
```typescript
function calculateTax(amount: number) {
  return amount * 0.08;
}

function isEligibleForFreeShipping(total: number) {
  return total > 50;
}
```

**After**:
```typescript
const TAX_RATE = 0.08;
const FREE_SHIPPING_THRESHOLD = 50;

function calculateTax(amount: number) {
  return amount * TAX_RATE;
}

function isEligibleForFreeShipping(total: number) {
  return total > FREE_SHIPPING_THRESHOLD;
}
```

### Replace Nested Conditional with Guard Clauses

**Before**:
```typescript
function getPaymentAmount(customer: Customer) {
  let result;

  if (customer.isDead) {
    result = deadAmount();
  } else {
    if (customer.isSeparated) {
      result = separatedAmount();
    } else {
      if (customer.isRetired) {
        result = retiredAmount();
      } else {
        result = normalAmount();
      }
    }
  }

  return result;
}
```

**After**:
```typescript
function getPaymentAmount(customer: Customer) {
  if (customer.isDead) return deadAmount();
  if (customer.isSeparated) return separatedAmount();
  if (customer.isRetired) return retiredAmount();

  return normalAmount();
}
```

### Replace Type Code with State/Strategy

**Before**:
```typescript
class Employee {
  type: 'engineer' | 'manager' | 'salesperson';

  getBonus() {
    switch (this.type) {
      case 'engineer':
        return this.salary * 0.1;
      case 'manager':
        return this.salary * 0.2;
      case 'salesperson':
        return this.salary * 0.15 + this.commission;
    }
  }
}
```

**After**:
```typescript
interface EmployeeType {
  getBonus(employee: Employee): number;
}

class Engineer implements EmployeeType {
  getBonus(employee: Employee) {
    return employee.salary * 0.1;
  }
}

class Manager implements EmployeeType {
  getBonus(employee: Employee) {
    return employee.salary * 0.2;
  }
}

class Salesperson implements EmployeeType {
  getBonus(employee: Employee) {
    return employee.salary * 0.15 + employee.commission;
  }
}

class Employee {
  type: EmployeeType;
  salary: number;
  commission?: number;

  getBonus() {
    return this.type.getBonus(this);
  }
}
```

---

## Language-Specific Refactoring Patterns

### Python: Replace Loop with Comprehension

**Before**:
```python
result = []
for item in items:
    if item.active:
        result.append(item.name.upper())
```

**After**:
```python
result = [item.name.upper() for item in items if item.active]
```

### Go: Replace Error Handling with Helper

**Before**:
```go
func processData() error {
    data, err := fetchData()
    if err != nil {
        return fmt.Errorf("fetch failed: %w", err)
    }

    result, err := transformData(data)
    if err != nil {
        return fmt.Errorf("transform failed: %w", err)
    }

    err = saveData(result)
    if err != nil {
        return fmt.Errorf("save failed: %w", err)
    }

    return nil
}
```

**After**:
```go
type step func() error

func processData() error {
    steps := []struct {
        fn   func() error
        desc string
    }{
        {fetchData, "fetch"},
        {transformData, "transform"},
        {saveData, "save"},
    }

    for _, s := range steps {
        if err := s.fn(); err != nil {
            return fmt.Errorf("%s failed: %w", s.desc, err)
        }
    }

    return nil
}
```

---

## Refactoring Safety Checklist

### Before Starting
- [ ] All tests pass
- [ ] Code coverage is adequate (>60%)
- [ ] You understand the current behavior
- [ ] You have time to do this properly
- [ ] You've created a checkpoint commit

### During Refactoring
- [ ] Make one change at a time
- [ ] Run tests after each change
- [ ] Commit after each successful change
- [ ] Don't change behavior (only structure)
- [ ] Don't add features while refactoring

### After Refactoring
- [ ] All tests still pass
- [ ] Coverage is maintained or improved
- [ ] No new linter warnings
- [ ] Code is measurably better (shorter, clearer, less complex)
- [ ] Documentation is updated
- [ ] Patterns are documented if needed

---

## Metrics to Track Improvement

### Before & After Comparison

| Metric | Before | After | Change |
|--------|--------|-------|--------|
| Lines of code | 450 | 380 | -15% |
| Function length (avg) | 85 | 35 | -59% |
| Cyclomatic complexity | 15 | 6 | -60% |
| Test coverage | 65% | 82% | +17% |
| Number of functions | 8 | 18 | +125% |

**Note**: More functions is often good (indicates proper decomposition)

---

## References

- **Book**: "Refactoring: Improving the Design of Existing Code" by Martin Fowler
- **Book**: "Working Effectively with Legacy Code" by Michael Feathers
- **Website**: https://refactoring.com/catalog/
- **Pattern**: SOLID principles
- **Pattern**: Design patterns (Strategy, Factory, etc.)
