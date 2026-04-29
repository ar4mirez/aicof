# RFD Creation Process — Detailed Guide

This document provides detailed examples, full state machine documentation, and verbose instructions for the create-rfd skill.

## Table of Contents

1. [Detailed State Machine](#detailed-state-machine)
2. [Discovery Questions Deep Dive](#discovery-questions-deep-dive)
3. [Complete Examples](#complete-examples)
4. [State Transition Details](#state-transition-details)
5. [Template Deep Dive](#template-deep-dive)

---

## Detailed State Machine

### State Diagram (Extended)

```
┌──────────────┐
│Prediscussion │  Very rough idea, author still thinking
└──────┬───────┘  Location: .claude/rfd/NNNN-rfd-topic.md
       │
       ├─→ Abandoned (anytime)
       │
       ↓
┌──────────────┐
│  Ideation    │  Ready for informal feedback, not finalized
└──────┬───────┘  Location: .claude/rfd/NNNN-rfd-topic.md
       │
       ├─→ Abandoned (anytime)
       │
       ↓
┌──────────────┐
│  Discussion  │  Open for team discussion, seeking consensus
└──────┬───────┘  Location: docs/rfd/NNNN.md
       │           (Promoted to public documentation)
       ├─→ Abandoned (anytime)
       │
       ↓
┌──────────────┐
│  Published   │  Decision made, documented for reference
└──────┬───────┘  Location: docs/rfd/NNNN.md
       │
       ├─→ Abandoned (rare, but possible if reversed)
       │
       ↓
┌──────────────┐
│  Committed   │  Implemented, part of the codebase
└──────────────┘  Location: docs/rfd/NNNN.md
                  (Final state, includes implementation links)
```

### State Transition Conditions

#### Prediscussion → Ideation

**Trigger**: Author has clearer picture of the problem and options.

**Requirements:**
- Problem statement is well-defined
- At least 2 options have been identified
- Ready for informal feedback

**Actions:**
- Update state in frontmatter
- No file move needed (both in `.claude/rfd/`)

#### Ideation → Discussion

**Trigger**: Ready for team input and formal discussion.

**Requirements:**
- Options are well-researched with pros/cons
- Problem statement is clear to external readers
- Author is ready to answer questions

**Actions:**
1. Update state to "Discussion" in frontmatter
2. Move file from `.claude/rfd/NNNN-rfd-topic.md` to `docs/rfd/NNNN.md`
3. Update `rfd-index.yaml` with new path
4. (Optional) Create GitHub discussion thread
5. (Optional) Update `mkdocs.yml` navigation if manually maintained

**Example:**
```bash
# Move file
mv .claude/rfd/0042-rfd-api-caching.md docs/rfd/0042.md

# Update frontmatter
sed -i 's/state: Ideation/state: Discussion/' docs/rfd/0042.md

# Update rfd-index.yaml
# (AI should do this programmatically)
```

#### Discussion → Published

**Trigger**: Decision has been made, consensus reached.

**Requirements:**
- Proposal section identifies recommended option
- Rationale for decision is documented
- Open questions are answered or removed
- Stakeholders have provided input

**Actions:**
1. Update state to "Published" in frontmatter
2. Ensure "Proposal" section has final decision and rationale
3. Remove unanswered "Open Questions" (or move to answered)
4. Update `rfd-index.yaml`
5. Mark as source of truth for this decision

**No File Move**: Stays in `docs/rfd/NNNN.md`

#### Published → Committed

**Trigger**: Implementation is complete.

**Requirements:**
- All code changes merged
- Related PRD (if exists) is complete
- Implementation matches decision in RFD

**Actions:**
1. Update state to "Committed" in frontmatter
2. Add links to implementation PRs/commits in References section
3. Add link to related PRD (if exists)
4. Update `rfd-index.yaml`

**No File Move**: Stays in `docs/rfd/NNNN.md`

#### Any State → Abandoned

**Trigger**: Proposal rejected, superseded, or no longer relevant.

**Requirements:**
- Clear reason for abandonment
- Note added explaining why

**Actions:**
1. Update state to "Abandoned" in frontmatter
2. Add abandonment note at top of document:
   ```markdown
   > **Note**: This RFD was abandoned on YYYY-MM-DD because [reason].
   > [Optional: Link to superseding RFD or explanation]
   ```
3. Keep file for historical reference (DO NOT DELETE)
4. Update `rfd-index.yaml`

**No File Move**: Stays in current location

---

## Discovery Questions Deep Dive

### Question Categories

#### 1. Problem/Goal Questions

**Purpose**: Understand what the user is trying to achieve.

**Questions:**
- "What problem are you trying to solve?"
- "What outcome do you want to achieve?"
- "What's the current pain point?"
- "Why is this important right now?"
- "What happens if we do nothing?"

**Example Follow-ups:**
```
User: "We need to decide on a caching strategy"

AI: "What problem are you trying to solve with caching?"

Options:
a) Slow response times
b) High database load
c) Expensive API calls to external services
d) All of the above
e) Other (please specify)
```

#### 2. Constraint Questions

**Purpose**: Identify technical, organizational, or resource constraints.

**Questions:**
- "What constraints exist?" (technical, team, timeline, budget)
- "Any hard requirements that eliminate options?"
- "What infrastructure do you have available?"
- "Any compliance or regulatory requirements?"
- "What's the team's familiarity with different technologies?"

**Example:**
```
AI: "What infrastructure constraints exist?"

Options:
a) Single server deployment
b) Multiple servers, no shared state
c) Cloud with managed services available (AWS, GCP, Azure)
d) Kubernetes cluster
e) Hybrid cloud/on-premise
```

#### 3. Stakeholder Questions

**Purpose**: Understand who needs to be involved in the decision.

**Questions:**
- "Who needs to weigh in on this decision?"
- "Is this a solo decision or team consensus needed?"
- "Are there other teams affected by this?"
- "Who will maintain this in the future?"

**Example:**
```
AI: "Who needs to be involved in this decision?"

Options:
a) Just me (individual decision)
b) Development team
c) DevOps/infrastructure team
d) Product/design team
e) Multiple teams and stakeholders
```

#### 4. Timeline Questions

**Purpose**: Understand urgency and time available for research.

**Questions:**
- "When does this decision need to be made?"
- "Is this urgent or can it wait for more research?"
- "When would implementation begin?"
- "Are there any deadlines driving this?"

**Example:**
```
AI: "What's the timeline for this decision?"

Options:
a) Urgent (needed this week)
b) Soon (within 2-4 weeks)
c) Medium-term (1-2 months)
d) Long-term planning (3+ months)
```

#### 5. Options Awareness Questions

**Purpose**: Understand what the user has already considered.

**Questions:**
- "Have you already considered some options?"
- "Are there approaches you've ruled out?"
- "What alternatives are you aware of?"
- "Any prior attempts or experiments?"

**Example:**
```
AI: "Have you already considered some options?"

If yes:
- "Which options are you considering?"
- "Which have you already ruled out and why?"

If no:
- "Would you like me to research common approaches?"
```

#### 6. Success Criteria Questions

**Purpose**: Define what a good decision looks like.

**Questions:**
- "How will you know if the decision was good?"
- "What would success look like?"
- "What metrics matter most?" (performance, cost, maintainability)
- "What's the most important factor in this decision?"

**Example:**
```
AI: "What's most important for this decision?"

Rank in order (1-5):
a) Performance
b) Cost efficiency
c) Developer experience
d) Maintainability
e) Time to implement
```

#### 7. Reversibility Questions

**Purpose**: Understand decision risk and flexibility.

**Questions:**
- "How hard would it be to change this decision later?"
- "Is this a one-way door or two-way door?"
- "What's the migration cost if we change our mind?"
- "How much code would be affected?"

**Example:**
```
AI: "If we make this decision, how hard would it be to reverse?"

Options:
a) Easy to change (configuration-level)
b) Moderate (some code changes)
c) Hard (architectural changes needed)
d) Very hard (complete rewrite)
```

### Adapting Questions to Topic

#### Example: Database Choice

```
1. Problem/Goal: "What database operations are bottlenecks?"
2. Constraints: "What's your data model complexity?" (simple key-value vs. complex relations)
3. Timeline: "When do you need to migrate?"
4. Options: "Considering SQL vs NoSQL vs NewSQL?"
5. Success: "Optimizing for reads, writes, or both?"
6. Reversibility: "How coupled is your code to the current database?"
```

#### Example: Authentication Strategy

```
1. Problem/Goal: "What's wrong with current auth?"
2. Constraints: "Need to support third-party login?"
3. Stakeholders: "Will mobile apps use this?"
4. Timeline: "Security audit coming up?"
5. Options: "Aware of OAuth, JWT, sessions?"
6. Success: "Prioritize security or user experience?"
7. Reversibility: "How many auth touchpoints exist?"
```

---

## Complete Examples

### Example 1: Caching Strategy RFD (Full Workflow)

#### Step 1: User Request

```
User: "Create an RFD exploring caching strategies for our API"
```

#### Step 2: AI Checks Context

AI reads:
- `CLAUDE.md` → Tech stack: Node.js, Express, PostgreSQL, AWS
- `rfd-index.yaml` → No related RFDs
- Codebase scan → API endpoints in `src/routes/`, database queries in `src/models/`

#### Step 3: AI Asks Questions

```
AI: I'll create an RFD to explore caching options. Let me ask a few questions:

1. What's the current pain point?
   a) Slow response times
   b) High database load
   c) Expensive external API calls
   d) All of the above

2. What's your read/write ratio for the data?
   a) Mostly reads (>80%)
   b) Balanced (40-60% reads)
   c) Write-heavy (<40% reads)

3. What infrastructure do you have available?
   a) Single server
   b) Multiple servers, no shared state
   c) AWS with managed services (ElastiCache, CloudFront)
   d) Kubernetes cluster

4. How stale can cached data be?
   a) Real-time (seconds)
   b) Near real-time (minutes)
   c) Relaxed (hours)
   d) Depends on data type

5. What's most important?
   a) Performance (fastest possible)
   b) Cost efficiency
   c) Simplicity
   d) Scalability
```

#### Step 4: User Responds

```
User: "1: b) High database load, 2: a) Mostly reads, 3: c) AWS with managed services,
4: d) Depends on data type, 5: a) Performance"
```

#### Step 5: AI Researches Options

AI identifies:
- **Option A**: Application-level in-memory caching (Node.js `memory-cache`)
- **Option B**: Redis distributed cache (AWS ElastiCache)
- **Option C**: CloudFront CDN edge caching
- **Option D**: Multi-tier caching (in-memory + Redis + CDN)

AI researches pros/cons for each.

#### Step 6: AI Generates RFD

AI gets next RFD number from `rfd-index.yaml` → `next_number: 44`

Creates file: `.claude/rfd/0044-rfd-api-caching.md`

```markdown
---
rfd: 44
title: API Caching Strategy
authors:
  - name: AI Assistant
state: Ideation
labels: [performance, infrastructure]
created: 2025-02-11
updated: 2025-02-11
discussion: null
related_prd: null
---

# RFD 44: API Caching Strategy

## Summary

We need to implement caching for our API to reduce database load caused by high read
traffic. This RFD evaluates four caching approaches with different tradeoffs for
performance, cost, and complexity.

## Problem Statement

Our PostgreSQL database is experiencing high load during peak hours (>80% CPU), primarily
from read queries. Response times have increased from 50ms to 200-300ms. Most of our
traffic (>80%) is read-heavy, with data that can tolerate some staleness depending on
the endpoint.

Current state:
- No caching implemented
- All requests hit the database
- Peak load: 500 req/sec, 80% are reads
- Database CPU: 80-90% during peak hours

If we do nothing: Database will become a bottleneck, requiring expensive vertical scaling
or read replicas.

## Background

**Tech Stack**: Node.js (Express), PostgreSQL, AWS (EC2, RDS)

**Infrastructure**: Multiple EC2 instances behind ALB, RDS PostgreSQL (db.t3.large)

**API Endpoints**:
- `/api/products` (read-heavy, can be stale for minutes)
- `/api/user/:id` (read-heavy, should be fresh)
- `/api/cart` (read/write balanced, must be real-time)
- `/api/orders` (write-heavy, no caching needed)

No previous caching attempts. Team has experience with Redis (used in previous projects).

## Options Considered

### Option A: Application-Level In-Memory Caching

Use Node.js `node-cache` or `memory-cache` to cache responses in application memory.

**Pros:**
- Zero infrastructure cost
- Simple to implement (15-30 min)
- Extremely fast (microseconds)
- No network latency

**Cons:**
- Not shared across instances (each instance has separate cache)
- Lost on deployment/restart
- Limited by instance memory
- Cache invalidation is per-instance (inconsistency risk)

**Effort**: Low (1-2 days)

**Best for**: `/api/products` (read-only, tolerate staleness)

### Option B: Redis Distributed Cache

Use AWS ElastiCache (Redis) as a shared cache layer.

**Pros:**
- Shared across all instances (consistent)
- Persists across deployments
- Rich data structures (sets, sorted sets, etc.)
- Battle-tested, widely used
- Fine-grained TTL control

**Cons:**
- Infrastructure cost (~$50-100/month for t3.small)
- Network latency (1-5ms)
- Requires cache invalidation strategy
- Additional operational complexity

**Effort**: Medium (3-5 days for setup + testing)

**Best for**: `/api/user/:id`, `/api/cart` (consistent, real-time)

### Option C: CloudFront CDN Edge Caching

Use AWS CloudFront to cache responses at edge locations.

**Pros:**
- Global distribution (low latency worldwide)
- Offloads origin traffic
- DDoS protection included
- Simple cache invalidation (by path)

**Cons:**
- Only works for GET requests
- Less granular control
- Header-based cache keys can be tricky
- Not suitable for user-specific data without custom logic
- Difficult to invalidate specific cache entries

**Effort**: Medium (3-4 days for configuration + testing)

**Best for**: `/api/products` (public, read-only, static)

### Option D: Multi-Tier Caching

Combine all three: In-memory L1 cache → Redis L2 cache → CloudFront CDN.

**Pros:**
- Maximum performance (best of all worlds)
- L1 cache handles hottest data (microseconds)
- L2 cache provides consistency (milliseconds)
- CDN reduces origin load globally

**Cons:**
- Most complex to implement and maintain
- Cache invalidation across tiers is tricky
- Highest operational overhead
- Over-engineering if not needed

**Effort**: High (1-2 weeks for full implementation)

**Best for**: High-scale systems (not needed yet)

### Rejected Options

- **Varnish Cache**: Rejected because we're on AWS with ALB (prefer native AWS services)
- **Memcached**: Rejected in favor of Redis (richer features, same cost)

## Proposal

**Recommended option**: **Hybrid B + C** (Redis for dynamic data, CloudFront for static)

**Rationale**:
1. **Redis (Option B)** for user-specific and session data (`/api/user/:id`, `/api/cart`)
   - Provides consistency across instances
   - Handles real-time invalidation
   - ~90% database load reduction for cached queries

2. **CloudFront (Option C)** for public product catalog (`/api/products`)
   - Offloads origin traffic entirely for popular products
   - Global distribution (future-proofing)
   - ~95% database load reduction for product queries

**Why not Option A**: Inconsistent cache across instances causes stale data issues.
**Why not Option D**: Over-engineering for current scale (500 req/sec).

## Implementation Considerations

**Phase 1**: Redis for user/cart data (highest impact)
- Set up AWS ElastiCache Redis cluster (t3.small)
- Implement cache-aside pattern in `src/models/`
- TTL: 5 minutes for user data, 1 minute for cart data
- Cache invalidation: On write operations (POST, PUT, DELETE)

**Phase 2**: CloudFront for product catalog
- Configure CloudFront distribution
- Set cache headers on `/api/products` responses
- TTL: 1 hour (products change infrequently)
- Invalidation: Manual invalidation on product updates (rare)

**Migration**: No breaking changes, additive only.

**Dependencies**: AWS ElastiCache (Redis 7.x), `redis` npm package (v4.x)

## Security Considerations

- **Redis**: Deploy in private subnet, no public access
- **Authentication**: Use AWS IAM for ElastiCache access
- **Sensitive data**: Do not cache responses with PII or payment info
- **CloudFront**: Configure signed URLs if product catalog becomes private

## Compatibility

- **Breaking changes**: None
- **Migration path**: Gradual rollout (can deploy without enabling cache)
- **Backwards compatibility**: Full (cache miss = database query, same behavior)

## Open Questions

1. Should we implement cache warming on deployment?
2. What monitoring/alerting do we need for cache hit rates?
3. Should we add cache versioning for easier invalidation?

## References

- Redis best practices: https://redis.io/docs/manual/patterns/
- AWS ElastiCache pricing: https://aws.amazon.com/elasticache/pricing/
- CloudFront caching: https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/
```

#### Step 7: AI Updates Index

AI updates `rfd-index.yaml`:

```yaml
next_number: 45
rfds:
  - number: 44
    title: API Caching Strategy
    state: Ideation
    path: .claude/rfd/0044-rfd-api-caching.md
    created: 2025-02-11
    updated: 2025-02-11
    authors:
      - AI Assistant
    labels:
      - performance
      - infrastructure
```

#### Step 8: AI Responds

```
✅ Created RFD 44: API Caching Strategy

📄 File: .claude/rfd/0044-rfd-api-caching.md
📊 State: Ideation

Based on your answers, I've explored four caching options:
- Option A: In-memory caching (simple, per-instance)
- Option B: Redis distributed cache (consistent, shared)
- Option C: CloudFront CDN (global, edge caching)
- Option D: Multi-tier (all three, complex)

**Recommended**: Hybrid approach (Redis + CloudFront) for best balance of
performance, consistency, and maintainability.

Next steps:
1. Review the options and rationale
2. Add any options I missed
3. When ready: "Promote RFD 44 to Discussion" to share with the team
4. After decision: "Create PRD from RFD 44" to plan implementation
```

---

### Example 2: State Promotion (Ideation → Discussion)

#### User Request

```
User: "Promote RFD 44 to Discussion"
```

#### AI Actions

1. **Read current RFD** at `.claude/rfd/0044-rfd-api-caching.md`
2. **Update state** in frontmatter: `state: Discussion`
3. **Move file**: `.claude/rfd/0044-rfd-api-caching.md` → `docs/rfd/0044.md`
4. **Update `rfd-index.yaml`**:
   ```yaml
   - number: 44
     title: API Caching Strategy
     state: Discussion
     path: docs/rfd/0044.md
     updated: 2025-02-12
   ```
5. **Check for `mkdocs.yml`** and offer to update navigation

#### AI Response

```
✅ Promoted RFD 44: API Caching Strategy to Discussion state

📄 New location: docs/rfd/0044.md
📊 State: Discussion
📅 Updated: 2025-02-12

The RFD is now publicly visible in documentation and ready for team discussion.

Would you like me to:
a) Create a GitHub discussion for structured feedback?
b) Update mkdocs.yml to add it to documentation navigation?
c) Both
d) Neither (I'll handle it manually)
```

---

### Example 3: RFD to PRD Transition

#### User Request

```
User: "We've decided on Option B (Redis + CloudFront) for RFD 44. Create a PRD for implementation."
```

#### AI Actions

1. **Update RFD 44 state** to "Published"
2. **Update frontmatter**: Ensure "Proposal" section confirms decision
3. **Update `rfd-index.yaml`**: Mark state as "Published"
4. **Create PRD** using `.claude/skills/create-prd/SKILL.md`
5. **Link RFD ↔ PRD**

#### AI Response

```
✅ Marked RFD 44 as Published with Redis + CloudFront decision

📄 Updated: docs/rfd/0044.md
📊 State: Published
✅ Decision: Hybrid approach (Redis for dynamic, CloudFront for static)

Now creating PRD for implementation...

---

[AI proceeds to create PRD using create-prd workflow]

PRD 5: Implement API Caching (Redis + CloudFront)

📄 File: .claude/tasks/0005-prd-api-caching.md
🔗 References: RFD 44 (decision rationale)

Next steps:
1. Review PRD requirements and acceptance criteria
2. "Generate tasks from PRD 5" to break down implementation
3. When implementation complete: "Mark RFD 44 as Committed"
```

---

### Example 4: Embedded Mode Suggestion

#### During Regular Conversation

**User:**
```
User: "I'm thinking about how to handle authentication. We could use JWT tokens or
session cookies. What do you think?"
```

**AI:**
```
Both are valid approaches. Let me outline the tradeoffs:

**JWT Tokens:**
✅ Stateless (no server-side storage)
✅ Works across microservices
✅ Good for API-first applications
✅ Mobile-friendly
❌ Can't invalidate easily (must wait for expiration)
❌ Larger payload (sent with every request)
❌ Security depends on client-side storage

**Session Cookies:**
✅ Easy to invalidate (server-side control)
✅ Smaller payload
✅ More secure (server-controlled)
✅ Mature ecosystem
❌ Requires session store (Redis, database)
❌ CSRF concerns (need CSRF tokens)
❌ Less suited for stateless APIs
❌ Trickier with microservices

---

I notice we're discussing multiple approaches with significant implications.
Would you like me to create an RFD to formally document these options?

This would help:
- Capture the pros/cons in more detail
- Document our decision rationale
- Consider hybrid approaches (e.g., JWT for API, sessions for web)
- Provide context for future developers

Shall I create **RFD 45: Authentication Strategy**?
```

**If User Says Yes:**
```
User: "Yes, create the RFD"

AI: "Great! Let me ask a few questions to make the RFD more specific..."

[Proceeds with discovery questions and RFD creation]
```

**If User Says No:**
```
User: "No, let's just go with JWT for now"

AI: "Understood. We'll use JWT tokens for authentication. Shall I implement..."

[Continues with implementation]
```

---

## State Transition Details

### State Transitions: Complete Matrix

| From | To | Trigger | Actions | File Move? |
|------|----|---------|---------| -----------|
| Prediscussion | Ideation | Clearer picture | Update state | No |
| Prediscussion | Abandoned | No longer relevant | Update state, add note | No |
| Ideation | Discussion | Ready for team | Update state, move file, update index | ✅ Yes |
| Ideation | Abandoned | No longer relevant | Update state, add note | No |
| Discussion | Published | Decision made | Update state, update proposal, update index | No |
| Discussion | Abandoned | Rejected | Update state, add note | No |
| Published | Committed | Implementation done | Update state, add links, update index | No |
| Published | Abandoned | Decision reversed | Update state, add note | No (rare) |

### Automation Checklist for Each Transition

#### Ideation → Discussion

**Checklist:**
- [ ] Read RFD file at `.claude/rfd/NNNN-rfd-topic.md`
- [ ] Verify state is "Ideation"
- [ ] Update frontmatter: `state: Discussion`
- [ ] Update frontmatter: `updated: YYYY-MM-DD`
- [ ] Create `docs/rfd/` directory if doesn't exist
- [ ] Move file to `docs/rfd/NNNN.md`
- [ ] Update `rfd-index.yaml`: `state: Discussion`, `path: docs/rfd/NNNN.md`
- [ ] Increment `updated` date in index
- [ ] Ask user about GitHub discussion / mkdocs.yml
- [ ] Confirm to user

**Error Handling:**
- If file doesn't exist: "RFD NNNN not found"
- If already in Discussion: "RFD NNNN is already in Discussion state"
- If state is Published/Committed: "Can't move backwards (use Abandon if needed)"

#### Discussion → Published

**Checklist:**
- [ ] Read RFD file at `docs/rfd/NNNN.md`
- [ ] Verify state is "Discussion"
- [ ] Verify "Proposal" section has recommendation (or explicit "seeking input" note)
- [ ] Update frontmatter: `state: Published`
- [ ] Update frontmatter: `updated: YYYY-MM-DD`
- [ ] Remove or answer "Open Questions" (or verify all answered)
- [ ] Update `rfd-index.yaml`: `state: Published`
- [ ] Increment `updated` date in index
- [ ] Confirm to user
- [ ] Suggest: "Create PRD from RFD NNNN?" (if implementation needed)

**Error Handling:**
- If no recommendation in Proposal: "Add recommendation to Proposal section first"
- If open questions exist: "Answer or remove open questions first (or note: 'Decided despite open questions')"

---

## Template Deep Dive

### Frontmatter Fields

```yaml
---
rfd: 44                          # RFD number (from rfd-index.yaml)
title: Short Descriptive Title   # Keep under 60 characters
authors:                         # List of authors
  - name: John Doe
  - name: Jane Smith
state: Ideation                  # One of: Prediscussion, Ideation, Discussion, Published, Committed, Abandoned
labels: [performance, security]  # Categorical tags for filtering
created: 2025-02-11             # ISO date format
updated: 2025-02-11             # ISO date format (update on each change)
discussion: https://...         # Optional: GitHub discussion link (for Discussion state)
related_prd: 0005-prd-feature.md # Optional: Link to PRD (for Published/Committed state)
---
```

### Section Purposes

#### Summary

**Purpose**: Give readers immediate context without reading the full document.

**Length**: 2-4 sentences (one paragraph).

**Content**: Problem + approach + outcome.

**Example:**
```markdown
## Summary

We need to implement caching for our API to reduce database load caused by high read
traffic. This RFD evaluates four caching approaches (in-memory, Redis, CDN, multi-tier)
with different tradeoffs for performance, cost, and complexity. Redis + CloudFront
hybrid is recommended for balancing consistency and global performance.
```

#### Problem Statement

**Purpose**: Define the problem clearly so readers understand why this matters.

**Content**:
- What's the problem?
- Why is it important?
- What happens if we do nothing?
- Quantify if possible (metrics, user impact)

**Example:**
```markdown
## Problem Statement

Our PostgreSQL database is experiencing high load during peak hours (>80% CPU), primarily
from read queries. Response times have increased from 50ms to 200-300ms over the past
month as user traffic grew from 200 req/sec to 500 req/sec.

This is important because:
- Slow response times hurt user experience (target: <100ms)
- Database is becoming a bottleneck for scaling
- Current RDS instance is near capacity (vertical scaling is expensive)

If we do nothing:
- Response times will continue degrading
- Database will require expensive vertical scaling ($500/month → $2000/month)
- Risk of outages during traffic spikes
```

#### Background

**Purpose**: Provide context that readers need to understand the proposal.

**Content**:
- Current state of the system
- Why this topic is being raised now
- Any previous attempts or related decisions
- Constraints or requirements

**Example:**
```markdown
## Background

**Current Tech Stack**: Node.js (Express), PostgreSQL, AWS (EC2, RDS)

**Infrastructure**: 3 EC2 t3.medium instances behind ALB, RDS PostgreSQL db.t3.large

**API Endpoints** (by traffic):
1. `/api/products` - 60% of traffic, read-only, can tolerate staleness
2. `/api/user/:id` - 20% of traffic, read-heavy, should be fresh
3. `/api/cart` - 15% of traffic, read/write balanced, must be real-time
4. `/api/orders` - 5% of traffic, write-heavy, no caching needed

**Previous Attempts**: None. This is the first time we're implementing caching.

**Why Now**: User traffic has doubled in the past 3 months. Database CPU consistently
hits 80-90% during peak hours (10am-2pm EST). We're planning a marketing campaign next
month that could double traffic again.

**Team Experience**: Team has used Redis in previous projects. Comfortable with AWS
managed services. No experience with Varnish or other specialized caching solutions.
```

#### Options Considered

**Purpose**: Document all viable options with honest pros/cons.

**Structure**: One subsection per option.

**Content per Option**:
- Clear description of the approach
- Pros (be specific, quantify if possible)
- Cons (be honest, include hidden costs)
- Effort estimate (Low/Medium/High or time estimate)

**Best Practices**:
- **Order by viability** (most viable first) OR **order by complexity** (simplest first)
- **Be specific**: "50ms vs 5ms" beats "faster"
- **Include edge cases**: "Works for X but not Y"
- **Note dependencies**: "Requires Z"

#### Rejected Options

**Purpose**: Document what was considered and ruled out early (prevents future "why didn't we consider X?" questions).

**Format**: Bullet list with reasons.

**Example:**
```markdown
### Rejected Options

- **Varnish Cache**: Rejected because we're on AWS with ALB. Varnish would require
  replacing ALB with a custom proxy layer (high complexity, low benefit).
- **Memcached**: Rejected in favor of Redis. Both have similar cost and performance,
  but Redis offers richer data structures (sets, sorted sets) that could be useful
  for future features like leaderboards.
- **Database Read Replicas**: Rejected as first approach because most queries are
  repetitive (90% cache hit rate expected). Read replicas would reduce load but
  not as effectively as caching.
```

#### Proposal

**Purpose**: State the recommendation (or explicitly note that none has been made yet).

**Content**:
- Recommended option (or "seeking input")
- Rationale for the recommendation
- Why not other options?

**If No Recommendation Yet:**
```markdown
## Proposal

> This RFD is seeking feedback. No recommendation has been made yet.

We're seeking input on:
1. Is the tradeoff analysis accurate?
2. Are there options we missed?
3. What would you prioritize: performance, cost, or simplicity?
```

**If Recommendation Made:**
```markdown
## Proposal

**Recommended option**: Hybrid B + C (Redis for dynamic data, CloudFront for static)

**Rationale**:
- Redis provides consistency across instances (solves Option A's weakness)
- CloudFront offloads static traffic entirely (reduces origin load by ~60%)
- Combined approach gives 90-95% database load reduction
- Cost is reasonable ($100/month vs $1500/month for read replicas)

**Why not Option A (in-memory)**: Inconsistent cache across instances causes stale data.
**Why not Option D (multi-tier)**: Over-engineering for current scale (500 req/sec).
```

#### Implementation Considerations

**Purpose**: High-level implementation guidance (not detailed HOW, that's for PRD).

**Content**:
- Key technical considerations
- Migration needs
- Dependencies or prerequisites
- Phasing (if multi-phase)

**Example:**
```markdown
## Implementation Considerations

**Phase 1**: Redis for user/cart data (highest impact, 2-3 days)
- Set up AWS ElastiCache Redis cluster (t3.small, single-AZ)
- Implement cache-aside pattern in `src/models/User.js` and `src/models/Cart.js`
- TTL: 5 minutes for user data, 1 minute for cart data
- Cache invalidation: On write operations (POST, PUT, DELETE to user/cart endpoints)

**Phase 2**: CloudFront for product catalog (1-2 days)
- Configure CloudFront distribution pointing to ALB
- Set `Cache-Control` headers on `/api/products` responses
- TTL: 1 hour (products change infrequently, ~5 times/day)
- Invalidation: Manual CloudFront invalidation on product updates (rare operation)

**Migration**: No breaking changes. Cache misses fall back to database (same behavior).
Can deploy without enabling cache initially (dark launch for testing).

**Dependencies**:
- AWS ElastiCache (Redis 7.x)
- `redis` npm package (v4.x)
- CloudFront distribution (new resource)

**Monitoring**:
- CloudWatch metrics for cache hit rates
- Redis connection pool monitoring
- Database load before/after (expect 80% → 20% CPU)
```

#### Security Considerations

**Purpose**: Flag any security implications of the decision.

**Content**:
- New attack surfaces
- Data handling changes
- Authentication/authorization impacts
- Compliance requirements

**If No Security Impact:**
```markdown
## Security Considerations

No significant security implications. Cache will not store sensitive data (no PII,
no payment info). All cached data is already accessible via public API endpoints.

**Note**: Redis will be deployed in private subnet with no public access.
```

**If Security Impact Exists:**
```markdown
## Security Considerations

**Cache Poisoning Risk**: Malicious requests could populate cache with bad data.
Mitigation: Validate and sanitize all data before caching.

**Sensitive Data**: Do NOT cache responses containing:
- PII (user emails, addresses, phone numbers)
- Payment information (credit cards, billing details)
- Session tokens or API keys

**Redis Security**:
- Deploy in private subnet, no public access
- Use AWS IAM authentication (not password-based)
- Enable encryption at rest and in transit

**CloudFront**:
- Configure signed URLs if product catalog becomes private in the future
- Set appropriate `Cache-Control` headers to prevent caching of user-specific data
```

#### Compatibility

**Purpose**: Document breaking changes, migration needs, and backwards compatibility.

**Content**:
- Breaking changes (yes/no, what breaks)
- Migration path (if needed)
- Backwards compatibility (what's preserved)

**Example:**
```markdown
## Compatibility

- **Breaking changes**: None. Cache is additive only.
- **Migration path**: No migration needed. New code paths are opt-in.
- **Backwards compatibility**: Full. If cache is unavailable, system falls back to
  database queries (same behavior as before caching).
- **API contract**: No changes to request/response formats.
```

#### Open Questions

**Purpose**: Document questions that need answers before deciding.

**Content**: Numbered list of questions.

**Note**: This section should be empty or removed when RFD moves to "Published" state.

**Example:**
```markdown
## Open Questions

1. Should we implement cache warming on deployment to avoid cold start performance?
2. What monitoring/alerting thresholds should we set for cache hit rates?
3. Should we add cache versioning (e.g., `cache:v1:user:123`) for easier invalidation?
4. Do we need a cache admin UI for manual invalidation during incidents?
5. Should we cache 404 responses (to prevent repeated database queries for nonexistent data)?
```

#### References

**Purpose**: Link to related resources for further reading.

**Content**:
- Related RFDs (internal)
- External resources (documentation, benchmarks, case studies)
- Related PRDs (if implementation has started)

**Example:**
```markdown
## References

**Related RFDs**:
- RFD 12: Database Scaling Strategy (related discussion)

**External Resources**:
- [Redis Best Practices](https://redis.io/docs/manual/patterns/)
- [AWS ElastiCache Pricing](https://aws.amazon.com/elasticache/pricing/)
- [CloudFront Caching Behavior](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/Expiration.html)
- [Cache-Aside Pattern](https://docs.microsoft.com/en-us/azure/architecture/patterns/cache-aside)

**Related PRDs**:
- PRD 5: Implement API Caching (implementation plan)

**Implementation**:
- [GitHub PR #456](https://github.com/org/repo/pull/456) - Redis cache layer
- [GitHub PR #457](https://github.com/org/repo/pull/457) - CloudFront distribution
```

---

## Tips for AI Assistants

### Do's

1. ✅ **Ask discovery questions before drafting**: Understand the problem space first.
2. ✅ **Research multiple options**: Even if user suggests one, explore alternatives.
3. ✅ **Be honest about tradeoffs**: Every option has cons. Document them.
4. ✅ **Quantify when possible**: "50ms vs 200ms" beats "faster vs slower".
5. ✅ **Include rejected options**: Future readers need to know what was considered.
6. ✅ **Reference existing context**: Check project.md, patterns.md, other RFDs.
7. ✅ **Suggest RFDs during discussions**: Use embedded mode when appropriate.
8. ✅ **Update rfd-index.yaml**: Always keep the index current.

### Don'ts

1. ❌ **Don't start implementation**: RFD is for discussion, not action.
2. ❌ **Don't force a recommendation**: Seeking input is valid ("no decision yet").
3. ❌ **Don't skip the index update**: rfd-index.yaml must be current.
4. ❌ **Don't only document the chosen option**: Document ALL considered options.
5. ❌ **Don't wait for perfection**: Share early, refine together.
6. ❌ **Don't combine multiple decisions**: One decision per RFD (split if needed).
7. ❌ **Don't abandon without explanation**: Always note why if abandoning.

### Common Pitfalls

**Pitfall 1: Not asking enough questions**
- **Problem**: AI generates RFD without understanding constraints.
- **Solution**: Always ask 3-5 discovery questions before drafting.

**Pitfall 2: Only documenting the "winning" option**
- **Problem**: Future readers wonder "why didn't we consider X?"
- **Solution**: Document 2-4 options, plus rejected options.

**Pitfall 3: Being vague about tradeoffs**
- **Problem**: "Option A is faster" (faster than what? by how much?)
- **Solution**: Quantify: "Option A: 50ms, Option B: 200ms"

**Pitfall 4: Forgetting to update state**
- **Problem**: RFDs stuck in "Discussion" forever.
- **Solution**: Remind user to promote state when decision is made.

**Pitfall 5: Scope creep**
- **Problem**: RFD covers multiple unrelated decisions.
- **Solution**: Split into multiple RFDs if covering >1 decision.

---

**End of Process Guide**

For the concise overview, see the main SKILL.md file.
