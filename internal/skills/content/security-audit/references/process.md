# Security Audit Process - Detailed Reference

This file contains extended security patterns, detailed checklists, and verbose examples for comprehensive security audits.

---

## Extended OWASP Top 10 Patterns

### A01:2021 - Broken Access Control (Extended)

#### Common Vulnerabilities

**Insecure Direct Object References (IDOR)**:
```javascript
// VULNERABLE: User can access any order by changing ID
app.get('/api/orders/:orderId', (req, res) => {
  const order = db.getOrder(req.params.orderId)
  return res.json(order)
})

// SECURE: Verify user owns the order
app.get('/api/orders/:orderId', authenticate, (req, res) => {
  const order = db.getOrder(req.params.orderId)

  if (!order) {
    return res.status(404).json({ error: 'Order not found' })
  }

  if (order.userId !== req.user.id && !req.user.isAdmin) {
    return res.status(403).json({ error: 'Access denied' })
  }

  return res.json(order)
})
```

**Path Traversal**:
```javascript
// VULNERABLE: Path traversal attack possible
app.get('/files/:filename', (req, res) => {
  res.sendFile(`./uploads/${req.params.filename}`)
})
// Attack: /files/../../etc/passwd

// SECURE: Validate and sanitize file path
const path = require('path')
app.get('/files/:filename', (req, res) => {
  const filename = path.basename(req.params.filename)
  const filepath = path.join(__dirname, 'uploads', filename)

  // Ensure file is within uploads directory
  if (!filepath.startsWith(path.join(__dirname, 'uploads'))) {
    return res.status(403).json({ error: 'Access denied' })
  }

  res.sendFile(filepath)
})
```

**Privilege Escalation**:
```python
# VULNERABLE: User can modify their own role
@app.route('/api/users/<user_id>', methods=['PATCH'])
@login_required
def update_user(user_id):
    data = request.json
    user = User.query.get(user_id)

    if str(user.id) != user_id:
        return {'error': 'Forbidden'}, 403

    # User can set their own role!
    for key, value in data.items():
        setattr(user, key, value)

    db.session.commit()
    return user.to_dict()

# SECURE: Separate endpoints for profile vs admin updates
@app.route('/api/users/<user_id>/profile', methods=['PATCH'])
@login_required
def update_profile(user_id):
    if str(current_user.id) != user_id:
        return {'error': 'Forbidden'}, 403

    data = request.json
    allowed_fields = {'name', 'email', 'avatar'}

    user = User.query.get(user_id)
    for key in allowed_fields:
        if key in data:
            setattr(user, key, data[key])

    db.session.commit()
    return user.to_dict()

@app.route('/api/users/<user_id>/role', methods=['PATCH'])
@login_required
@admin_required
def update_role(user_id):
    data = request.json
    user = User.query.get(user_id)
    user.role = data['role']
    db.session.commit()
    return user.to_dict()
```

#### Testing Checklist

- [ ] Test accessing resources with different user accounts
- [ ] Test modifying IDs in URLs and request bodies
- [ ] Test accessing admin endpoints without admin role
- [ ] Test horizontal privilege escalation (user A accessing user B's data)
- [ ] Test vertical privilege escalation (user becoming admin)
- [ ] Test CORS misconfigurations allowing unauthorized origins
- [ ] Test missing function-level access control
- [ ] Test metadata manipulation (user ID, role in JWT)

---

### A02:2021 - Cryptographic Failures (Extended)

#### Password Hashing Best Practices

**Wrong**:
```javascript
// NEVER do this
const crypto = require('crypto')
const hashedPassword = crypto.createHash('sha256')
  .update(password)
  .digest('hex')
```

**Right**:
```javascript
// Use bcrypt
const bcrypt = require('bcrypt')
const saltRounds = 12 // Cost factor (2^12 iterations)

// Hashing
const hashedPassword = await bcrypt.hash(password, saltRounds)

// Verification
const isValid = await bcrypt.compare(password, hashedPassword)
```

**Python (argon2)**:
```python
from argon2 import PasswordHasher

ph = PasswordHasher()

# Hashing
hashed = ph.hash(password)

# Verification
try:
    ph.verify(hashed, password)
    # Password is correct
except:
    # Password is incorrect
    pass
```

#### TLS Configuration

**Nginx**:
```nginx
server {
    listen 443 ssl http2;
    server_name example.com;

    # TLS 1.2 and 1.3 only
    ssl_protocols TLSv1.2 TLSv1.3;

    # Strong cipher suites
    ssl_ciphers 'ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384';
    ssl_prefer_server_ciphers off;

    # SSL certificates
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    # HSTS
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
}
```

#### Encryption at Rest

**Sensitive Fields**:
```python
from cryptography.fernet import Fernet
import os

class EncryptedField:
    def __init__(self):
        # Store key in environment variable or key management service
        key = os.environ.get('ENCRYPTION_KEY').encode()
        self.cipher = Fernet(key)

    def encrypt(self, value):
        return self.cipher.encrypt(value.encode()).decode()

    def decrypt(self, encrypted_value):
        return self.cipher.decrypt(encrypted_value.encode()).decode()

# Usage in model
class User:
    def __init__(self):
        self._ssn = None
        self.encryptor = EncryptedField()

    @property
    def ssn(self):
        if self._ssn:
            return self.encryptor.decrypt(self._ssn)
        return None

    @ssn.setter
    def ssn(self, value):
        self._ssn = self.encryptor.encrypt(value)
```

---

### A03:2021 - Injection (Extended)

#### SQL Injection Prevention

**Python with SQLAlchemy**:
```python
from sqlalchemy import text

# VULNERABLE
user_id = request.args.get('id')
query = f"SELECT * FROM users WHERE id = {user_id}"
result = db.session.execute(query)

# SECURE: Parameterized query
user_id = request.args.get('id')
query = text("SELECT * FROM users WHERE id = :user_id")
result = db.session.execute(query, {'user_id': user_id})

# BEST: Use ORM
user = User.query.filter_by(id=user_id).first()
```

**Node.js with PostgreSQL**:
```javascript
// VULNERABLE
const userId = req.query.id
const query = `SELECT * FROM users WHERE id = ${userId}`
const result = await db.query(query)

// SECURE: Parameterized query
const userId = req.query.id
const query = 'SELECT * FROM users WHERE id = $1'
const result = await db.query(query, [userId])
```

**Go with database/sql**:
```go
// VULNERABLE
userID := r.URL.Query().Get("id")
query := fmt.Sprintf("SELECT * FROM users WHERE id = %s", userID)
rows, err := db.Query(query)

// SECURE: Parameterized query
userID := r.URL.Query().Get("id")
query := "SELECT * FROM users WHERE id = ?"
rows, err := db.Query(query, userID)
```

#### NoSQL Injection Prevention

**MongoDB**:
```javascript
// VULNERABLE: NoSQL injection
app.get('/users', (req, res) => {
  const filter = { username: req.query.username }
  // Attack: ?username[$ne]=null returns all users
  db.collection('users').find(filter).toArray()
})

// SECURE: Validate input types
app.get('/users', (req, res) => {
  const username = req.query.username

  // Ensure username is a string, not an object
  if (typeof username !== 'string') {
    return res.status(400).json({ error: 'Invalid username' })
  }

  const filter = { username }
  db.collection('users').find(filter).toArray()
})

// BETTER: Use schema validation
const { z } = require('zod')

const querySchema = z.object({
  username: z.string().min(1).max(100)
})

app.get('/users', (req, res) => {
  const { username } = querySchema.parse(req.query)
  const filter = { username }
  db.collection('users').find(filter).toArray()
})
```

#### XSS Prevention

**React (automatic escaping)**:
```jsx
// SAFE: React automatically escapes
function UserProfile({ user }) {
  return <div>{user.name}</div>
}

// DANGEROUS: dangerouslySetInnerHTML bypasses escaping
function UnsafeComponent({ html }) {
  return <div dangerouslySetInnerHTML={{ __html: html }} />
}

// SAFE: Use DOMPurify for user HTML
import DOMPurify from 'dompurify'

function SafeComponent({ html }) {
  const sanitized = DOMPurify.sanitize(html)
  return <div dangerouslySetInnerHTML={{ __html: sanitized }} />
}
```

**Backend (template escaping)**:
```python
from jinja2 import Template

# VULNERABLE: disable autoescaping
template = Template('{{ user_input }}', autoescape=False)

# SECURE: auto-escaping enabled by default
from flask import render_template_string

@app.route('/profile')
def profile():
    user_input = request.args.get('name')
    # Jinja2 auto-escapes by default
    return render_template_string('<h1>{{ name }}</h1>', name=user_input)
```

#### Command Injection Prevention

```python
import subprocess

# VULNERABLE: Shell injection
user_file = request.args.get('file')
subprocess.run(f'cat {user_file}', shell=True)
# Attack: file=test.txt; rm -rf /

# SECURE: Avoid shell, use list
user_file = request.args.get('file')
subprocess.run(['cat', user_file], shell=False)

# BEST: Validate input and use safe alternatives
import os

user_file = request.args.get('file')
# Validate filename
if not re.match(r'^[a-zA-Z0-9_-]+\.txt$', user_file):
    return {'error': 'Invalid filename'}, 400

# Use Python instead of shell command
filepath = os.path.join('uploads', user_file)
with open(filepath, 'r') as f:
    content = f.read()
```

---

### A04:2021 - Insecure Design (Extended)

#### Threat Modeling Checklist

**STRIDE Framework**:
- [ ] **S**poofing: Can users impersonate others?
- [ ] **T**ampering: Can data be modified in transit/storage?
- [ ] **R**epudiation: Can users deny their actions?
- [ ] **I**nformation Disclosure: Can sensitive data leak?
- [ ] **D**enial of Service: Can the system be overwhelmed?
- [ ] **E**levation of Privilege: Can users gain unauthorized access?

**Trust Boundaries**:
```
Client → API → Service → Database
  ↓       ↓        ↓         ↓
Validate  Auth    Validate  Encrypt
Input     Token   Business  At Rest
          Rate    Logic
          Limit
```

#### Defense in Depth Example

```javascript
// Layer 1: Input validation
const userSchema = z.object({
  email: z.string().email(),
  amount: z.number().positive().max(10000)
})

// Layer 2: Authentication
const authenticate = (req, res, next) => {
  const token = req.headers.authorization?.split(' ')[1]
  if (!token) return res.status(401).json({ error: 'Unauthorized' })

  try {
    req.user = jwt.verify(token, process.env.JWT_SECRET)
    next()
  } catch {
    return res.status(401).json({ error: 'Invalid token' })
  }
}

// Layer 3: Authorization
const authorize = (requiredRole) => (req, res, next) => {
  if (!req.user.roles.includes(requiredRole)) {
    return res.status(403).json({ error: 'Forbidden' })
  }
  next()
}

// Layer 4: Rate limiting
const limiter = rateLimit({
  windowMs: 15 * 60 * 1000, // 15 minutes
  max: 100 // limit each IP to 100 requests per windowMs
})

// Layer 5: Business logic validation
const validateTransfer = async (req, res, next) => {
  const { amount } = req.body
  const balance = await getBalance(req.user.id)

  if (balance < amount) {
    return res.status(400).json({ error: 'Insufficient funds' })
  }
  next()
}

// All layers combined
app.post('/api/transfer',
  limiter,
  authenticate,
  authorize('user'),
  validateInput(userSchema),
  validateTransfer,
  async (req, res) => {
    // Transfer logic
  }
)
```

---

### A05:2021 - Security Misconfiguration (Extended)

#### Security Headers Middleware

**Express.js**:
```javascript
const helmet = require('helmet')

app.use(helmet({
  contentSecurityPolicy: {
    directives: {
      defaultSrc: ["'self'"],
      styleSrc: ["'self'", "'unsafe-inline'"],
      scriptSrc: ["'self'"],
      imgSrc: ["'self'", "data:", "https:"],
      connectSrc: ["'self'"],
      fontSrc: ["'self'"],
      objectSrc: ["'none'"],
      mediaSrc: ["'self'"],
      frameSrc: ["'none'"],
    },
  },
  hsts: {
    maxAge: 31536000,
    includeSubDomains: true,
    preload: true
  },
  referrerPolicy: { policy: 'no-referrer' },
  noSniff: true,
  xssFilter: true,
  frameguard: { action: 'deny' }
}))
```

**Python (Flask)**:
```python
from flask_talisman import Talisman

csp = {
    'default-src': "'self'",
    'script-src': "'self'",
    'style-src': ["'self'", "'unsafe-inline'"],
    'img-src': ["'self'", "data:", "https:"],
}

Talisman(app,
    content_security_policy=csp,
    force_https=True,
    strict_transport_security=True,
    strict_transport_security_max_age=31536000,
    frame_options='DENY',
    content_type_options=True
)
```

#### Environment Configuration

```bash
# .env.example (commit this)
NODE_ENV=development
PORT=3000
DATABASE_URL=postgresql://localhost/myapp_dev
JWT_SECRET=changeme
API_KEY=your_api_key_here

# .env (DO NOT commit)
NODE_ENV=production
PORT=3000
DATABASE_URL=postgresql://user:pass@prod-server/myapp
JWT_SECRET=random_generated_secret_here
API_KEY=actual_production_api_key
```

**Loading environment variables**:
```javascript
// config.js
require('dotenv').config()

const config = {
  env: process.env.NODE_ENV || 'development',
  port: parseInt(process.env.PORT, 10) || 3000,

  database: {
    url: process.env.DATABASE_URL || 'postgresql://localhost/myapp_dev'
  },

  jwt: {
    secret: process.env.JWT_SECRET,
    expiresIn: '1h'
  },

  // Validate required variables
  validate() {
    const required = ['DATABASE_URL', 'JWT_SECRET']
    const missing = required.filter(key => !process.env[key])

    if (missing.length > 0) {
      throw new Error(`Missing required environment variables: ${missing.join(', ')}`)
    }
  }
}

config.validate()

module.exports = config
```

#### Error Handling

```javascript
// VULNERABLE: Exposes stack trace
app.use((err, req, res, next) => {
  res.status(500).json({
    error: err.message,
    stack: err.stack // DO NOT expose in production
  })
})

// SECURE: Generic message to client, detailed to logs
app.use((err, req, res, next) => {
  // Log full error server-side
  console.error({
    message: err.message,
    stack: err.stack,
    user: req.user?.id,
    url: req.url,
    method: req.method
  })

  // Send generic message to client
  if (process.env.NODE_ENV === 'production') {
    res.status(500).json({
      error: 'Internal server error',
      requestId: req.id // For support debugging
    })
  } else {
    // Detailed errors in development only
    res.status(500).json({
      error: err.message,
      stack: err.stack
    })
  }
})
```

---

### A07:2021 - Authentication Failures (Extended)

#### Password Reset Flow

```javascript
const crypto = require('crypto')

// Generate reset token
const generateResetToken = () => {
  return crypto.randomBytes(32).toString('hex')
}

// Request password reset
app.post('/api/auth/forgot-password', async (req, res) => {
  const { email } = req.body
  const user = await User.findOne({ email })

  // IMPORTANT: Always return success to prevent email enumeration
  if (!user) {
    return res.json({ message: 'If the email exists, a reset link has been sent' })
  }

  // Generate token
  const resetToken = generateResetToken()
  const resetTokenHash = crypto
    .createHash('sha256')
    .update(resetToken)
    .digest('hex')

  // Store hash (not raw token) with expiration
  user.resetPasswordToken = resetTokenHash
  user.resetPasswordExpires = Date.now() + 3600000 // 1 hour
  await user.save()

  // Send email with raw token
  await sendEmail({
    to: user.email,
    subject: 'Password Reset',
    text: `Reset your password: https://example.com/reset/${resetToken}`
  })

  res.json({ message: 'If the email exists, a reset link has been sent' })
})

// Reset password
app.post('/api/auth/reset-password/:token', async (req, res) => {
  const { token } = req.params
  const { password } = req.body

  // Hash provided token
  const resetTokenHash = crypto
    .createHash('sha256')
    .update(token)
    .digest('hex')

  // Find user with valid token
  const user = await User.findOne({
    resetPasswordToken: resetTokenHash,
    resetPasswordExpires: { $gt: Date.now() }
  })

  if (!user) {
    return res.status(400).json({ error: 'Invalid or expired reset token' })
  }

  // Update password
  user.password = await bcrypt.hash(password, 12)
  user.resetPasswordToken = undefined
  user.resetPasswordExpires = undefined
  await user.save()

  res.json({ message: 'Password has been reset' })
})
```

#### Session Management

```javascript
const session = require('express-session')
const RedisStore = require('connect-redis').default
const { createClient } = require('redis')

// Redis client for session storage
const redisClient = createClient({
  host: process.env.REDIS_HOST,
  port: process.env.REDIS_PORT
})

app.use(session({
  store: new RedisStore({ client: redisClient }),
  secret: process.env.SESSION_SECRET,
  resave: false,
  saveUninitialized: false,
  name: 'sessionId', // Don't use default 'connect.sid'
  cookie: {
    secure: process.env.NODE_ENV === 'production', // HTTPS only in production
    httpOnly: true, // Prevent JavaScript access
    maxAge: 1000 * 60 * 60 * 24, // 24 hours
    sameSite: 'strict' // CSRF protection
  }
}))

// Regenerate session on login
app.post('/api/auth/login', async (req, res) => {
  const { email, password } = req.body
  const user = await User.findOne({ email })

  if (!user || !await bcrypt.compare(password, user.password)) {
    return res.status(401).json({ error: 'Invalid credentials' })
  }

  // Regenerate session to prevent fixation
  req.session.regenerate((err) => {
    if (err) {
      return res.status(500).json({ error: 'Session error' })
    }

    req.session.userId = user.id
    res.json({ message: 'Logged in successfully' })
  })
})

// Logout
app.post('/api/auth/logout', (req, res) => {
  req.session.destroy((err) => {
    if (err) {
      return res.status(500).json({ error: 'Logout failed' })
    }
    res.clearCookie('sessionId')
    res.json({ message: 'Logged out successfully' })
  })
})
```

#### Multi-Factor Authentication

```javascript
const speakeasy = require('speakeasy')
const QRCode = require('qrcode')

// Enable MFA
app.post('/api/auth/mfa/enable', authenticate, async (req, res) => {
  const secret = speakeasy.generateSecret({
    name: `MyApp (${req.user.email})`
  })

  // Store secret temporarily
  req.user.mfaTempSecret = secret.base32
  await req.user.save()

  // Generate QR code
  const qrCode = await QRCode.toDataURL(secret.otpauth_url)

  res.json({
    qrCode,
    secret: secret.base32 // For manual entry
  })
})

// Verify and activate MFA
app.post('/api/auth/mfa/verify', authenticate, async (req, res) => {
  const { token } = req.body

  const verified = speakeasy.totp.verify({
    secret: req.user.mfaTempSecret,
    encoding: 'base32',
    token
  })

  if (!verified) {
    return res.status(400).json({ error: 'Invalid code' })
  }

  // Activate MFA
  req.user.mfaSecret = req.user.mfaTempSecret
  req.user.mfaTempSecret = undefined
  req.user.mfaEnabled = true
  await req.user.save()

  // Generate recovery codes
  const recoveryCodes = Array.from({ length: 10 }, () =>
    crypto.randomBytes(4).toString('hex')
  )

  req.user.recoveryCodes = recoveryCodes.map(code =>
    crypto.createHash('sha256').update(code).digest('hex')
  )
  await req.user.save()

  res.json({
    message: 'MFA enabled successfully',
    recoveryCodes // Show these ONCE to the user
  })
})

// Login with MFA
app.post('/api/auth/login', async (req, res) => {
  const { email, password, mfaToken } = req.body
  const user = await User.findOne({ email })

  if (!user || !await bcrypt.compare(password, user.password)) {
    return res.status(401).json({ error: 'Invalid credentials' })
  }

  // Check if MFA is enabled
  if (user.mfaEnabled) {
    if (!mfaToken) {
      return res.status(401).json({ error: 'MFA token required' })
    }

    const verified = speakeasy.totp.verify({
      secret: user.mfaSecret,
      encoding: 'base32',
      token: mfaToken
    })

    if (!verified) {
      return res.status(401).json({ error: 'Invalid MFA token' })
    }
  }

  // Login successful
  req.session.regenerate((err) => {
    req.session.userId = user.id
    res.json({ message: 'Logged in successfully' })
  })
})
```

---

### A09:2021 - Security Logging Failures (Extended)

#### Comprehensive Logging

```javascript
const winston = require('winston')

// Configure logger
const logger = winston.createLogger({
  level: 'info',
  format: winston.format.combine(
    winston.format.timestamp(),
    winston.format.errors({ stack: true }),
    winston.format.json()
  ),
  defaultMeta: { service: 'api' },
  transports: [
    new winston.transports.File({ filename: 'logs/error.log', level: 'error' }),
    new winston.transports.File({ filename: 'logs/combined.log' }),
  ],
})

// Security event logger
const logSecurityEvent = (event, details) => {
  logger.warn({
    type: 'security_event',
    event,
    ...details,
    timestamp: new Date().toISOString()
  })
}

// Authentication events
app.post('/api/auth/login', async (req, res) => {
  const { email, password } = req.body
  const user = await User.findOne({ email })

  if (!user || !await bcrypt.compare(password, user.password)) {
    logSecurityEvent('login_failed', {
      email,
      ip: req.ip,
      userAgent: req.headers['user-agent']
    })
    return res.status(401).json({ error: 'Invalid credentials' })
  }

  logSecurityEvent('login_success', {
    userId: user.id,
    email: user.email,
    ip: req.ip
  })

  // ... login logic
})

// Authorization failures
const authorize = (requiredRole) => (req, res, next) => {
  if (!req.user.roles.includes(requiredRole)) {
    logSecurityEvent('authorization_failed', {
      userId: req.user.id,
      requiredRole,
      userRoles: req.user.roles,
      resource: req.path,
      method: req.method
    })
    return res.status(403).json({ error: 'Forbidden' })
  }
  next()
}

// Sensitive operations
app.delete('/api/users/:id', authenticate, authorize('admin'), async (req, res) => {
  logSecurityEvent('user_deleted', {
    deletedBy: req.user.id,
    deletedUser: req.params.id
  })
  // ... delete logic
})

// Password changes
app.post('/api/auth/change-password', authenticate, async (req, res) => {
  // ... password change logic

  logSecurityEvent('password_changed', {
    userId: req.user.id,
    ip: req.ip
  })
})

// Admin actions
app.post('/api/admin/users/:id/role', authenticate, authorize('admin'), async (req, res) => {
  const { role } = req.body

  logSecurityEvent('role_changed', {
    adminId: req.user.id,
    userId: req.params.id,
    newRole: role
  })

  // ... role change logic
})
```

#### Log Analysis & Alerting

```javascript
// Monitor for suspicious activity
const detectBruteForce = async (email) => {
  const recentFailures = await getLoginFailures(email, {
    since: Date.now() - 15 * 60 * 1000 // Last 15 minutes
  })

  if (recentFailures.length >= 5) {
    logSecurityEvent('brute_force_detected', {
      email,
      attemptCount: recentFailures.length
    })

    // Send alert to security team
    await sendAlert({
      type: 'brute_force',
      email,
      attempts: recentFailures.length
    })

    // Temporarily lock account
    await lockAccount(email, 30 * 60 * 1000) // 30 minutes
  }
}

// Monitor for privilege escalation attempts
const detectPrivilegeEscalation = (userId) => {
  const recentAuthFailures = await getAuthorizationFailures(userId, {
    since: Date.now() - 60 * 60 * 1000 // Last hour
  })

  if (recentAuthFailures.length >= 10) {
    logSecurityEvent('privilege_escalation_detected', {
      userId,
      attemptCount: recentAuthFailures.length
    })

    await sendAlert({
      type: 'privilege_escalation',
      userId,
      attempts: recentAuthFailures.length
    })
  }
}
```

---

### A10:2021 - Server-Side Request Forgery (Extended)

#### SSRF Prevention

```javascript
const axios = require('axios')
const { URL } = require('url')

// Allowlist of safe domains
const ALLOWED_DOMAINS = [
  'api.example.com',
  'cdn.example.com'
]

// IP address blocklist
const BLOCKED_IPS = [
  '127.0.0.1',
  '0.0.0.0',
  'localhost',
  // Private IP ranges
  /^10\./,
  /^172\.(1[6-9]|2[0-9]|3[01])\./,
  /^192\.168\./,
  // Link-local
  /^169\.254\./,
  // Loopback
  /^127\./,
  /^::1$/,
  /^fe80:/
]

const isValidUrl = (urlString) => {
  try {
    const url = new URL(urlString)

    // Only allow HTTP/HTTPS
    if (!['http:', 'https:'].includes(url.protocol)) {
      return false
    }

    // Check domain allowlist
    if (!ALLOWED_DOMAINS.includes(url.hostname)) {
      return false
    }

    // Check IP blocklist
    const isBlocked = BLOCKED_IPS.some(pattern => {
      if (pattern instanceof RegExp) {
        return pattern.test(url.hostname)
      }
      return url.hostname === pattern
    })

    return !isBlocked
  } catch {
    return false
  }
}

// VULNERABLE: SSRF possible
app.get('/api/fetch', async (req, res) => {
  const { url } = req.query
  const response = await axios.get(url)
  res.json(response.data)
})
// Attack: /api/fetch?url=http://localhost:8080/admin

// SECURE: Validate and restrict URLs
app.get('/api/fetch', async (req, res) => {
  const { url } = req.query

  if (!isValidUrl(url)) {
    return res.status(400).json({ error: 'Invalid or disallowed URL' })
  }

  try {
    const response = await axios.get(url, {
      timeout: 5000,
      maxRedirects: 0 // Prevent redirect attacks
    })
    res.json(response.data)
  } catch (error) {
    logSecurityEvent('ssrf_attempt_blocked', {
      url,
      ip: req.ip
    })
    res.status(400).json({ error: 'Request failed' })
  }
})
```

---

## Language-Specific Security Considerations

### Node.js / JavaScript

- Use `npm audit` regularly
- Avoid `eval()`, `Function()`, and `new Function()`
- Use strict mode (`'use strict'`)
- Validate regex to prevent ReDoS attacks
- Use `crypto.timingSafeEqual()` for secret comparison
- Avoid `JSON.parse()` on untrusted input without validation

### Python

- Use `secrets` module for tokens (not `random`)
- Avoid `eval()`, `exec()`, and `pickle.loads()` on untrusted input
- Use parameterized queries with SQLAlchemy or similar
- Validate file paths with `os.path.normpath()` and check prefix
- Use `hashlib.compare_digest()` for secret comparison

### Go

- Run `govulncheck ./...` regularly
- Avoid `os.Exec` with user input
- Use prepared statements for database queries
- Validate file paths before operations
- Use `subtle.ConstantTimeCompare()` for secret comparison
- Enable race detection in tests: `go test -race`

### Rust

- Run `cargo audit` regularly
- Use `sqlx` for compile-time checked SQL
- Avoid `unsafe` blocks without thorough review
- Use `constant_time_eq` for secret comparison
- Validate all `unsafe` code in security review

---

## Advanced Threat Modeling

### Attack Surface Analysis

**External Attack Surface**:
- Public APIs
- Web forms
- File uploads
- WebSocket endpoints
- OAuth callbacks
- Webhooks

**Internal Attack Surface**:
- Admin panels
- Internal APIs
- Database queries
- File system operations
- Inter-service communication

### Attack Tree Example

```
Goal: Compromise User Account
├── Brute Force Password
│   ├── No rate limiting
│   ├── Weak password policy
│   └── No account lockout
├── Steal Session Token
│   ├── XSS vulnerability
│   ├── Session fixation
│   └── Token in URL
├── Password Reset Bypass
│   ├── Predictable reset tokens
│   ├── No expiration
│   └── Email enumeration
└── SQL Injection
    ├── Unsanitized input
    ├── String concatenation
    └── ORM bypass
```

---

## Compliance Checklists

### GDPR Compliance

- [ ] User consent for data collection
- [ ] Right to access data
- [ ] Right to delete data (right to be forgotten)
- [ ] Data portability
- [ ] Privacy policy
- [ ] Data breach notification (72 hours)
- [ ] Data encryption at rest and in transit
- [ ] Minimize data collection (data minimization)

### PCI DSS (Payment Card Industry)

- [ ] Secure network (firewall, no default passwords)
- [ ] Protect cardholder data (encryption)
- [ ] Vulnerability management program
- [ ] Access control (need-to-know basis)
- [ ] Network monitoring and testing
- [ ] Information security policy
- [ ] Never store CVV/CVC
- [ ] Tokenize card numbers

### SOC 2

- [ ] Security policies documented
- [ ] Access controls implemented
- [ ] Encryption for data at rest and in transit
- [ ] Vulnerability scanning
- [ ] Incident response plan
- [ ] Backup and recovery procedures
- [ ] Vendor management
- [ ] Annual security training

---

## Security Testing Tools

### Static Analysis

- **JavaScript/TypeScript**: ESLint with security plugins, SonarQube
- **Python**: Bandit, safety, pylint
- **Go**: gosec, staticcheck
- **Rust**: cargo-audit, cargo-clippy

### Dynamic Analysis

- **OWASP ZAP**: Web application security scanner
- **Burp Suite**: Web vulnerability scanner
- **SQLMap**: SQL injection testing
- **Nmap**: Network scanning

### Dependency Scanning

- **Snyk**: Dependency vulnerability scanning
- **Dependabot**: Automated dependency updates
- **npm audit**: Node.js dependencies
- **pip-audit**: Python dependencies
- **cargo-audit**: Rust dependencies

---

## Incident Response Plan

### Detection

1. Monitor logs for security events
2. Set up alerts for suspicious activity
3. Review failed authentication attempts
4. Track authorization failures
5. Monitor for unusual API usage

### Containment

1. Identify affected systems
2. Isolate compromised resources
3. Preserve evidence (logs, snapshots)
4. Block malicious IPs/users
5. Revoke compromised credentials

### Eradication

1. Identify root cause
2. Remove malware/backdoors
3. Patch vulnerabilities
4. Update compromised credentials
5. Review access controls

### Recovery

1. Restore from clean backups
2. Verify system integrity
3. Monitor for re-infection
4. Re-enable affected services
5. Communicate with stakeholders

### Lessons Learned

1. Document incident timeline
2. Identify detection gaps
3. Update security controls
4. Improve monitoring
5. Conduct post-mortem review
6. Update incident response plan

---

**Remember**: Security is an ongoing process, not a one-time checklist. Regular audits, continuous monitoring, and staying updated on new vulnerabilities are essential for maintaining a secure application.
