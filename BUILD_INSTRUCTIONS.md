# Build Instructions

> This file must be read before starting any build step. It contains the rules and preferences for how we develop Hamster Wheel.

---

## Development Process

1. **Incremental builds only.** One small step at a time. Complete it, present for review, refine if needed, then move on.
2. **Every step must include unit tests.** No code is considered complete without full test coverage.
3. **No skipping ahead.** Each step builds on the last. Do not combine multiple steps.
4. **Dependencies first.** Always build prerequisites/dependencies before the thing that depends on them. (e.g., CRUD layer before scheduler, scheduler before UI wiring.)
5. **Think in deliverable features.** Each major feature (job fetching, job matching, document tailoring) is a deliverable. A deliverable is only complete when it works end-to-end including the UI. Individual steps within a deliverable are reviewed incrementally, but stakeholder sign-off happens when the full feature is working in the UI.
6. **Update PROGRESS.md after every step.** This is mandatory — it's how fresh context windows catch up. See the [Documentation Updates](#documentation-updates) section below.

## Starting a New Session

Every time a new AI agent (or new context window) begins work on this project, it MUST:

1. **Read `PLAN.md`** — to understand what we're building and the full architecture.
2. **Read `BUILD_INSTRUCTIONS.md`** — to understand how we build (this file).
3. **Read `PROGRESS.md`** — to understand what's already built and what comes next.

This replaces the need to explore the entire codebase from scratch. `PROGRESS.md` is kept current after every step, so it's always the fastest way to orient.

## Documentation Updates

**PROGRESS.md must be updated after every completed step.** This is not optional.

After each step is approved and committed, update `PROGRESS.md` with:

1. **Move the completed step** from "Next Steps" to the appropriate "Completed" section.
2. **Add details** about what was built: files created/modified, key decisions made.
3. **Update "Next Steps"** to reflect what should be built next.
4. **Update any counts** (test counts, file counts, etc.) if they changed.

This ensures the next context window (or the next AI session) can pick up exactly where you left off without re-analyzing the entire codebase.

**When to update other docs:**
- **PLAN.md** — Only when architecture decisions change (new adapter, changed data model, etc.). Don't update for routine implementation work.
- **BUILD_INSTRUCTIONS.md** — Only when process rules change or new patterns are established.

## Build Order (within a deliverable)

1. Data layer (DB CRUD for the feature's entities)
2. Business logic (adapters, services, schedulers)
3. Wails bindings (expose Go functions to the frontend)
4. React UI (components, pages, wiring)
5. E2E verification (stakeholder reviews the working feature)

## Code Standards

- Write clean, well-structured code from the start.
- Architecture must be extensible where the plan calls for it (job source adapters, LLM providers).
- Follow Go conventions and idioms.
- Follow React/TypeScript conventions for the frontend.

## Definition of Done

A step is complete when ALL of the following are true:

- [ ] Code written and follows standards
- [ ] All unit tests pass (`go test ./...`)
- [ ] Test coverage meets requirements (>90% for business logic)
- [ ] No linter warnings (`golangci-lint run` or `go vet ./...`)
- [ ] Error handling is explicit and comprehensive
- [ ] Logging added for important operations
- [ ] Complex logic has inline comments explaining the "why"
- [ ] Code reviewed and approved by stakeholder
- [ ] Changes committed to git with descriptive message
- [ ] **PROGRESS.md updated** with what was built and what's next

**Do not proceed to the next step until all checkboxes are satisfied.**

## Error Handling Standards

Go's explicit error handling is a feature, not a bug. Embrace it.

### Rules
1. **Never ignore errors.** Every `err` return must be checked.
   ```go
   // ❌ BAD
   db.Insert(job)
   
   // ✅ GOOD
   if err := db.Insert(job); err != nil {
       return fmt.Errorf("failed to insert job: %w", err)
   }
   ```

2. **Wrap errors with context** using `%w` for the error chain.
   ```go
   if err := fetchJobs(source); err != nil {
       return fmt.Errorf("failed to fetch jobs from %s: %w", source.Name, err)
   }
   ```

3. **Custom errors for business logic** that needs special handling.
   ```go
   var ErrJobNotFound = errors.New("job not found")
   var ErrDuplicateJob = errors.New("job already exists")
   ```

4. **Sentinel errors** should be exported and used with `errors.Is()`.
   ```go
   if errors.Is(err, ErrJobNotFound) {
       // handle specifically
   }
   ```

5. **Log before returning** errors up the stack (but don't log at every layer).
   ```go
   if err := criticalOperation(); err != nil {
       log.Error().Err(err).Msg("critical operation failed")
       return err
   }
   ```

6. **UI gets friendly messages**, not raw error strings.
   ```go
   // In Wails binding layer
   if err := service.DoThing(); err != nil {
       return "", fmt.Errorf("Unable to process your request. Please try again.")
   }
   ```

### Python Comparison
- Python: Try/except blocks, raise exceptions
- Go: Explicit `if err != nil` checks, return errors as values
- Go errors are values that flow through your program, not exceptions that jump the stack

## Logging Conventions

### Use Structured Logging
- **Library:** Use Go's standard `log/slog` package (built-in, structured, performant)
- **Levels:** DEBUG → INFO → WARN → ERROR
  - **DEBUG:** Development details, verbose tracing
  - **INFO:** Important events (job fetched, match found, document generated)
  - **WARN:** Recoverable issues (API rate limit, retry attempted)
  - **ERROR:** Failures that prevent operations

### What to Log
```go
// ✅ GOOD - structured with context
slog.Info("job fetched successfully",
    "source", "LinkedIn",
    "job_id", jobID,
    "duration_ms", elapsed.Milliseconds())

slog.Error("failed to tailor document",
    "job_id", jobID,
    "error", err)

// ❌ BAD - unstructured, no context
log.Println("Got a job!")
log.Println("Error:", err)
```

### What NOT to Log
- API keys, tokens, passwords
- Full user resumes or personal data (use IDs instead)
- Sensitive company information from job listings

### Setup
```go
// main.go
func initLogger() {
    handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo, // Change to LevelDebug during dev
    })
    slog.SetDefault(slog.New(handler))
}
```

## Database Migration Strategy

SQLite schema changes must be versioned and trackable.

### Approach
1. **Migration files** in `internal/db/migrations/`
   - `001_initial_schema.sql`
   - `002_add_job_source.sql`
   - `003_add_matching_scores.sql`

2. **Version tracking table**
   ```sql
   CREATE TABLE IF NOT EXISTS schema_version (
       version INTEGER PRIMARY KEY,
       applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
   );
   ```

3. **Auto-apply on startup**
   ```go
   // In db initialization
   func (db *DB) ApplyMigrations() error {
       currentVersion := db.getCurrentVersion()
       for _, migration := range availableMigrations {
           if migration.Version > currentVersion {
               if err := db.applyMigration(migration); err != nil {
                   return err
               }
           }
       }
       return nil
   }
   ```

4. **Test migrations** on a copy before applying to your dev database
   ```bash
   cp hamster_wheel.db hamster_wheel_backup.db
   # Test new code
   # If it works, delete backup. If not, restore.
   ```

### Alternative (Simpler for Early Development)
- Document schema in `schema.sql`
- Use `CREATE TABLE IF NOT EXISTS` for idempotency
- Add columns with `ALTER TABLE` wrapped in existence checks
- Transition to proper migrations before production

## Git Practices

### Commit Strategy
- **One approved step = one commit**
- Commit immediately after stakeholder approval
- Keep commits atomic and focused

### Commit Message Format
```
[CATEGORY] Brief description

Optional longer explanation if needed.

Examples:
[DB] Add jobs CRUD layer with SQLite implementation
[SERVICE] Implement job matching algorithm
[UI] Add job list component with filtering
[TEST] Add integration tests for job fetcher
[FIX] Handle edge case in document tailoring
[REFACTOR] Extract common DB helpers
```

Categories: `DB`, `SERVICE`, `API`, `UI`, `TEST`, `FIX`, `REFACTOR`, `DOCS`, `CONFIG`

### Branch Strategy (if using branches)
- `main` = stable, working code
- `dev` = current development work
- Feature branches optional for complex work

### What to Commit
- Source code and tests
- Configuration files
- Documentation
- `go.mod` and `go.sum` (dependency manifests)

### What NOT to Commit
- `hamster_wheel.db` (actual data)
- `.env` files with secrets
- Binary artifacts
- IDE-specific files (add to `.gitignore`)

## Dependency Management

### Principles
1. **Prefer standard library** when reasonable (it's excellent in Go)
2. **Pin versions** in `go.mod` - never use `latest`
3. **Minimize dependencies** - each one is a maintenance burden
4. **Vet dependencies** before adding:
   - Is it actively maintained?
   - Does it have good documentation?
   - What's its license?

### Adding Dependencies
```bash
# Add a specific version
go get github.com/user/repo@v1.2.3

# Update go.mod and go.sum
go mod tidy

# Vendor if you want reproducible builds (optional)
go mod vendor
```

### Document Major Dependencies
In `PLAN.md` or inline comments, note why you chose each significant dependency:
```go
// Using github.com/jmoiron/sqlx for ergonomic SQL
// Chosen over: manual scanning (too verbose), GORM (too heavy)
```

### Current Expected Dependencies
- **Wails:** Application framework (required)
- **SQLite driver:** `modernc.org/sqlite` (pure Go, no CGO)
- **LLM SDK:** Anthropic's official Go SDK
- **HTTP client:** Standard library `net/http`
- **Testing:** Standard library `testing`

## Code Organization Patterns

### File Naming
- **Use lowercase with underscores:** `job_matcher.go`, not `jobMatcher.go`
- **Test files:** `*_test.go`
- **Example:** `job_matcher.go` → `job_matcher_test.go`

### Package Naming
- **Short, lowercase, singular:** `job`, `matcher`, `db`
- **Not:** `jobs`, `JobPackage`, `job_pkg`
- **Avoid:** `utils`, `helpers`, `common` (be specific)

### Interface Naming
- **Usually ends in `-er`:** `Fetcher`, `Matcher`, `Scheduler`, `Tailer`
- **Describes behavior:** `Reader`, `Writer`, `Closer`
- **Keep interfaces small:** 1-3 methods is ideal (Go philosophy)

### Struct Naming
```go
// ✅ GOOD
type JobMatcher struct { ... }
type LinkedInAdapter struct { ... }
type DocumentTailorer struct { ... }

// ❌ BAD
type JobMatcherService struct { ... }  // "Service" is noise
type IJobMatcher struct { ... }        // No "I" prefix
```

### Method Receivers
```go
// Use pointer receivers when:
// - Method modifies the struct
// - Struct is large (>few fields)
// - Consistency (if any method needs pointer, use it for all)

// ✅ GOOD - modifying state
func (m *JobMatcher) UpdateScores() error

// ✅ GOOD - read-only but consistent with other methods
func (m *JobMatcher) GetScore(jobID string) float64

// ✅ GOOD - small read-only struct
func (j Job) IsActive() bool
```

### Error Variables
```go
// Exported errors (used by other packages)
var ErrJobNotFound = errors.New("job not found")

// Unexported errors (internal only)
var errInvalidInput = errors.New("invalid input")
```

## Common Go Gotchas

Track these as you encounter them. Here are the most common:

### 1. Forgetting to Close Resources
```go
// ❌ BAD
func queryDB() error {
    rows, err := db.Query("SELECT * FROM jobs")
    if err != nil {
        return err
    }
    // If error happens below, rows never closes!
    // Process rows...
    rows.Close()
}

// ✅ GOOD
func queryDB() error {
    rows, err := db.Query("SELECT * FROM jobs")
    if err != nil {
        return err
    }
    defer rows.Close()  // Guaranteed to run
    // Process rows...
}
```

### 2. Pointer vs Value Receivers
```go
// This modifies a COPY, not the original
func (j Job) Activate() {
    j.Active = true  // Original Job unchanged!
}

// This modifies the original
func (j *Job) Activate() {
    j.Active = true  // Original Job updated ✅
}
```

### 3. Range Loop Variable Capture
```go
// ❌ BAD - all goroutines see the same job
for _, job := range jobs {
    go func() {
        process(job)  // 'job' variable is shared!
    }()
}

// ✅ GOOD - each goroutine gets its own copy
for _, job := range jobs {
    job := job  // Shadow the variable (or pass as param)
    go func() {
        process(job)
    }()
}

// ✅ BETTER (Go 1.22+) - loop variables are per-iteration
for _, job := range jobs {
    go func() {
        process(job)  // Safe in Go 1.22+
    }()
}
```

### 4. Nil Slice vs Empty Slice
```go
var nilSlice []string        // nil, len=0
emptySlice := []string{}     // not nil, len=0
jsonSlice := make([]string, 0)  // not nil, len=0

// Both have len=0, but JSON encoding differs:
json.Marshal(nilSlice)   // "null"
json.Marshal(emptySlice) // "[]"

// Prefer: make([]string, 0) when returning to frontend
```

### 5. Goroutine Leaks
```go
// ❌ BAD - goroutine runs forever if channel never receives
go func() {
    for msg := range ch {
        process(msg)
    }
}()

// ✅ GOOD - use context for cancellation
go func() {
    for {
        select {
        case msg := <-ch:
            process(msg)
        case <-ctx.Done():
            return
        }
    }
}()
```

## When You're Stuck

### If a Step Isn't Working
1. **Explain the issue clearly**
   - What were you trying to do?
   - What happened instead?
   - What's the error message? (full text)
   
2. **Share relevant context**
   - Code snippet that's failing
   - Test output
   - Logs or stack traces

3. **Suggest alternatives** if you're uncertain
   - "Should we approach this differently?"
   - "Is there a simpler way to achieve this?"

4. **It's okay to revise** before moving forward
   - Better to fix architecture issues now than later
   - Technical debt compounds quickly

### If You're Unsure About Design
- Ask before implementing
- Reference `PLAN.md` for architectural guidance
- Propose options: "I see two approaches: A or B. Here are the tradeoffs..."

### If Tests Are Failing
- Don't skip or comment them out
- Failing tests mean the code isn't ready
- Fix the code or fix the test expectations

### If You Need to Learn a Go Concept
- Ask for explanation with Python comparison
- Request code examples
- It's expected - you're learning Go through this project

## Performance Expectations

### Initial Benchmarks (Establish After Basic Implementation)
- Job fetching: <5s for 50 jobs per source
- Job matching: <2s for 100 jobs against user profile
- Document tailoring: <30s per document (LLM-dependent)
- Database queries: <100ms for typical CRUD operations
- UI responsiveness: <200ms for user interactions

### When to Optimize
- **Not yet.** Build for correctness first.
- Profile before optimizing (use Go's `pprof`)
- Only optimize hot paths (measure with benchmarks)
- Document performance-critical sections

### When to Worry
- Database queries taking >1s
- UI freezing during operations
- Memory leaks (use `go test -memprofile`)
- Goroutines not terminating

**Rule:** Make it work, make it right, make it fast - in that order.

## Testing Strategy

### Unit Tests
- **Go tests live next to the source files** (Go standard practice).
  - `internal/db/sqlite.go` → `internal/db/sqlite_test.go`
- Test files use the same package as the source (white-box testing, can access unexported functions).
- Use `t.TempDir()` for test databases and files (auto-cleanup).
- Use a shared `testDB(t)` helper per package for common setup.
- **Test all likely edge cases:** empty inputs, duplicates, missing data, error conditions, boundary values, concurrent access where relevant.
- Aim for >80% coverage on business logic; 100% on critical paths (matching, tailoring).

### Integration Tests
- Test multi-layer interactions (e.g., service → DB, adapter → external API).
- Use real database instances (in temp directories) not mocks.
- Can live in `*_integration_test.go` files or separate `test/integration/` folder.
- Mark with build tags if they're slow: `//go:build integration`

### E2E Tests
- Validate complete user workflows through the UI.
- Only needed at deliverable milestones, not every step.
- Manual testing is acceptable for early iterations; automate later if value justifies it.

## Teaching Context

- The developer is learning Go as part of this project.
- They are proficient in Python and React/TypeScript.
- When writing Go code, explain new concepts using Python comparisons where helpful.
- Don't over-explain things they already know from Python/React (e.g., basic loops, conditionals, HTTP concepts).
- Focus explanations on Go-specific patterns: interfaces, goroutines, error handling, packages, pointers, structs, etc.

## Review Protocol

- After each step, present:
  1. What was built
  2. The code written (with Go concepts explained)
  3. The tests written
  4. How to run/verify it
  5. The PROGRESS.md update (so stakeholder can verify it's accurate)
- Wait for explicit approval before proceeding to the next step.

## Reference

- **`PLAN.md`** — What to build (architecture, features, data model). Single source of truth for design.
- **`BUILD_INSTRUCTIONS.md`** — How to build (this file). Process rules, code standards, conventions.
- **`PROGRESS.md`** — What's been built and what's next. Updated after every step. **Read this first when starting a new session.**

## Quick Reference

### Essential Commands
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests verbosely
go test -v ./...

# Run integration tests only
go test -tags=integration ./...

# Run linter
golangci-lint run
# or
go vet ./...

# Format code
go fmt ./...

# Update dependencies
go mod tidy

# Build the application
wails build

# Run in dev mode
wails dev
```

### Project Structure Quick Reminder
```
hamster-wheel/
├── internal/
│   ├── db/              # Database layer (CRUD)
│   ├── service/         # Business logic
│   ├── adapter/         # External integrations (job sources, LLMs)
│   └── scheduler/       # Background tasks
├── frontend/
│   └── src/
│       ├── components/  # React components
│       └── pages/       # Main views
├── migrations/          # Database schema versions
└── wails.json          # Wails configuration
```

### Key Resources
- **Go Documentation:** https://go.dev/doc/
- **Go by Example:** https://gobyexample.com/
- **Wails Docs:** https://wails.io/docs/introduction
- **React Docs:** https://react.dev/
- **SQLite Docs:** https://sqlite.org/docs.html

### Go vs Python Quick Lookup
| Concept | Python | Go |
|---------|--------|-----|
| Error handling | `try/except` | `if err != nil` |
| Null value | `None` | `nil` |
| String formatting | f-strings | `fmt.Sprintf()` |
| Loops | `for x in list:` | `for _, x := range list {` |
| Class | `class Job:` | `type Job struct {` |
| Method | `def method(self):` | `func (j *Job) Method() {` |
| Constructor | `__init__` | Constructor function `NewJob()` |
| Inheritance | Classes inherit | Interfaces + composition |
| Async | `async/await` | `go func() {}` (goroutines) |
| Package import | `import module` | `import "package"` |

---

*This file will be updated as we establish more patterns and preferences during development.*
