# C++ Patterns Reference

## Contents

- [RAII File Wrapper](#raii-file-wrapper)
- [RAII Socket Wrapper](#raii-socket-wrapper)
- [RAII Read-Write Lock](#raii-read-write-lock)
- [Smart Pointer Factory](#smart-pointer-factory)
- [Pimpl Idiom](#pimpl-idiom)
- [Thread-Safe Singleton](#thread-safe-singleton)
- [Pattern Guardrails](#pattern-guardrails)

## RAII File Wrapper

```cpp
class ScopedTempFile {
public:
    explicit ScopedTempFile(std::filesystem::path path)
        : path_(std::move(path)) {
        stream_.open(path_, std::ios::out | std::ios::trunc);
        if (!stream_.is_open()) {
            throw std::runtime_error("Cannot create temp file: " + path_.string());
        }
    }

    ~ScopedTempFile() {
        stream_.close();
        std::error_code ec;
        std::filesystem::remove(path_, ec);  // noexcept cleanup
    }

    ScopedTempFile(const ScopedTempFile&) = delete;
    ScopedTempFile& operator=(const ScopedTempFile&) = delete;

    std::ofstream& stream() { return stream_; }
    const std::filesystem::path& path() const { return path_; }

private:
    std::filesystem::path path_;
    std::ofstream stream_;
};

// Usage: file is deleted when scope exits, even on exception
void export_report(const Report& report) {
    ScopedTempFile tmp("/tmp/report_staging.csv");
    tmp.stream() << report.to_csv();
}  // tmp file automatically deleted here
```

## RAII Socket Wrapper

```cpp
class Socket {
public:
    explicit Socket(int domain, int type, int protocol)
        : fd_(::socket(domain, type, protocol)) {
        if (fd_ < 0) throw std::runtime_error("socket() failed");
    }

    ~Socket() { close(); }

    Socket(const Socket&) = delete;
    Socket& operator=(const Socket&) = delete;
    Socket(Socket&& other) noexcept : fd_(std::exchange(other.fd_, -1)) {}
    Socket& operator=(Socket&& other) noexcept {
        if (this != &other) { close(); fd_ = std::exchange(other.fd_, -1); }
        return *this;
    }

    int fd() const noexcept { return fd_; }

private:
    void close() noexcept {
        if (fd_ >= 0) { ::close(fd_); fd_ = -1; }
    }
    int fd_;
};
```

## RAII Read-Write Lock

```cpp
// Read-write lock: many concurrent readers, exclusive writer
class ThreadSafeCache {
public:
    std::optional<std::string> get(const std::string& key) const {
        std::shared_lock lock(mutex_);  // shared: multiple readers
        auto it = data_.find(key);
        if (it == data_.end()) return std::nullopt;
        return it->second;
    }

    void put(const std::string& key, std::string value) {
        std::unique_lock lock(mutex_);  // exclusive: single writer
        data_[key] = std::move(value);
    }

private:
    mutable std::shared_mutex mutex_;
    std::unordered_map<std::string, std::string> data_;
};
```

## Smart Pointer Factory

```cpp
class Shape {
public:
    virtual ~Shape() = default;
    virtual double area() const = 0;
};

class Circle : public Shape {
public:
    explicit Circle(double radius) : radius_(radius) {}
    double area() const override { return 3.14159265 * radius_ * radius_; }
private:
    double radius_;
};

class Rectangle : public Shape {
public:
    Rectangle(double w, double h) : width_(w), height_(h) {}
    double area() const override { return width_ * height_; }
private:
    double width_, height_;
};

// Factory returns unique_ptr: caller owns the result
std::unique_ptr<Shape> create_shape(std::string_view type, double a, double b = 0) {
    if (type == "circle")    return std::make_unique<Circle>(a);
    if (type == "rectangle") return std::make_unique<Rectangle>(a, b);
    return nullptr;
}
```

## Pimpl Idiom

```cpp
// widget.hpp -- public header, hides implementation details
class Widget {
public:
    explicit Widget(std::string name);
    ~Widget();  // must be declared here, defined in .cpp

    Widget(Widget&&) noexcept;
    Widget& operator=(Widget&&) noexcept;

    void draw() const;
    const std::string& name() const;

private:
    struct Impl;
    std::unique_ptr<Impl> impl_;
};

// widget.cpp -- implementation details hidden from consumers
struct Widget::Impl {
    std::string name;
    int render_count = 0;
};

Widget::Widget(std::string name) : impl_(std::make_unique<Impl>()) {
    impl_->name = std::move(name);
}

Widget::~Widget() = default;
Widget::Widget(Widget&&) noexcept = default;
Widget& Widget::operator=(Widget&&) noexcept = default;

void Widget::draw() const { ++impl_->render_count; }
const std::string& Widget::name() const { return impl_->name; }
```

## Thread-Safe Singleton

```cpp
class Logger {
public:
    // Meyer's Singleton: thread-safe in C++11+ (guaranteed by the standard)
    static Logger& instance() {
        static Logger logger;
        return logger;
    }

    Logger(const Logger&) = delete;
    Logger& operator=(const Logger&) = delete;
    Logger(Logger&&) = delete;
    Logger& operator=(Logger&&) = delete;

    void log(std::string_view message) {
        std::lock_guard lock(mutex_);
        // write to file, stdout, or buffer
    }

private:
    Logger() = default;
    ~Logger() = default;
    std::mutex mutex_;
};

// Usage
Logger::instance().log("application started");
```

## Pattern Guardrails

- Every RAII wrapper must be non-copyable or explicitly define copy semantics
- Move constructors and move assignment operators must be `noexcept`
- Use `std::exchange` in move constructors to leave moved-from objects valid
- Prefer `std::make_unique` / `std::make_shared` over raw `new` in all factories
- Singletons should use Meyer's Singleton (local static) not double-checked locking
- Always define the Pimpl destructor in the `.cpp` file (where `Impl` is complete)
