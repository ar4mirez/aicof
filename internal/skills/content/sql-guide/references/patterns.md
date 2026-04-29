# SQL Patterns Reference

## Contents

- [Window Functions](#window-functions)
- [CTE Patterns](#cte-patterns)
- [Indexing Strategies](#indexing-strategies)
- [Advanced Query Techniques](#advanced-query-techniques)

## Window Functions

### ROW_NUMBER -- Deduplicate or Pick Latest per Group

```sql
-- Get the most recent order per user
WITH ranked AS (
    SELECT
        o.*,
        ROW_NUMBER() OVER (
            PARTITION BY o.user_id
            ORDER BY o.created_at DESC
        ) AS rn
    FROM orders o
    WHERE o.status != 'cancelled'
)
SELECT id, user_id, total_cents, created_at
FROM ranked
WHERE rn = 1;
```

### RANK and DENSE_RANK -- Leaderboards with Ties

```sql
-- Rank products by sales; ties get the same rank
SELECT
    p.id,
    p.name,
    SUM(oi.quantity) AS units_sold,
    RANK() OVER (ORDER BY SUM(oi.quantity) DESC) AS rank,
    DENSE_RANK() OVER (ORDER BY SUM(oi.quantity) DESC) AS dense_rank
FROM products p
JOIN order_items oi ON oi.product_id = p.id
JOIN orders o ON o.id = oi.order_id
WHERE o.status = 'delivered'
  AND o.created_at >= now() - INTERVAL '90 days'
GROUP BY p.id, p.name
ORDER BY units_sold DESC;
```

### LAG and LEAD -- Compare Adjacent Rows

```sql
-- Month-over-month revenue change
SELECT
    date_trunc('month', created_at) AS month,
    SUM(total_cents) AS revenue,
    LAG(SUM(total_cents)) OVER (ORDER BY date_trunc('month', created_at)) AS prev_month,
    ROUND(
        (SUM(total_cents) - LAG(SUM(total_cents)) OVER (ORDER BY date_trunc('month', created_at)))
        * 100.0
        / NULLIF(LAG(SUM(total_cents)) OVER (ORDER BY date_trunc('month', created_at)), 0),
        2
    ) AS pct_change
FROM orders
WHERE status = 'delivered'
GROUP BY date_trunc('month', created_at)
ORDER BY month;
```

### Running Totals and Moving Averages

```sql
-- Running total of daily signups
SELECT
    date_trunc('day', created_at) AS day,
    COUNT(*) AS daily_signups,
    SUM(COUNT(*)) OVER (ORDER BY date_trunc('day', created_at)) AS cumulative_signups
FROM users
GROUP BY date_trunc('day', created_at)
ORDER BY day;

-- 7-day moving average of order value
SELECT
    date_trunc('day', created_at) AS day,
    AVG(total_cents) AS daily_avg,
    AVG(AVG(total_cents)) OVER (
        ORDER BY date_trunc('day', created_at)
        ROWS BETWEEN 6 PRECEDING AND CURRENT ROW
    ) AS moving_avg_7d
FROM orders
WHERE status = 'delivered'
GROUP BY date_trunc('day', created_at)
ORDER BY day;
```

### NTILE -- Percentile Buckets

```sql
-- Segment users into spending quartiles
SELECT
    u.id,
    u.email,
    SUM(o.total_cents) AS lifetime_value,
    NTILE(4) OVER (ORDER BY SUM(o.total_cents) DESC) AS quartile
FROM users u
JOIN orders o ON o.user_id = u.id
WHERE o.status = 'delivered'
GROUP BY u.id, u.email;
```

## CTE Patterns

### Recursive CTE -- Hierarchical Data

```sql
-- Org chart: find all reports under a manager
WITH RECURSIVE org_tree AS (
    -- Base case: the root manager
    SELECT id, name, manager_id, 0 AS depth
    FROM employees
    WHERE id = $1

    UNION ALL

    -- Recursive step: direct reports of current level
    SELECT e.id, e.name, e.manager_id, ot.depth + 1
    FROM employees e
    JOIN org_tree ot ON e.manager_id = ot.id
    WHERE ot.depth < 10  -- safety limit to prevent infinite recursion
)
SELECT id, name, depth
FROM org_tree
ORDER BY depth, name;
```

### CTE for Staged Transformations

```sql
-- Multi-step analytics: clean, aggregate, rank
WITH daily_metrics AS (
    SELECT
        date_trunc('day', created_at) AS day,
        user_id,
        COUNT(*) AS actions,
        SUM(CASE WHEN action_type = 'purchase' THEN 1 ELSE 0 END) AS purchases
    FROM user_actions
    WHERE created_at >= now() - INTERVAL '30 days'
    GROUP BY date_trunc('day', created_at), user_id
),
user_summary AS (
    SELECT
        user_id,
        SUM(actions) AS total_actions,
        SUM(purchases) AS total_purchases,
        COUNT(DISTINCT day) AS active_days
    FROM daily_metrics
    GROUP BY user_id
)
SELECT
    us.*,
    RANK() OVER (ORDER BY us.total_purchases DESC) AS purchase_rank
FROM user_summary us
WHERE us.active_days >= 5
ORDER BY us.total_purchases DESC
LIMIT 100;
```

### CTE with INSERT (Writeable CTEs -- PostgreSQL)

```sql
-- Archive and delete old records in one statement
WITH archived AS (
    DELETE FROM orders
    WHERE status = 'cancelled'
      AND created_at < now() - INTERVAL '2 years'
    RETURNING *
)
INSERT INTO orders_archive
SELECT * FROM archived;
```

## Indexing Strategies

### Composite Index Column Order

The column order in a composite index matters. Place columns in this priority:

1. **Equality** columns first (`WHERE status = 'active'`)
2. **Range** columns next (`WHERE created_at > '2025-01-01'`)
3. **Sort** columns last (`ORDER BY created_at DESC`)

```sql
-- Query: WHERE status = $1 AND created_at > $2 ORDER BY created_at DESC
-- Optimal index:
CREATE INDEX idx_orders_status_created
ON orders (status, created_at DESC);

-- This index supports:
--   WHERE status = 'pending'                          (uses first column)
--   WHERE status = 'pending' AND created_at > '...'   (uses both columns)
--   WHERE status = 'pending' ORDER BY created_at DESC (equality + sort)
-- It does NOT efficiently support:
--   WHERE created_at > '...'  (skips first column; needs separate index)
```

### Covering Indexes (INCLUDE)

Avoid table lookups by including extra columns in the index.

```sql
-- Query frequently fetches email along with active user lookups
CREATE INDEX idx_users_active_email
ON users (is_active) INCLUDE (email, created_at)
WHERE is_active = true;

-- The query below is satisfied entirely from the index (index-only scan)
SELECT email, created_at
FROM users
WHERE is_active = true;
```

### GIN Indexes for JSONB and Arrays

```sql
-- Index JSONB column for containment queries
CREATE INDEX idx_products_metadata ON products USING GIN (metadata);

-- Supports: WHERE metadata @> '{"color": "red"}'
-- Supports: WHERE metadata ? 'color'

-- Index array column
CREATE INDEX idx_articles_tags ON articles USING GIN (tags);

-- Supports: WHERE tags @> ARRAY['sql', 'postgres']
-- Supports: WHERE 'sql' = ANY(tags)
```

### Expression Indexes

```sql
-- Index on lower(email) for case-insensitive lookups
CREATE UNIQUE INDEX idx_users_email_lower
ON users (lower(email));

-- Query uses the index:
SELECT * FROM users WHERE lower(email) = lower($1);

-- Index on extracted JSONB field
CREATE INDEX idx_events_type
ON events ((payload->>'type'));

-- Query uses the index:
SELECT * FROM events WHERE payload->>'type' = 'signup';
```

## Advanced Query Techniques

### Lateral Joins -- Top-N Per Group

```sql
-- Get the 3 most recent orders per user (more efficient than window function for small N)
SELECT u.id, u.email, recent.id AS order_id, recent.total_cents, recent.created_at
FROM users u
CROSS JOIN LATERAL (
    SELECT o.id, o.total_cents, o.created_at
    FROM orders o
    WHERE o.user_id = u.id
      AND o.status != 'cancelled'
    ORDER BY o.created_at DESC
    LIMIT 3
) recent
WHERE u.is_active = true;
```

### FILTER Clause -- Conditional Aggregation

```sql
-- Single pass over orders table for multiple metrics
SELECT
    date_trunc('month', created_at) AS month,
    COUNT(*) AS total_orders,
    COUNT(*) FILTER (WHERE status = 'delivered') AS delivered,
    COUNT(*) FILTER (WHERE status = 'cancelled') AS cancelled,
    SUM(total_cents) FILTER (WHERE status = 'delivered') AS delivered_revenue,
    AVG(total_cents) FILTER (WHERE status = 'delivered') AS avg_order_value
FROM orders
WHERE created_at >= now() - INTERVAL '12 months'
GROUP BY date_trunc('month', created_at)
ORDER BY month;
```

### Generate Series -- Fill Date Gaps

```sql
-- Ensure every day appears even when no orders exist
SELECT
    d.day,
    COALESCE(SUM(o.total_cents), 0) AS revenue,
    COALESCE(COUNT(o.id), 0) AS order_count
FROM generate_series(
    date_trunc('day', now() - INTERVAL '30 days'),
    date_trunc('day', now()),
    '1 day'::interval
) AS d(day)
LEFT JOIN orders o
    ON date_trunc('day', o.created_at) = d.day
   AND o.status = 'delivered'
GROUP BY d.day
ORDER BY d.day;
```
