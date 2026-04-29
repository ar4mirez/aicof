# React Patterns Reference

## Contents

- [Compound Components](#compound-components)
- [Custom Hooks](#custom-hooks)
- [Data Fetching Patterns](#data-fetching-patterns)
- [Form Patterns](#form-patterns)
- [Testing Patterns](#testing-patterns)
- [Performance Optimization](#performance-optimization)
- [Error Handling](#error-handling)
- [Security Patterns](#security-patterns)
- [Accessibility Patterns](#accessibility-patterns)

## Compound Components

Use compound components when a group of components share implicit state and must be used together.

```tsx
interface TabsContextValue {
  activeTab: string;
  setActiveTab: (tab: string) => void;
}

const TabsContext = createContext<TabsContextValue | null>(null);

function Tabs({ children, defaultTab }: { children: React.ReactNode; defaultTab: string }) {
  const [activeTab, setActiveTab] = useState(defaultTab);

  return (
    <TabsContext.Provider value={{ activeTab, setActiveTab }}>
      <div className="tabs">{children}</div>
    </TabsContext.Provider>
  );
}

function TabList({ children }: { children: React.ReactNode }) {
  return <div className="tab-list" role="tablist">{children}</div>;
}

function Tab({ value, children }: { value: string; children: React.ReactNode }) {
  const context = useContext(TabsContext);
  if (!context) throw new Error('Tab must be used within Tabs');

  return (
    <button
      role="tab"
      aria-selected={context.activeTab === value}
      onClick={() => context.setActiveTab(value)}
    >
      {children}
    </button>
  );
}

function TabPanel({ value, children }: { value: string; children: React.ReactNode }) {
  const context = useContext(TabsContext);
  if (!context) throw new Error('TabPanel must be used within Tabs');

  if (context.activeTab !== value) return null;
  return <div role="tabpanel">{children}</div>;
}

Tabs.List = TabList;
Tabs.Tab = Tab;
Tabs.Panel = TabPanel;

export { Tabs };

// Usage
<Tabs defaultTab="profile">
  <Tabs.List>
    <Tabs.Tab value="profile">Profile</Tabs.Tab>
    <Tabs.Tab value="settings">Settings</Tabs.Tab>
  </Tabs.List>
  <Tabs.Panel value="profile">Profile content</Tabs.Panel>
  <Tabs.Panel value="settings">Settings content</Tabs.Panel>
</Tabs>
```

## Custom Hooks

### useQuery (Data Fetching)

```tsx
interface UseQueryResult<T> {
  data: T | null;
  loading: boolean;
  error: Error | null;
  refetch: () => Promise<void>;
}

function useQuery<T>(fetcher: () => Promise<T>, deps: unknown[] = []): UseQueryResult<T> {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetch = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await fetcher();
      setData(result);
    } catch (e) {
      setError(e instanceof Error ? e : new Error('Unknown error'));
    } finally {
      setLoading(false);
    }
  }, deps);

  useEffect(() => {
    fetch();
  }, [fetch]);

  return { data, loading, error, refetch: fetch };
}

// Usage
const { data: users, loading, error, refetch } = useQuery(
  () => userService.getAll(),
  []
);
```

### useLocalStorage

```tsx
function useLocalStorage<T>(key: string, initialValue: T) {
  const [storedValue, setStoredValue] = useState<T>(() => {
    try {
      const item = window.localStorage.getItem(key);
      return item ? JSON.parse(item) : initialValue;
    } catch {
      return initialValue;
    }
  });

  const setValue = useCallback((value: T | ((val: T) => T)) => {
    try {
      const valueToStore = value instanceof Function ? value(storedValue) : value;
      setStoredValue(valueToStore);
      window.localStorage.setItem(key, JSON.stringify(valueToStore));
    } catch (error) {
      console.error('Error saving to localStorage:', error);
    }
  }, [key, storedValue]);

  return [storedValue, setValue] as const;
}
```

### useDebounce

```tsx
function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState<T>(value);

  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);

    return () => {
      clearTimeout(handler);
    };
  }, [value, delay]);

  return debouncedValue;
}

// Usage
const [searchTerm, setSearchTerm] = useState('');
const debouncedSearch = useDebounce(searchTerm, 300);

useEffect(() => {
  if (debouncedSearch) {
    searchUsers(debouncedSearch);
  }
}, [debouncedSearch]);
```

### useMediaQuery

```tsx
function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(() =>
    typeof window !== 'undefined' ? window.matchMedia(query).matches : false
  );

  useEffect(() => {
    const mediaQuery = window.matchMedia(query);
    const handler = (event: MediaQueryListEvent) => setMatches(event.matches);

    mediaQuery.addEventListener('change', handler);
    return () => mediaQuery.removeEventListener('change', handler);
  }, [query]);

  return matches;
}

// Usage
const isMobile = useMediaQuery('(max-width: 768px)');
const prefersDark = useMediaQuery('(prefers-color-scheme: dark)');
```

### useClickOutside

```tsx
function useClickOutside(ref: React.RefObject<HTMLElement>, handler: () => void) {
  useEffect(() => {
    const listener = (event: MouseEvent | TouchEvent) => {
      if (!ref.current || ref.current.contains(event.target as Node)) return;
      handler();
    };

    document.addEventListener('mousedown', listener);
    document.addEventListener('touchstart', listener);
    return () => {
      document.removeEventListener('mousedown', listener);
      document.removeEventListener('touchstart', listener);
    };
  }, [ref, handler]);
}

// Usage
const dropdownRef = useRef<HTMLDivElement>(null);
useClickOutside(dropdownRef, () => setIsOpen(false));
```

## Data Fetching Patterns

### TanStack Query with Pagination

```tsx
function usePaginatedUsers(page: number) {
  return useQuery({
    queryKey: ['users', page],
    queryFn: () => userService.getAll({ page, limit: 20 }),
    placeholderData: keepPreviousData,
    staleTime: 5 * 60 * 1000,
  });
}
```

### Optimistic Updates with TanStack Query

```tsx
function useUpdateUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: UpdateUserDTO) => userService.update(data.id, data),
    onMutate: async (newData) => {
      await queryClient.cancelQueries({ queryKey: ['users', newData.id] });
      const previousUser = queryClient.getQueryData(['users', newData.id]);

      queryClient.setQueryData(['users', newData.id], (old: User) => ({
        ...old,
        ...newData,
      }));

      return { previousUser };
    },
    onError: (_err, _newData, context) => {
      queryClient.setQueryData(
        ['users', context?.previousUser?.id],
        context?.previousUser
      );
    },
    onSettled: (data) => {
      queryClient.invalidateQueries({ queryKey: ['users', data?.id] });
    },
  });
}
```

### Infinite Scroll

```tsx
function useInfiniteUsers() {
  return useInfiniteQuery({
    queryKey: ['users', 'infinite'],
    queryFn: ({ pageParam = 0 }) =>
      userService.getAll({ offset: pageParam, limit: 20 }),
    getNextPageParam: (lastPage, allPages) =>
      lastPage.hasMore ? allPages.length * 20 : undefined,
    initialPageParam: 0,
  });
}

function UserInfiniteList() {
  const { data, fetchNextPage, hasNextPage, isFetchingNextPage } =
    useInfiniteUsers();

  const observerRef = useRef<IntersectionObserver>();
  const lastElementRef = useCallback(
    (node: HTMLElement | null) => {
      if (isFetchingNextPage) return;
      if (observerRef.current) observerRef.current.disconnect();

      observerRef.current = new IntersectionObserver((entries) => {
        if (entries[0].isIntersecting && hasNextPage) {
          fetchNextPage();
        }
      });

      if (node) observerRef.current.observe(node);
    },
    [isFetchingNextPage, fetchNextPage, hasNextPage]
  );

  return (
    <div>
      {data?.pages.map((page, i) =>
        page.users.map((user, j) => {
          const isLast =
            i === data.pages.length - 1 && j === page.users.length - 1;
          return (
            <UserCard
              key={user.id}
              ref={isLast ? lastElementRef : null}
              user={user}
            />
          );
        })
      )}
      {isFetchingNextPage && <Spinner />}
    </div>
  );
}
```

## Form Patterns

### Multi-Step Form

```tsx
interface FormStep {
  title: string;
  component: React.ComponentType<StepProps>;
  validation: z.ZodSchema;
}

function MultiStepForm({ steps, onComplete }: {
  steps: FormStep[];
  onComplete: (data: Record<string, unknown>) => void;
}) {
  const [currentStep, setCurrentStep] = useState(0);
  const [formData, setFormData] = useState<Record<string, unknown>>({});

  const handleStepSubmit = (stepData: Record<string, unknown>) => {
    const updatedData = { ...formData, ...stepData };
    setFormData(updatedData);

    if (currentStep === steps.length - 1) {
      onComplete(updatedData);
    } else {
      setCurrentStep((prev) => prev + 1);
    }
  };

  const StepComponent = steps[currentStep].component;

  return (
    <div>
      <StepIndicator steps={steps} current={currentStep} />
      <StepComponent
        data={formData}
        onSubmit={handleStepSubmit}
        onBack={() => setCurrentStep((prev) => Math.max(0, prev - 1))}
      />
    </div>
  );
}
```

### Controlled Input Component

```tsx
interface TextFieldProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label: string;
  error?: string;
  helperText?: string;
}

const TextField = forwardRef<HTMLInputElement, TextFieldProps>(
  ({ label, error, helperText, id, ...props }, ref) => {
    const inputId = id ?? `field-${label.toLowerCase().replace(/\s+/g, '-')}`;

    return (
      <div className="text-field">
        <label htmlFor={inputId}>{label}</label>
        <input
          ref={ref}
          id={inputId}
          aria-invalid={!!error}
          aria-describedby={error ? `${inputId}-error` : undefined}
          {...props}
        />
        {error && (
          <span id={`${inputId}-error`} className="error" role="alert">
            {error}
          </span>
        )}
        {helperText && !error && (
          <span className="helper-text">{helperText}</span>
        )}
      </div>
    );
  }
);

TextField.displayName = 'TextField';
```

## Testing Patterns

### Component Testing with React Testing Library

```tsx
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

describe('UserCard', () => {
  const mockUser = { id: '1', name: 'John Doe', email: 'john@example.com' };

  it('renders user information', () => {
    render(<UserCard user={mockUser} />);
    expect(screen.getByText('John Doe')).toBeInTheDocument();
    expect(screen.getByText('john@example.com')).toBeInTheDocument();
  });

  it('calls onSelect when clicked', async () => {
    const onSelect = vi.fn();
    render(<UserCard user={mockUser} onSelect={onSelect} />);
    await userEvent.click(screen.getByRole('button'));
    expect(onSelect).toHaveBeenCalledWith(mockUser);
  });

  it('shows loading state', () => {
    render(<UserCard userId="1" />);
    expect(screen.getByTestId('skeleton')).toBeInTheDocument();
  });

  it('fetches and displays user data', async () => {
    render(<UserCard userId="1" />);
    await waitFor(() => {
      expect(screen.getByText('John Doe')).toBeInTheDocument();
    });
  });
});
```

### Hook Testing

```tsx
import { renderHook, act } from '@testing-library/react';

describe('useCounter', () => {
  it('initializes with default value', () => {
    const { result } = renderHook(() => useCounter());
    expect(result.current.count).toBe(0);
  });

  it('increments counter', () => {
    const { result } = renderHook(() => useCounter());
    act(() => { result.current.increment(); });
    expect(result.current.count).toBe(1);
  });

  it('accepts initial value', () => {
    const { result } = renderHook(() => useCounter(10));
    act(() => { result.current.decrement(); });
    expect(result.current.count).toBe(9);
  });
});
```

### Testing with Providers (Wrapper Pattern)

```tsx
function createTestWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <MemoryRouter>{children}</MemoryRouter>
        </AuthProvider>
      </QueryClientProvider>
    );
  };
}

it('displays users from API', async () => {
  server.use(
    rest.get('/api/users', (_req, res, ctx) =>
      res(ctx.json([{ id: '1', name: 'John' }]))
    )
  );

  render(<UserList />, { wrapper: createTestWrapper() });

  await waitFor(() => {
    expect(screen.getByText('John')).toBeInTheDocument();
  });
});
```

### Testing Async Operations

```tsx
it('submits form and shows success message', async () => {
  const onSubmit = vi.fn().mockResolvedValueOnce({ success: true });
  render(<UserForm onSubmit={onSubmit} />);

  await userEvent.type(screen.getByLabelText('Email'), 'john@example.com');
  await userEvent.type(screen.getByLabelText('Password'), 'password123');
  await userEvent.type(screen.getByLabelText('Name'), 'John Doe');
  await userEvent.click(screen.getByRole('button', { name: /submit/i }));

  await waitFor(() => {
    expect(onSubmit).toHaveBeenCalledWith({
      email: 'john@example.com',
      password: 'password123',
      name: 'John Doe',
    });
  });
});
```

## Performance Optimization

### Memoization

```tsx
import { memo, useMemo, useCallback } from 'react';

// Memoize component (only re-renders when props change)
const ExpensiveList = memo(function ExpensiveList({ items, onItemClick }: Props) {
  return (
    <ul>
      {items.map((item) => (
        <li key={item.id} onClick={() => onItemClick(item)}>{item.name}</li>
      ))}
    </ul>
  );
});

// Memoize computed values
function Dashboard({ data }: { data: DataPoint[] }) {
  const statistics = useMemo(() => computeExpensiveStatistics(data), [data]);
  return <StatsDisplay stats={statistics} />;
}

// Memoize callbacks to prevent child re-renders
function ParentComponent() {
  const [items, setItems] = useState<Item[]>([]);
  const handleItemClick = useCallback((item: Item) => {
    console.log('Clicked:', item);
  }, []);

  return <ExpensiveList items={items} onItemClick={handleItemClick} />;
}
```

### Code Splitting

```tsx
import { lazy, Suspense } from 'react';

// Route-level code splitting
const Dashboard = lazy(() => import('./pages/Dashboard'));
const Settings = lazy(() => import('./pages/Settings'));

function App() {
  return (
    <Suspense fallback={<LoadingSpinner />}>
      <Routes>
        <Route path="/dashboard" element={<Dashboard />} />
        <Route path="/settings" element={<Settings />} />
      </Routes>
    </Suspense>
  );
}

// Component-level code splitting (for heavy components)
const HeavyChart = lazy(() => import('./components/HeavyChart'));

function AnalyticsPage() {
  const [showChart, setShowChart] = useState(false);

  return (
    <div>
      <button onClick={() => setShowChart(true)}>Show Chart</button>
      {showChart && (
        <Suspense fallback={<ChartSkeleton />}>
          <HeavyChart />
        </Suspense>
      )}
    </div>
  );
}
```

### Virtual Lists

```tsx
import { useVirtualizer } from '@tanstack/react-virtual';

function VirtualList({ items }: { items: Item[] }) {
  const parentRef = useRef<HTMLDivElement>(null);

  const virtualizer = useVirtualizer({
    count: items.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 50,
    overscan: 5,
  });

  return (
    <div ref={parentRef} style={{ height: '400px', overflow: 'auto' }}>
      <div style={{ height: `${virtualizer.getTotalSize()}px`, position: 'relative' }}>
        {virtualizer.getVirtualItems().map((virtualItem) => (
          <div
            key={virtualItem.key}
            style={{
              position: 'absolute',
              top: 0,
              transform: `translateY(${virtualItem.start}px)`,
              height: `${virtualItem.size}px`,
              width: '100%',
            }}
          >
            <ItemRow item={items[virtualItem.index]} />
          </div>
        ))}
      </div>
    </div>
  );
}
```

### Performance Anti-Patterns

Avoid these common mistakes:

```tsx
// BAD: Creating new object/array on every render
function Bad({ items }) {
  return <Child style={{ color: 'red' }} data={items.filter(Boolean)} />;
}

// GOOD: Memoize or lift out constants
const style = { color: 'red' };
function Good({ items }) {
  const filteredItems = useMemo(() => items.filter(Boolean), [items]);
  return <Child style={style} data={filteredItems} />;
}

// BAD: Anonymous function in JSX causing re-renders
function Bad() {
  return <Button onClick={() => doSomething(id)} />;
}

// GOOD: Use useCallback for stable reference
function Good() {
  const handleClick = useCallback(() => doSomething(id), [id]);
  return <Button onClick={handleClick} />;
}

// BAD: State stored that can be derived
function Bad({ items }) {
  const [count, setCount] = useState(items.length);
  useEffect(() => setCount(items.length), [items]);
  return <span>{count}</span>;
}

// GOOD: Derive during render
function Good({ items }) {
  const count = items.length;
  return <span>{count}</span>;
}
```

## Error Handling

### Error Boundary

```tsx
import { Component, ErrorInfo, ReactNode } from 'react';

interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { hasError: false, error: null };

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    this.props.onError?.(error, errorInfo);
  }

  render() {
    if (this.state.hasError) {
      return this.props.fallback ?? (
        <div role="alert">
          <h2>Something went wrong</h2>
          <p>{this.state.error?.message}</p>
          <button onClick={() => this.setState({ hasError: false, error: null })}>
            Try again
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}

// Usage: wrap route-level or feature-level components
<ErrorBoundary fallback={<ErrorFallback />} onError={logToService}>
  <Dashboard />
</ErrorBoundary>
```

### Async Error Handling

```tsx
function useAsyncAction<T>(action: () => Promise<T>) {
  const [state, setState] = useState<{
    loading: boolean;
    error: Error | null;
    data: T | null;
  }>({ loading: false, error: null, data: null });

  const execute = useCallback(async () => {
    setState({ loading: true, error: null, data: null });
    try {
      const result = await action();
      setState({ loading: false, error: null, data: result });
      return result;
    } catch (e) {
      const error = e instanceof Error ? e : new Error('Unknown error');
      setState({ loading: false, error, data: null });
      throw error;
    }
  }, [action]);

  return { ...state, execute };
}
```

## Security Patterns

### XSS Prevention

```tsx
// React auto-escapes JSX content. NEVER use dangerouslySetInnerHTML
// without sanitization.

// BAD: Direct HTML injection
function Bad({ html }: { html: string }) {
  return <div dangerouslySetInnerHTML={{ __html: html }} />;
}

// GOOD: Sanitize if HTML rendering is absolutely necessary
import DOMPurify from 'dompurify';

function SafeHtml({ html }: { html: string }) {
  const sanitized = DOMPurify.sanitize(html);
  return <div dangerouslySetInnerHTML={{ __html: sanitized }} />;
}

// BEST: Use structured data instead of HTML strings
function Best({ content }: { content: ContentBlock[] }) {
  return (
    <div>
      {content.map((block) => {
        switch (block.type) {
          case 'text': return <p key={block.id}>{block.value}</p>;
          case 'heading': return <h2 key={block.id}>{block.value}</h2>;
          default: return null;
        }
      })}
    </div>
  );
}
```

### Sensitive Data Handling

```tsx
// Never store tokens in localStorage (XSS vulnerable)
// Use httpOnly cookies for auth tokens

// BAD
localStorage.setItem('token', authToken);

// GOOD: Let the server set httpOnly cookies
async function login(credentials: LoginCredentials) {
  await fetch('/api/auth/login', {
    method: 'POST',
    credentials: 'include', // sends/receives cookies
    body: JSON.stringify(credentials),
  });
}

// Always validate and sanitize URL parameters
function UserProfile() {
  const { userId } = useParams();
  const sanitizedId = userId?.replace(/[^a-zA-Z0-9-]/g, '');

  if (!sanitizedId || sanitizedId !== userId) {
    return <Navigate to="/404" />;
  }

  return <ProfileContent userId={sanitizedId} />;
}
```

### Environment Variables

```tsx
// Only VITE_* prefixed vars are exposed to client-side code (Vite)
// Only NEXT_PUBLIC_* prefixed vars are exposed (Next.js)
// NEVER put secrets in client-side env vars

const apiUrl = import.meta.env.VITE_API_URL;

// Validate at startup
if (!apiUrl) {
  throw new Error('VITE_API_URL is required');
}
```

## Accessibility Patterns

### Focus Management

```tsx
function Modal({ isOpen, onClose, title, children }: ModalProps) {
  const closeButtonRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (isOpen) {
      closeButtonRef.current?.focus();
    }
  }, [isOpen]);

  if (!isOpen) return null;

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
      onKeyDown={(e) => e.key === 'Escape' && onClose()}
    >
      <h2 id="modal-title">{title}</h2>
      <div>{children}</div>
      <button ref={closeButtonRef} onClick={onClose}>
        Close
      </button>
    </div>
  );
}
```

### Live Regions (Screen Reader Announcements)

```tsx
function SearchResults({ results, isLoading }: SearchResultsProps) {
  return (
    <>
      <div aria-live="polite" aria-atomic="true" className="sr-only">
        {isLoading
          ? 'Searching...'
          : `${results.length} results found`}
      </div>
      <ul>
        {results.map((result) => (
          <li key={result.id}>{result.title}</li>
        ))}
      </ul>
    </>
  );
}
```

### Keyboard Navigation

```tsx
function Dropdown({ options, onSelect }: DropdownProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [focusedIndex, setFocusedIndex] = useState(-1);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setFocusedIndex((prev) => Math.min(prev + 1, options.length - 1));
        break;
      case 'ArrowUp':
        e.preventDefault();
        setFocusedIndex((prev) => Math.max(prev - 1, 0));
        break;
      case 'Enter':
        if (focusedIndex >= 0) {
          onSelect(options[focusedIndex]);
          setIsOpen(false);
        }
        break;
      case 'Escape':
        setIsOpen(false);
        break;
    }
  };

  return (
    <div onKeyDown={handleKeyDown}>
      <button
        aria-expanded={isOpen}
        aria-haspopup="listbox"
        onClick={() => setIsOpen(!isOpen)}
      >
        Select option
      </button>
      {isOpen && (
        <ul role="listbox">
          {options.map((option, index) => (
            <li
              key={option.id}
              role="option"
              aria-selected={focusedIndex === index}
              onClick={() => { onSelect(option); setIsOpen(false); }}
            >
              {option.label}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
```
