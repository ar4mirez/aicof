# Shell Patterns Reference

## CLI Script Template

```bash
#!/usr/bin/env bash
set -euo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_NAME="$(basename "${BASH_SOURCE[0]}")"
VERBOSE=false

usage() { cat <<EOF
Usage: $SCRIPT_NAME [OPTIONS] <target>
  -h, --help       Show help
  -v, --verbose    Verbose output
  -n, --dry-run    Show what would be done
EOF
}

log_info()  { printf "[INFO]  %s\n" "$*"; }
log_error() { printf "[ERROR] %s\n" "$*" >&2; }
log_debug() { [[ "$VERBOSE" == true ]] && printf "[DEBUG] %s\n" "$*"; }

cleanup() { local rc=$?; rm -f "${tmpfile:-}"; exit "$rc"; }
trap cleanup EXIT

main() {
    local target=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -h|--help)    usage; exit 0 ;;
            -v|--verbose) VERBOSE=true; shift ;;
            -n|--dry-run) DRY_RUN=true; shift ;;
            -*)           log_error "Unknown option: $1"; exit 2 ;;
            *)            target="$1"; shift ;;
        esac
    done
    [[ -z "$target" ]] && { log_error "Missing <target>"; usage; exit 2; }
    log_info "Running: $target"
}

main "$@"
```

## Colored Logging (NO_COLOR-aware)

```bash
if [[ -t 2 ]] && [[ -z "${NO_COLOR:-}" ]]; then
    readonly RED=$'\033[31m' GREEN=$'\033[32m' YELLOW=$'\033[33m' RESET=$'\033[0m'
else
    readonly RED="" GREEN="" YELLOW="" RESET=""
fi
log_info()  { printf "%s[INFO]%s  %s\n" "$GREEN" "$RESET" "$*"; }
log_warn()  { printf "%s[WARN]%s  %s\n" "$YELLOW" "$RESET" "$*" >&2; }
log_error() { printf "%s[ERROR]%s %s\n" "$RED" "$RESET" "$*" >&2; }
```

## Trap Patterns

```bash
# Multi-resource cleanup
TMPFILE="" BG_PID=""
cleanup() {
    local rc=$?
    [[ -n "$BG_PID" ]]  && kill "$BG_PID" 2>/dev/null && wait "$BG_PID" 2>/dev/null
    [[ -n "$TMPFILE" ]] && rm -f "$TMPFILE"
    exit "$rc"
}
trap cleanup EXIT
trap 'echo "Interrupted" >&2; exit 130' INT TERM

# Debug: print failing line on error
trap 'echo "Error on line $LINENO: \"$BASH_COMMAND\" exited $?" >&2' ERR
```

## Portable Scripting

```bash
# Cross-platform path resolution (macOS + Linux)
resolve_path() {
    if command -v realpath &>/dev/null; then realpath "$1"
    elif command -v greadlink &>/dev/null; then greadlink -f "$1"
    else (cd "$(dirname "$1")" && echo "$(pwd)/$(basename "$1")")
    fi
}

# Portable sed in-place (BSD vs GNU)
sed_inplace() {
    if sed --version 2>/dev/null | grep -q GNU; then sed -i "$@"
    else sed -i '' "$@"
    fi
}

# Portable checksum
sha256() {
    if command -v sha256sum &>/dev/null; then sha256sum "$1" | cut -d' ' -f1
    elif command -v shasum &>/dev/null; then shasum -a 256 "$1" | cut -d' ' -f1
    else log_error "No SHA-256 tool found"; return 1
    fi
}
```

## Retry with Backoff

```bash
retry() {
    local max="$1" delay="$2"; shift 2
    local attempt=1
    while (( attempt <= max )); do
        "$@" && return 0
        (( attempt == max )) && { log_error "Failed after $max attempts: $*"; return 1; }
        log_warn "Attempt $attempt/$max failed, retry in ${delay}s..."
        sleep "$delay"; delay=$(( delay * 2 )); attempt=$(( attempt + 1 ))
    done
}
# Usage: retry 5 2 curl --fail --silent "https://api.example.com/health"
```

## Argument Parsing (getopts, POSIX-compatible)

```bash
parse_args() {
    local OPTIND opt
    while getopts ":hvo:" opt; do
        case "$opt" in
            h) usage; exit 0 ;;
            v) VERBOSE=true ;;
            o) OUTPUT_DIR="$OPTARG" ;;
            :) log_error "Option -$OPTARG requires an argument"; exit 2 ;;
            ?) log_error "Unknown option: -$OPTARG"; exit 2 ;;
        esac
    done
    shift $((OPTIND - 1))
}
```
