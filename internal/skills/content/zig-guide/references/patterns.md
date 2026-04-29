# Zig Patterns Reference

## Contents

- [Arena Allocator](#arena-allocator)
- [General Purpose Allocator with Leak Detection](#general-purpose-allocator-with-leak-detection)
- [Comptime Generic Data Structure](#comptime-generic-data-structure)
- [Comptime Interface Validation](#comptime-interface-validation)
- [C Library Wrapper](#c-library-wrapper)
- [Sentinel-Terminated String Conversion](#sentinel-terminated-string-conversion)
- [Tagged Union State Machine](#tagged-union-state-machine)
- [Resource Cleanup with Errdefer Chain](#resource-cleanup-with-errdefer-chain)
- [build.zig Minimal Template](#buildzig-minimal-template)

## Arena Allocator

Use an arena when many allocations share a lifetime and can be freed together.

```zig
fn processRequest(backing_allocator: std.mem.Allocator, raw: []const u8) !Response {
    var arena = std.heap.ArenaAllocator.init(backing_allocator);
    defer arena.deinit(); // frees everything at once

    const alloc = arena.allocator();
    const parsed = try parseHeaders(alloc, raw);
    const body = try decodeBody(alloc, parsed.body_raw);
    return Response.from(try validatePayload(alloc, body));
}
```

## General Purpose Allocator with Leak Detection

Use GPA during development and testing to catch memory bugs.

```zig
pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{ .stack_trace_frames = 8 }){};
    defer {
        const check = gpa.deinit();
        if (check == .leak) std.log.err("memory leak detected", .{});
    }
    try runApp(gpa.allocator());
}
```

## Comptime Generic Data Structure

Generate specialized types at compile time with full type safety.

```zig
pub fn BoundedQueue(comptime T: type, comptime capacity: usize) type {
    return struct {
        const Self = @This();
        items: [capacity]T = undefined,
        head: usize = 0,
        tail: usize = 0,
        len: usize = 0,

        pub fn push(self: *Self, value: T) error{QueueFull}!void {
            if (self.len == capacity) return error.QueueFull;
            self.items[self.tail] = value;
            self.tail = (self.tail + 1) % capacity;
            self.len += 1;
        }

        pub fn pop(self: *Self) ?T {
            if (self.len == 0) return null;
            const value = self.items[self.head];
            self.head = (self.head + 1) % capacity;
            self.len -= 1;
            return value;
        }
    };
}

test "bounded queue push and pop" {
    var q = BoundedQueue(u32, 4){};
    try q.push(10);
    try std.testing.expectEqual(@as(?u32, 10), q.pop());
    try std.testing.expectEqual(@as(?u32, null), q.pop());
}
```

## Comptime Interface Validation

Enforce duck-typed interfaces at compile time with clear error messages.

```zig
fn Serializable(comptime T: type) void {
    if (!@hasDecl(T, "serialize"))
        @compileError(@typeName(T) ++ " must implement serialize");
    if (!@hasDecl(T, "deserialize"))
        @compileError(@typeName(T) ++ " must implement deserialize");
}

pub fn writeToFile(comptime T: type, value: T, path: []const u8) !void {
    comptime Serializable(T);
    const file = try std.fs.cwd().createFile(path, .{});
    defer file.close();
    try value.serialize(file.writer());
}
```

## C Library Wrapper

Wrap a C library with safe Zig types at the FFI boundary.

```zig
const c = @cImport({ @cInclude("sqlite3.h"); });

pub const Database = struct {
    handle: *c.sqlite3,

    pub fn open(path: [:0]const u8) !Database {
        var db: ?*c.sqlite3 = null;
        if (c.sqlite3_open(path.ptr, &db) != c.SQLITE_OK) {
            if (db) |d| _ = c.sqlite3_close(d);
            return error.CannotOpen;
        }
        return .{ .handle = db.? };
    }

    pub fn close(self: *Database) void {
        _ = c.sqlite3_close(self.handle);
    }

    pub fn exec(self: *Database, sql: [:0]const u8) !void {
        var err_msg: [*c]u8 = null;
        if (c.sqlite3_exec(self.handle, sql.ptr, null, null, &err_msg) != c.SQLITE_OK) {
            if (err_msg) |msg| {
                std.log.err("sqlite: {s}", .{std.mem.span(msg)});
                c.sqlite3_free(msg);
            }
            return error.ExecFailed;
        }
    }
};
```

## Sentinel-Terminated String Conversion

Convert between Zig slices and C strings safely.

```zig
const std = @import("std");

/// Converts a Zig string slice to a null-terminated C string.
/// Caller owns the returned memory.
fn toCString(allocator: std.mem.Allocator, slice: []const u8) ![:0]u8 {
    return allocator.dupeZ(u8, slice);
}

/// Converts a C string to a Zig slice (no allocation, borrows pointer).
fn fromCString(c_str: [*:0]const u8) []const u8 {
    return std.mem.span(c_str);
}
```

## Tagged Union State Machine

Model finite states with tagged unions for exhaustive switch checking.

```zig
const Connection = union(enum) {
    disconnected: void,
    connecting: struct { attempts: u32, last_error: ?anyerror },
    connected: struct { socket: std.net.Stream },
    closing: struct { socket: std.net.Stream, reason: []const u8 },

    pub fn tick(self: *Connection, allocator: std.mem.Allocator) !void {
        switch (self.*) {
            .disconnected => {
                self.* = .{ .connecting = .{ .attempts = 1, .last_error = null } };
            },
            .connecting => |*state| {
                const socket = std.net.tcpConnectToHost(allocator, "localhost", 8080) catch |err| {
                    state.attempts += 1;
                    state.last_error = err;
                    if (state.attempts > 3) return error.MaxRetriesExceeded;
                    return;
                };
                self.* = .{ .connected = .{ .socket = socket } };
            },
            .connected => {},
            .closing => |state| {
                state.socket.close();
                self.* = .disconnected;
            },
        }
    }
};
```

## Resource Cleanup with Errdefer Chain

Properly handle partial initialization with chained errdefer.

```zig
const Server = struct {
    listener: std.net.Server,
    db: Database,
    allocator: std.mem.Allocator,

    pub fn init(allocator: std.mem.Allocator, config: Config) !Server {
        var db = try Database.open(config.db_path);
        errdefer db.close(); // closes DB if listener setup fails

        const address = try std.net.Address.parseIp(config.host, config.port);
        var listener = try address.listen(.{ .reuse_address = true });
        errdefer listener.deinit(); // closes listener if subsequent init fails

        try db.exec("PRAGMA journal_mode=WAL;");

        return .{
            .listener = listener,
            .db = db,
            .allocator = allocator,
        };
    }

    pub fn deinit(self: *Server) void {
        self.listener.deinit();
        self.db.close();
    }
};
```

## build.zig Minimal Template

Standard build.zig for a Zig project with executable and tests.

```zig
const std = @import("std");
pub fn build(b: *std.Build) void {
    const target = b.standardTargetOptions(.{});
    const optimize = b.standardOptimizeOption(.{});

    const exe = b.addExecutable(.{
        .name = "myapp", .root_source_file = b.path("src/main.zig"),
        .target = target, .optimize = optimize,
    });
    b.installArtifact(exe);
    b.step("run", "Run the application").dependOn(&b.addRunArtifact(exe).step);

    const tests = b.addTest(.{
        .root_source_file = b.path("src/root.zig"),
        .target = target, .optimize = optimize,
    });
    b.step("test", "Run unit tests").dependOn(&b.addRunArtifact(tests).step);
}
```
