# Go Security Best Practices

## Guardrails

- Use parameterized queries (prevents SQL injection)
- Validate all user inputs
- Use `crypto/rand` for random numbers (not `math/rand`)
- Hash passwords with `bcrypt` or `argon2`
- Use HTTPS (TLS) for production
- Set secure headers (CORS, CSP, etc.)
- Run `gosec` to detect security issues

## Password Hashing

```go
import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
    return string(bytes), err
}

func CheckPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```
