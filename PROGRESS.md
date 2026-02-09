# Progress Tracker

> This file is the fastest way to understand where the project stands.
> **Updated after every completed step.** Read this first when starting a new session.
>
> For full architecture details, see [PLAN.md](PLAN.md).
> For build process rules, see [BUILD_INSTRUCTIONS.md](BUILD_INSTRUCTIONS.md).

---

## Current Phase: Phase 1 — Foundation (MVP)

**Goal:** App runs, polls Reed UK via API, stores jobs, displays them in UI.

**Status:** 100% complete! ✅ All Phase 1 features are implemented and tested. Ready for initial git commit.

---

## What's Been Built

### Go Backend

#### Database Layer — `internal/db/`

| File | Purpose |
|------|---------|
| `sqlite.go` | Opens SQLite DB at OS config dir, enables WAL mode + foreign keys |
| `migrations.go` | Schema versioning via `PRAGMA user_version`, auto-applies on startup |
| `jobs.go` | Full CRUD for jobs: Create, Get, List, ListByFilter, Count, Delete |
| `filters.go` | Full CRUD for search filters: Create, Get, List, Update, Delete |
| `settings.go` | Key-value settings: Get, Set |
| `errors.go` | Sentinel errors: `ErrNotFound`, `ErrDuplicateJob` |
| `*_test.go` | Unit tests for all modules above |

**Schema (Migration 1):** 6 tables — `search_filters`, `jobs`, `job_matches`, `llm_prompts`, `approved_domains`, `settings`. Jobs have UNIQUE(source, source_id) for deduplication.

**Note:** The `search_filters` table has a default source of `indeed_uk` in the migration SQL. This is harmless — the application code passes `reed_uk` when creating filters. Existing filters from before the switch will show the old default but function correctly.

#### Adapter System — `internal/adapter/`

| File | Purpose |
|------|---------|
| `adapter.go` | `Adapter` interface: `Name()`, `DisplayName()`, `FetchNewJobs()`, `FetchJobDetails()`, `Validate()` |
| `registry.go` | Thread-safe adapter registry (sync.RWMutex) with Register/Get/List |
| `reed/reed.go` | Reed UK adapter: HTTP Basic Auth, JSON API, 2s rate limiting, salary formatting |
| `reed/reed_test.go` | 44 tests: auth, fetch, errors, salary, empty results, dedup determinism |

**Reed adapter details:**
- Name: `reed_uk`, Display: `Reed UK`
- API base: `https://www.reed.co.uk/api/1.0`
- Auth: HTTP Basic (API key as username, empty password)
- `FetchNewJobs` → GET `/search?keywords=...&location=...`
- `FetchJobDetails` → GET `/jobs/{id}`
- Uses `externalUrl` from Reed when available
- `SetAPIKey()` allows runtime key updates from UI

#### Scheduler — `internal/scheduler/`

| File | Purpose |
|------|---------|
| `scheduler.go` | Background polling loop: concurrent fetch (goroutine per filter), sequential DB writes, panic recovery |
| `scheduler_test.go` | Tests for polling, pause/resume, dedup, error handling |

**Key design:** Phase 1 (concurrent reads) fetches from all enabled filters in parallel via goroutines. Phase 2 (sequential writes) inserts results into SQLite one at a time to avoid write contention. Supports pause/resume, manual `PollOnce()`, tracks next poll time.

#### Wails v3 Bindings — Root

| File | Purpose |
|------|---------|
| `app.go` | `AppService` struct with all frontend-exposed methods — v3 service pattern with `ServiceStartup/ServiceShutdown` |
| `main.go` | Entry point: structured logging, DB init, Reed adapter, scheduler, **system tray setup**, Wails v3 run |
| `assets/iconTemplate.png` | 22x22 template icon for system tray (adapts to light/dark mode) |

**Exposed to frontend (via Wails v3 bindings):**
- Jobs: `GetJobs`, `GetJob`, `GetJobsByFilter`, `GetJobCount`, `DeleteJob`
- Filters: `GetFilters`, `GetFilter`, `CreateFilter`, `UpdateFilter`, `DeleteFilter`
- Polling: `PollNow`, `GetPollingStatus`, `SetPollingPaused`
- Settings: `GetReedAPIKey`, `SetReedAPIKey`

**System Tray:**
- **Icon & Label:** "HW" label with template icon (auto-adapts to macOS dark mode)
- **Menu Items:**
  - "Open Hamster Wheel" — shows and focuses main window
  - "Pause/Resume Monitoring" — toggles scheduler pause state (label updates dynamically)
  - "Quit" — exits application
- **Hide-to-tray:** Window close button hides window (app stays running via tray)

### React Frontend — `frontend/src/`

#### Components

| Component | File | Purpose |
|-----------|------|---------|
| App | `App.tsx` | Root layout — three-pane split (filters / jobs / detail) |
| Header | `components/Header.tsx` | Top bar: job count, poll button, pause toggle, next-poll timer |
| FilterPanel | `components/FilterPanel.tsx` | Left pane: list filters, create new, toggle enable, delete |
| FilterCard | `components/FilterCard.tsx` | Single filter card with toggle + delete |
| CreateFilterForm | `components/CreateFilterForm.tsx` | Form for new search filters |
| JobList | `components/JobList.tsx` | Center pane: scrollable job cards, filter-by-filter dropdown |
| JobCard | `components/JobCard.tsx` | Single job row: title, company, location, discovered time |
| JobDetail | `components/JobDetail.tsx` | Right pane: full job details, description, open URL, delete |
| APIKeyInput | `components/APIKeyInput.tsx` | Reed API key entry field |
| PollResultToast | `components/PollResultToast.tsx` | Toast notification after manual poll |
| ErrorBanner | `components/ErrorBanner.tsx` | Dismissable error bar |
| EmptyState | `components/EmptyState.tsx` | Shown when no jobs/filters exist |

#### Hooks & Utilities

| File | Purpose |
|------|---------|
| `hooks/useJobs.ts` | Fetches jobs, delete, refresh — wraps Wails bindings |
| `hooks/useFilters.ts` | CRUD for filters — wraps Wails bindings |
| `lib/format.ts` | Date/time formatting ("2h ago"), text truncation |
| `lib/sanitize.ts` | HTML sanitization for job descriptions |

#### Test Coverage

- 22 frontend test files (Vitest + React Testing Library)
- Every component and hook has a corresponding `.test.tsx` / `.test.ts` file

### Configuration

| File | Purpose |
|------|---------|
| `wails.json` | Wails v3 config: frontend build commands, dev server URL |
| `go.mod` | Go 1.24, **Wails v3.0.0-alpha.70**, modernc.org/sqlite v1.44.3, google/uuid v1.6.0 |
| `frontend/package.json` | React 18, Vite, Tailwind CSS, Vitest, **@wailsio/runtime** |
| `frontend/tsconfig.json` | `allowJs: true` to support v3 JS bindings with JSDoc types |
| `frontend/bindings/` | Auto-generated v3 bindings: `appservice.js`, model classes |
| `.gitignore` | Ignores binaries, node_modules, dist, .db files, IDE files |
| `LICENSE` | Apache 2.0 |

---

## What's NOT Built Yet

### Remaining Phase 1
- [x] **System tray** — ✅ DONE: tray icon, menu (Open/Pause-Resume/Quit), hide-to-tray
- [x] **Wails v2 → v3 migration** — ✅ DONE: service pattern, v3 bindings, system tray support
- [ ] **Initial git commit** — all code is currently untracked

### Phase 2: LLM Matching
- [ ] Keychain manager (`internal/keychain/`) — store/retrieve API keys via OS keychain
- [ ] LLMProvider interface + provider registry (`internal/llm/`)
- [ ] Claude provider implementation (`internal/llm/claude/`)
- [ ] CV PDF parser (`internal/cv/parser.go`)
- [ ] Job matching logic — provider-agnostic, uses LLMProvider.SendMessage (`internal/llm/matcher.go`)
- [ ] Integrate matching into poll cycle (scheduler calls matcher after job fetch)
- [ ] OS-native notifications for high matches (`internal/notifier/`)
- [ ] Match score display in UI (job list badges, detail view)
- [ ] Match threshold configuration in settings

### Phase 3: Document Tailoring
- [ ] CV tailoring prompt and flow
- [ ] Cover letter tailoring prompt and flow
- [ ] AI detection self-check loop
- [ ] PDF generation for tailored documents
- [ ] Document preview in UI
- [ ] Download + regenerate buttons

### Phase 4: Full UI & Polish
- [ ] Setup wizard (first-run experience)
- [ ] Full split-view dashboard
- [ ] Status tracking pipeline (status transitions)
- [ ] Kanban board view
- [ ] Complete settings page
- [ ] Custom instructions + prompt customization UI
- [ ] Dark mode, keyboard shortcuts, loading/error states

### Phase 5: Distribution
- [ ] macOS .dmg build
- [ ] Windows .msi build
- [ ] GitHub Actions CI
- [ ] User documentation

---

## Next Steps (in order)

The next piece of work should follow the build order in BUILD_INSTRUCTIONS.md (data layer → business logic → bindings → UI).

### 1. Initial Git Commit (Phase 1 completion)
✅ **Phase 1 is complete!** All code is untracked. Make the first commit to establish the baseline.

### 2. Begin Phase 2: LLM Matching
Start with the data layer: keychain manager for secure API key storage, then LLMProvider interface and Claude implementation.

---

## Key Architecture Decisions

These decisions have already been made and implemented. New sessions should not re-decide them.

1. **Reed UK instead of Indeed** — Indeed decommissioned their RSS feeds. Reed provides a JSON REST API with free API keys.
2. **Wails v3** — Migrated from v2 to v3 for native system tray support. Uses service pattern, generated JS bindings with JSDoc types.
3. **API key in DB settings** — Phase 1 stores the Reed API key in the `settings` table (plaintext). Phase 2 will move to OS keychain via `zalando/go-keyring`.
4. **Concurrent fetch, sequential write** — Scheduler fetches from multiple filters in parallel goroutines but writes to SQLite sequentially to avoid write contention.
5. **Deduplication via UNIQUE(source, source_id)** — Each job is identified by its source adapter name + a deterministic hash. Duplicates are rejected at the DB level.
6. **Pure Go SQLite** — Using `modernc.org/sqlite` (no CGO required) for easy cross-compilation.
7. **Three-pane layout** — Filters (left) | Job List (center) | Job Detail (right). Not the split view described in PLAN.md section 4.5 yet — that's the Phase 4 polish version.

---

## How to Verify the Current Build

```bash
# Run all Go tests
cd /Users/jmp/Projects/hamster-wheel
go test ./...

# Run frontend tests
cd /Users/jmp/Projects/hamster-wheel/frontend
npm test

# Run the app in dev mode (requires Wails v3 CLI installed)
cd /Users/jmp/Projects/hamster-wheel
wails3 dev

# Check for lint issues
go vet ./...
```

**To test with real data:** Set `REED_API_KEY` environment variable with a Reed API key (get one at https://www.reed.co.uk/developers), then run `wails dev`. Create a filter in the UI and click "Poll Now".

---

*Last updated: 2026-02-09 — Phase 1 complete (Wails v3 migration + system tray implemented).*
