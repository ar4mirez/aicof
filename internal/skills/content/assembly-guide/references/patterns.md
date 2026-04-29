# Assembly Patterns Reference

## x86-64 Function Prologue/Epilogue

### Recursive Function (Callee-Saved Registers)

```nasm
; long factorial(long n)
; Args: rdi = n  |  Returns: rax = n!
global factorial
factorial:
    push    rbp                     ; --- prologue ---
    mov     rbp, rsp
    push    rbx                     ; save callee-saved register
    mov     rbx, rdi                ; rbx = n (preserved across call)
    cmp     rbx, 1
    jle     .base_case
    lea     rdi, [rbx - 1]          ; arg0 = n - 1
    call    factorial               ; rax = factorial(n - 1)
    imul    rax, rbx                ; rax = n * factorial(n - 1)
    jmp     .epilogue
.base_case:
    mov     rax, 1
.epilogue:
    pop     rbx                     ; --- epilogue (reverse order) ---
    pop     rbp
    ret
```

## Linux Syscall Examples

### File Open / Read / Close

```nasm
SYS_OPEN equ 2
SYS_READ equ 0
SYS_CLOSE equ 3

section .bss
    buffer resb 4096
section .data
    filepath db "/etc/hostname", 0

; ssize_t read_file(void) -- Returns bytes read (negative = error)
global read_file
read_file:
    push    rbp
    mov     rbp, rsp
    push    rbx
    mov     rax, SYS_OPEN           ; open(filepath, O_RDONLY)
    lea     rdi, [rel filepath]
    xor     esi, esi
    xor     edx, edx
    syscall
    test    rax, rax
    js      .done                   ; error -> return negative
    mov     rbx, rax                ; save fd in callee-saved reg
    mov     rax, SYS_READ           ; read(fd, buffer, 4096)
    mov     rdi, rbx
    lea     rsi, [rel buffer]
    mov     rdx, 4096
    syscall
    push    rax
    mov     rax, SYS_CLOSE
    mov     rdi, rbx
    syscall
    pop     rax                     ; return bytes_read
.done:
    pop     rbx
    pop     rbp
    ret
```

## SIMD Patterns

### SSE2 Horizontal Sum (int32, 4 elements/iteration)

```nasm
; int sum_i32(const int32_t *arr, size_t n)
; Args: rdi = arr, rsi = n  |  Returns: eax
global sum_i32
sum_i32:
    pxor    xmm0, xmm0             ; accumulator = [0,0,0,0]
    mov     rcx, rsi
    shr     rcx, 2
    jz      .reduce
.vec:
    movdqu  xmm1, [rdi]            ; load 4 ints
    paddd   xmm0, xmm1             ; packed add
    add     rdi, 16
    dec     rcx
    jnz     .vec
.reduce:                            ; horizontal sum across 4 lanes
    pshufd  xmm1, xmm0, 0x4E       ; swap pairs
    paddd   xmm0, xmm1
    pshufd  xmm1, xmm0, 0x11
    paddd   xmm0, xmm1
    movd    eax, xmm0              ; extract scalar
    and     esi, 3                  ; handle remainder
    jz      .done
.tail:
    add     eax, [rdi]
    add     rdi, 4
    dec     esi
    jnz     .tail
.done:
    ret
```

### AVX2 Scaling (ymm registers, 4 doubles/iteration)

```nasm
; void scale_f64(double *arr, double scalar, size_t n)
; Args: rdi = arr, xmm0 = scalar, rsi = n
global scale_f64
scale_f64:
    vbroadcastsd ymm1, xmm0        ; broadcast scalar to 4 lanes
    shr     rsi, 2
.loop:
    test    rsi, rsi
    jz      .done
    vmovupd ymm0, [rdi]            ; load 4 doubles
    vmulpd  ymm0, ymm0, ymm1       ; scale
    vmovupd [rdi], ymm0            ; store
    add     rdi, 32
    dec     rsi
    jnz     .loop
.done:
    vzeroupper                      ; required after AVX to avoid penalties
    ret
```

### Performance Notes

- Handle scalar remainder after SIMD loops (n % vector_width)
- Use `vzeroupper` after AVX code to avoid SSE/AVX transition penalties
- Prefer `movdqu`/`vmovdqu` unless alignment is guaranteed
- Use `lock` prefix for atomic ops; profile with `perf stat` to verify
