# Advanced Next.js Patterns

> Complements SKILL.md with authentication, advanced Server Actions, testing strategies,
> performance optimization, caching, rendering strategies, and deployment patterns.

---

## Authentication (NextAuth.js / Auth.js)

### Setup

```tsx
// app/api/auth/[...nextauth]/route.ts
import NextAuth from 'next-auth';
import { authOptions } from '@/lib/auth';

const handler = NextAuth(authOptions);
export { handler as GET, handler as POST };
```

### Auth Configuration

```tsx
// lib/auth.ts
import { NextAuthOptions } from 'next-auth';
import { PrismaAdapter } from '@auth/prisma-adapter';
import GitHubProvider from 'next-auth/providers/github';
import CredentialsProvider from 'next-auth/providers/credentials';
import { db } from './db';

export const authOptions: NextAuthOptions = {
  adapter: PrismaAdapter(db),
  providers: [
    GitHubProvider({
      clientId: process.env.GITHUB_ID!,
      clientSecret: process.env.GITHUB_SECRET!,
    }),
    CredentialsProvider({
      name: 'credentials',
      credentials: {
        email: { label: 'Email', type: 'email' },
        password: { label: 'Password', type: 'password' },
      },
      async authorize(credentials) {
        const user = await validateUser(credentials);
        return user;
      },
    }),
  ],
  session: { strategy: 'jwt' },
  pages: { signIn: '/login' },
};
```

### Auth Utility Functions

```tsx
// lib/auth-utils.ts
import { getServerSession } from 'next-auth';
import { redirect } from 'next/navigation';
import { authOptions } from './auth';

export async function getSession() {
  return await getServerSession(authOptions);
}

export async function getCurrentUser() {
  const session = await getSession();
  return session?.user;
}

export async function requireAuth() {
  const user = await getCurrentUser();
  if (!user) redirect('/login');
  return user;
}
```

### Protected Server Component

```tsx
// app/dashboard/page.tsx
import { requireAuth } from '@/lib/auth-utils';

export default async function DashboardPage() {
  const user = await requireAuth();
  return <div>Welcome, {user.name}</div>;
}
```

### Middleware-Based Auth

```tsx
// middleware.ts
import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';
import { getToken } from 'next-auth/jwt';

export async function middleware(request: NextRequest) {
  const token = await getToken({ req: request });
  const isAuthPage = request.nextUrl.pathname.startsWith('/login');
  const isProtected = request.nextUrl.pathname.startsWith('/dashboard');

  if (isAuthPage && token) {
    return NextResponse.redirect(new URL('/dashboard', request.url));
  }

  if (isProtected && !token) {
    return NextResponse.redirect(new URL('/login', request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: ['/dashboard/:path*', '/login'],
};
```

---

## Advanced Server Actions

### useFormState for Validation Feedback

```tsx
// app/actions.ts
'use server';

import { z } from 'zod';
import { revalidatePath } from 'next/cache';

type ActionState = {
  errors?: { title?: string[]; content?: string[] };
  message?: string;
};

const createPostSchema = z.object({
  title: z.string().min(1),
  content: z.string().min(10),
});

export async function createPost(
  prevState: ActionState,
  formData: FormData
): Promise<ActionState> {
  const validated = createPostSchema.safeParse({
    title: formData.get('title'),
    content: formData.get('content'),
  });

  if (!validated.success) {
    return { errors: validated.error.flatten().fieldErrors };
  }

  try {
    await db.post.create({ data: validated.data });
    revalidatePath('/posts');
    return { message: 'Post created successfully' };
  } catch {
    return { message: 'Failed to create post' };
  }
}
```

### Client Form with useFormState and useFormStatus

```tsx
// components/PostForm.tsx
'use client';

import { useFormState, useFormStatus } from 'react-dom';
import { createPost } from '@/app/actions';

function SubmitButton() {
  const { pending } = useFormStatus();
  return (
    <button type="submit" disabled={pending}>
      {pending ? 'Creating...' : 'Create Post'}
    </button>
  );
}

export function PostForm() {
  const [state, formAction] = useFormState(createPost, {});

  return (
    <form action={formAction}>
      <div>
        <input name="title" placeholder="Title" />
        {state.errors?.title && <p className="error">{state.errors.title}</p>}
      </div>
      <div>
        <textarea name="content" placeholder="Content" />
        {state.errors?.content && <p className="error">{state.errors.content}</p>}
      </div>
      <SubmitButton />
      {state.message && <p>{state.message}</p>}
    </form>
  );
}
```

### Optimistic Updates

```tsx
'use client';

import { useOptimistic } from 'react';
import { addTodo } from '@/app/actions';

export function TodoList({ todos }: { todos: Todo[] }) {
  const [optimisticTodos, addOptimistic] = useOptimistic(
    todos,
    (state: Todo[], newTodo: string) => [
      ...state,
      { id: `temp-${Date.now()}`, text: newTodo, completed: false },
    ]
  );

  async function handleSubmit(formData: FormData) {
    const text = formData.get('text') as string;
    addOptimistic(text);
    await addTodo(formData);
  }

  return (
    <>
      <ul>
        {optimisticTodos.map((todo) => (
          <li key={todo.id}>{todo.text}</li>
        ))}
      </ul>
      <form action={handleSubmit}>
        <input name="text" required />
        <button type="submit">Add</button>
      </form>
    </>
  );
}
```

---

## Testing

### Component Testing with React Testing Library

```tsx
// __tests__/components/Counter.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { Counter } from '@/components/Counter';

describe('Counter', () => {
  it('should increment count on click', () => {
    render(<Counter />);
    const button = screen.getByRole('button');

    expect(button).toHaveTextContent('Count: 0');
    fireEvent.click(button);
    expect(button).toHaveTextContent('Count: 1');
  });
});
```

### Testing Server Components

Server Components cannot be rendered directly with React Testing Library. Test the data-fetching logic separately and the rendering with a shallow approach:

```tsx
// __tests__/app/users/page.test.ts
import { db } from '@/lib/db';
import UsersPage from '@/app/users/page';

vi.mock('@/lib/db', () => ({
  db: { user: { findMany: vi.fn() } },
}));

describe('UsersPage', () => {
  it('should render user list', async () => {
    vi.mocked(db.user.findMany).mockResolvedValue([
      { id: '1', name: 'Alice' },
      { id: '2', name: 'Bob' },
    ]);

    const result = await UsersPage();
    // Assert on the returned JSX structure
    expect(result).toBeTruthy();
  });
});
```

### Testing API Route Handlers

```tsx
// __tests__/api/users.test.ts
import { GET, POST } from '@/app/api/users/route';
import { NextRequest } from 'next/server';

describe('GET /api/users', () => {
  it('should return paginated users', async () => {
    const request = new NextRequest('http://localhost/api/users?page=1');
    const response = await GET(request);
    const data = await response.json();

    expect(response.status).toBe(200);
    expect(Array.isArray(data)).toBe(true);
  });
});

describe('POST /api/users', () => {
  it('should return 400 for invalid input', async () => {
    const request = new NextRequest('http://localhost/api/users', {
      method: 'POST',
      body: JSON.stringify({ name: '' }),
    });
    const response = await POST(request);

    expect(response.status).toBe(400);
  });
});
```

### Testing Server Actions

```tsx
// __tests__/actions.test.ts
import { createPost } from '@/app/actions';

describe('createPost', () => {
  it('should return errors for invalid input', async () => {
    const formData = new FormData();
    formData.set('title', '');
    formData.set('content', 'short');

    const result = await createPost({}, formData);
    expect(result.errors?.title).toBeDefined();
  });
});
```

### E2E Testing with Playwright

```tsx
// e2e/blog.spec.ts
import { test, expect } from '@playwright/test';

test('should create a new blog post', async ({ page }) => {
  await page.goto('/posts/new');
  await page.fill('input[name="title"]', 'Test Post');
  await page.fill('textarea[name="content"]', 'This is test content for the post.');
  await page.click('button[type="submit"]');

  await expect(page).toHaveURL(/\/posts/);
  await expect(page.getByText('Test Post')).toBeVisible();
});
```

---

## Rendering Strategies

### Static Site Generation (SSG)

Pages are rendered at build time. Default for pages without dynamic data.

```tsx
// app/blog/[slug]/page.tsx
export async function generateStaticParams() {
  const posts = await getPosts();
  return posts.map((post) => ({ slug: post.slug }));
}

export default async function BlogPost({ params }: { params: { slug: string } }) {
  const post = await getPost(params.slug);
  return <article><h1>{post.title}</h1><div>{post.content}</div></article>;
}
```

### Incremental Static Regeneration (ISR)

Revalidate static pages on a time-based or on-demand schedule.

```tsx
// Time-based revalidation
async function getData() {
  const res = await fetch('https://api.example.com/data', {
    next: { revalidate: 60 }, // Regenerate every 60 seconds
  });
  return res.json();
}

// On-demand revalidation (in a Server Action or Route Handler)
import { revalidatePath, revalidateTag } from 'next/cache';

export async function updatePost(id: string, data: PostData) {
  await db.post.update({ where: { id }, data });
  revalidatePath('/posts');         // Revalidate by path
  revalidateTag('posts');           // Revalidate by cache tag
}
```

### Server-Side Rendering (SSR)

Force dynamic rendering with `no-store` or dynamic functions.

```tsx
// Option 1: no-store fetch
async function getData() {
  const res = await fetch('https://api.example.com/data', {
    cache: 'no-store',
  });
  return res.json();
}

// Option 2: Route segment config
export const dynamic = 'force-dynamic';
export const revalidate = 0;
```

### Route Segment Config Options

```tsx
// Apply to any page.tsx or layout.tsx
export const dynamic = 'auto' | 'force-dynamic' | 'error' | 'force-static';
export const revalidate = false | 0 | number;
export const runtime = 'nodejs' | 'edge';
export const preferredRegion = 'auto' | 'global' | 'home' | string[];
export const maxDuration = number;
```

---

## Caching

### Four Caching Layers

1. **Request Memoization**: `fetch` calls with the same URL/options are deduped within a single render pass
2. **Data Cache**: `fetch` results are cached on the server across requests (persists across deploys on Vercel)
3. **Full Route Cache**: Rendered HTML and RSC payload cached at build time for static routes
4. **Router Cache**: Client-side RSC payload cache for visited routes (reduces server requests)

### Cache Control Patterns

```tsx
// Static: cached indefinitely (default)
fetch('https://api.example.com/data');

// Revalidate: cached with time-based expiry
fetch('https://api.example.com/data', { next: { revalidate: 3600 } });

// Dynamic: never cached
fetch('https://api.example.com/data', { cache: 'no-store' });

// Tag-based: invalidate on demand
fetch('https://api.example.com/posts', { next: { tags: ['posts'] } });
// Later: revalidateTag('posts');
```

### Opting Out of Caching

- Using `cache: 'no-store'` on fetch
- Using dynamic functions: `cookies()`, `headers()`, `searchParams`
- Setting `export const dynamic = 'force-dynamic'`
- Using `POST` method in route handlers (not cached by default)

---

## Performance Optimization

### Image Optimization

```tsx
import Image from 'next/image';

// Local image (auto-sized at build time)
import profilePic from './me.png';
<Image src={profilePic} alt="Profile" placeholder="blur" />

// Remote image (must specify dimensions)
<Image
  src="https://example.com/photo.jpg"
  alt="Photo"
  width={800}
  height={600}
  priority            // Preload for above-the-fold images
  sizes="(max-width: 768px) 100vw, 50vw"
/>
```

### Font Optimization

```tsx
import { Inter, Roboto_Mono } from 'next/font/google';

const inter = Inter({ subsets: ['latin'], display: 'swap' });
const robotoMono = Roboto_Mono({ subsets: ['latin'], variable: '--font-mono' });

export default function Layout({ children }: { children: React.ReactNode }) {
  return (
    <html className={`${inter.className} ${robotoMono.variable}`}>
      <body>{children}</body>
    </html>
  );
}
```

### Script Optimization

```tsx
import Script from 'next/script';

// Load after page becomes interactive
<Script src="https://analytics.example.com/script.js" strategy="afterInteractive" />

// Load during idle time
<Script src="https://third-party.example.com/widget.js" strategy="lazyOnload" />

// Inline scripts with worker strategy (offloads to web worker)
<Script strategy="worker" src="https://heavy-script.example.com/lib.js" />
```

### Bundle Analysis

```bash
# Install analyzer
npm install @next/bundle-analyzer

# Add to next.config.js
const withBundleAnalyzer = require('@next/bundle-analyzer')({
  enabled: process.env.ANALYZE === 'true',
});
module.exports = withBundleAnalyzer(nextConfig);

# Run analysis
ANALYZE=true npm run build
```

### Dynamic Imports (Code Splitting)

```tsx
import dynamic from 'next/dynamic';

// Lazy-load heavy component (renders on client only)
const HeavyChart = dynamic(() => import('@/components/HeavyChart'), {
  loading: () => <ChartSkeleton />,
  ssr: false, // Skip server rendering if component uses browser APIs
});

// Named export
const Modal = dynamic(() =>
  import('@/components/Modals').then((mod) => mod.ConfirmModal)
);
```

---

## Metadata and SEO

### Static Metadata

```tsx
// app/about/page.tsx
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'About Us',
  description: 'Learn more about our company',
  openGraph: {
    title: 'About Us',
    description: 'Learn more about our company',
    images: [{ url: '/og-about.png', width: 1200, height: 630 }],
  },
  twitter: {
    card: 'summary_large_image',
  },
};
```

### Dynamic Metadata

```tsx
// app/blog/[slug]/page.tsx
export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const post = await getPost(params.slug);

  return {
    title: post.title,
    description: post.excerpt,
    openGraph: {
      title: post.title,
      images: [post.coverImage],
    },
  };
}
```

### Sitemap and Robots

```tsx
// app/sitemap.ts
import type { MetadataRoute } from 'next';

export default function sitemap(): MetadataRoute.Sitemap {
  return [
    { url: 'https://example.com', lastModified: new Date(), changeFrequency: 'yearly', priority: 1 },
    { url: 'https://example.com/about', lastModified: new Date(), changeFrequency: 'monthly', priority: 0.8 },
  ];
}

// app/robots.ts
export default function robots(): MetadataRoute.Robots {
  return {
    rules: { userAgent: '*', allow: '/', disallow: '/private/' },
    sitemap: 'https://example.com/sitemap.xml',
  };
}
```

---

## Deployment

### Vercel (Recommended)

```bash
npm i -g vercel
vercel              # Deploy preview
vercel --prod       # Deploy production
```

### Docker

```dockerfile
FROM node:20-alpine AS base

FROM base AS deps
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production

FROM base AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM base AS runner
WORKDIR /app
ENV NODE_ENV=production
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static

EXPOSE 3000
ENV PORT=3000
CMD ["node", "server.js"]
```

Enable standalone output in `next.config.js`:

```js
const nextConfig = {
  output: 'standalone',
};
```

### Environment Variables

```bash
# .env.local (development, not committed)
DATABASE_URL=postgresql://user:pass@localhost:5432/mydb
NEXTAUTH_SECRET=dev-secret-change-in-prod
NEXTAUTH_URL=http://localhost:3000

# .env.production (or set in deployment platform)
DATABASE_URL=postgresql://user:pass@prod-host:5432/mydb
NEXTAUTH_SECRET=<generated-secret>
NEXTAUTH_URL=https://myapp.com
```

Client-side env vars must be prefixed with `NEXT_PUBLIC_`:

```bash
NEXT_PUBLIC_API_URL=https://api.example.com   # Available in client and server
API_SECRET=secret-value                         # Server-only
```

---

## Common Patterns

### Global Error Boundary

```tsx
// app/global-error.tsx
'use client';

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <html>
      <body>
        <h2>Something went wrong!</h2>
        <button onClick={() => reset()}>Try again</button>
      </body>
    </html>
  );
}
```

### Parallel Routes

```
app/
├── @analytics/
│   └── page.tsx
├── @team/
│   └── page.tsx
└── layout.tsx
```

```tsx
// app/layout.tsx
export default function Layout({
  children,
  analytics,
  team,
}: {
  children: React.ReactNode;
  analytics: React.ReactNode;
  team: React.ReactNode;
}) {
  return (
    <div>
      {children}
      <div className="grid grid-cols-2">
        {analytics}
        {team}
      </div>
    </div>
  );
}
```

### Intercepting Routes

Used for modal patterns (e.g., photo galleries, login modals).

```
app/
├── feed/
│   └── page.tsx
├── photo/[id]/
│   └── page.tsx              # Full page view
└── @modal/
    └── (.)photo/[id]/
        └── page.tsx           # Intercepted modal view
```

Convention: `(.)` same level, `(..)` one level up, `(..)(..)` two levels up, `(...)` from root.

### Route Groups for Multiple Root Layouts

```
app/
├── (marketing)/
│   ├── layout.tsx             # Marketing layout
│   ├── page.tsx               # /
│   └── about/page.tsx         # /about
└── (app)/
    ├── layout.tsx             # App layout (with sidebar)
    └── dashboard/page.tsx     # /dashboard
```
