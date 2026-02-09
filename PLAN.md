# Hamster Wheel - Project Plan

> A self-hosted desktop application that monitors job boards for new postings, matches them against the user's CV using AI, and helps tailor application documents.

**Status:** Phase 1 — ✅ Complete! (see [PROGRESS.md](PROGRESS.md) for detailed status)
**License:** Apache 2.0
**Repository:** GitHub (TBD)

---

## Table of Contents

1. [Product Overview](#1-product-overview)
2. [Architecture & Tech Stack](#2-architecture--tech-stack)
3. [Data Model](#3-data-model)
4. [Feature Specifications](#4-feature-specifications)
5. [Job Source Adapter System](#5-job-source-adapter-system)
6. [LLM Integration](#6-llm-integration)
7. [Security](#7-security)
8. [UI/UX Design](#8-uiux-design)
9. [Distribution & Installation](#9-distribution--installation)
10. [Project Phases](#10-project-phases)
11. [Out of Scope (Future)](#11-out-of-scope-future)
12. [Open Questions / Risks](#12-open-questions--risks)

---

## 1. Product Overview

### 1.1 Problem

Applying to jobs early dramatically increases your chances, but manually monitoring job boards is tedious and time-consuming. By the time most people see a listing, dozens of applicants have already applied.

### 1.2 Solution

Hamster Wheel runs on the user's own machine (Windows/macOS), silently polls job boards in the background, uses AI to match new postings against the user's CV, and notifies them instantly when a strong match is found. It can then tailor the user's CV and cover letter for that specific role.

### 1.3 Core Principles

- **Self-hosted**: All data stays on the user's machine. No servers, no accounts, no data laws.
- **Free & open-source**: Zero cost to use. Users only pay for their own LLM API usage.
- **Privacy-first**: No telemetry, no analytics, no data leaves the machine except API calls the user explicitly configures.
- **Extensible**: Plugin architecture for adding new job sources and features.
- **Accessible**: Easy to install and use for non-technical people.

### 1.4 Target Platforms

- macOS (Apple Silicon + Intel)
- Windows (x64)
- (Linux: deferred to a future release)

---

## 2. Architecture & Tech Stack

### 2.1 Framework: Wails v3

- **Backend**: Go
- **Frontend**: React + TypeScript + Vite
- **Output**: Single native binary per platform
- **Why Wails**: Lightweight (~10-15 MB), uses system webview (no bundled Chromium), Go backend is ideal for HTTP polling, concurrency (goroutines), and API calls. Single binary makes distribution trivial. **v3 provides native system tray support** across all platforms.

### 2.2 High-Level Architecture

```
+------------------------------------------------------------------+
|                        Hamster Wheel App                         |
|                                                                  |
|  +---------------------------+  +-----------------------------+  |
|  |     React Frontend        |  |       Go Backend            |  |
|  |                           |  |                             |  |
|  |  - Split view UI          |  |  - Job Source Manager       |  |
|  |  - Job list + details     |  |  - Scheduler (30 min poll)  |  |
|  |  - Settings/config panel  |  |  - LLM Provider Manager     |  |
|  |  - Status tracking board  |  |  - CV Parser (PDF)          |  |
|  |  - Setup wizard           |  |  - Document Generator       |  |
|  |                           |  |  - Notification Manager     |  |
|  |  Communicates via Wails   |  |  - Keychain Manager         |  |
|  |  bindings (RPC bridge)    |  |  - SQLite Database          |  |
|  +---------------------------+  +-----------------------------+  |
|                                                                  |
|  +------------------------------------------------------------+  |
|  |                    System Tray Service                      |  |
|  |  - Background polling loop                                  |  |
|  |  - OS-native notifications                                  |  |
|  |  - Tray icon + context menu (Open / Pause / Quit)           |  |
|  +------------------------------------------------------------+  |
+------------------------------------------------------------------+
         |                    |                    |
         v                    v                    v
   [Reed API]         [Claude API]         [Local Filesystem]
   (Job sources)      (LLM matching)       (CV, data, config)
```

### 2.3 Backend Packages (Go)

| Concern | Approach |
|---|---|
| HTTP requests | `net/http` (stdlib) |
| Database | `SQLite` via `modernc.org/sqlite` (pure Go, no CGO) |
| PDF reading (CV input) | `unipdf` or `pdfcpu` |
| PDF generation (tailored CV) | `go-pdf` or `maroto` |
| LLM providers | Direct HTTP calls; provider abstraction layer (Claude first, extensible) |
| Keychain | `zalando/go-keyring` (macOS Keychain + Windows Credential Manager) |
| Notifications | Wails native notifications / `gen2brain/beeep` |
| Scheduling | `robfig/cron` or custom goroutine ticker |
| Logging | `slog` (stdlib, structured logging) |

### 2.4 Frontend Packages (React + TypeScript)

| Concern | Approach |
|---|---|
| UI framework | React 18+ with TypeScript |
| Build tool | Vite (bundled with Wails) |
| Styling | Tailwind CSS |
| State management | Zustand (lightweight) or React Context |
| Routing | React Router (settings, main view, setup wizard) |
| Component library | shadcn/ui (accessible, customizable) |
| Drag/drop (status board) | `dnd-kit` |
| PDF preview | `react-pdf` |

### 2.5 Directory Structure

```
hamster-wheel/
├── build/                    # Wails build config, icons, installer config
├── frontend/                 # React app
│   ├── src/
│   │   ├── components/       # UI components (Header, JobList, JobDetail, FilterPanel, etc.)
│   │   ├── hooks/            # Custom React hooks (useJobs, useFilters)
│   │   ├── lib/              # Utilities (format, sanitize)
│   │   └── App.tsx           # Root component — three-pane layout
│   ├── wailsjs/              # Auto-generated Wails bindings (do not edit)
│   ├── index.html
│   └── package.json
├── internal/                 # Go internal packages (not importable externally)
│   ├── adapter/              # Job source adapter interface + implementations
│   │   ├── adapter.go        # Adapter interface definition
│   │   ├── registry.go       # Thread-safe adapter registry
│   │   └── reed/             # Reed UK adapter (implemented)
│   │       ├── reed.go
│   │       └── reed_test.go
│   ├── db/                   # Database layer
│   │   ├── sqlite.go         # SQLite connection, WAL mode, foreign keys
│   │   ├── migrations.go     # Schema versioning (PRAGMA user_version)
│   │   ├── jobs.go           # Job CRUD operations
│   │   ├── filters.go        # Search filter CRUD
│   │   ├── settings.go       # Key-value settings storage
│   │   ├── errors.go         # Sentinel errors (ErrNotFound, ErrDuplicateJob, etc.)
│   │   └── *_test.go         # Tests for each module
│   └── scheduler/            # Background polling scheduler
│       ├── scheduler.go      # Concurrent polling, single-writer DB inserts
│       └── scheduler_test.go
│   # --- NOT YET BUILT ---
│   # ├── llm/                # LLM provider abstraction (Phase 2)
│   # ├── cv/                 # CV parsing and generation (Phase 2-3)
│   # ├── keychain/           # OS keychain for API keys (Phase 2)
│   # └── notifier/           # OS notifications (Phase 2)
├── app.go                    # Main Wails app struct + all frontend-exposed methods
├── main.go                   # Entry point, dependency injection, Wails config
├── wails.json                # Wails project config
├── go.mod
├── go.sum
├── PLAN.md                   # What to build (this file)
├── BUILD_INSTRUCTIONS.md     # How to build (process rules)
├── PROGRESS.md               # What's been built (status tracker)
├── LICENSE                   # Apache 2.0
└── README.md
```

---

## 3. Data Model

All data stored in a local SQLite database at the OS-appropriate config directory:
- macOS: `~/Library/Application Support/HamsterWheel/hamsterwheel.db`
- Windows: `%APPDATA%\HamsterWheel\hamsterwheel.db`

**Why SQLite:** Data is inherently relational (filters → jobs → matches with status history). SQLite is the industry standard for local desktop app storage — single file, zero external dependencies, compiled into the binary via `modernc.org/sqlite` (pure Go). Supports JSON column types for any fields needing flexible/semi-structured data.

### 3.1 Tables

#### `search_filters`
| Column | Type | Description |
|---|---|---|
| id | TEXT (UUID) | Primary key |
| name | TEXT | User-given name ("Backend London", "Remote React") |
| keywords | TEXT | Search keywords |
| location | TEXT | Location string |
| source | TEXT | Job source identifier (e.g., "reed_uk") |
| enabled | BOOLEAN | Whether this filter is actively polled |
| created_at | DATETIME | |
| updated_at | DATETIME | |

#### `jobs`
| Column | Type | Description |
|---|---|---|
| id | TEXT (UUID) | Primary key |
| source | TEXT | Which adapter found this ("reed_uk") |
| source_id | TEXT | External ID / hash for deduplication |
| title | TEXT | Job title |
| company | TEXT | Company name |
| location | TEXT | Job location |
| description | TEXT | Full job description |
| url | TEXT | Link to original posting |
| posted_at | DATETIME | When the job was posted (from source) |
| discovered_at | DATETIME | When Hamster Wheel found it |
| filter_id | TEXT (FK) | Which search filter matched this job |

#### `job_matches`
| Column | Type | Description |
|---|---|---|
| id | TEXT (UUID) | Primary key |
| job_id | TEXT (FK) | Reference to jobs table |
| match_score | REAL | 0.0 - 1.0 score from LLM |
| match_summary | TEXT | LLM-generated explanation of match |
| status | TEXT | One of: `matched`, `accepted`, `rejected`, `applied`, `interview`, `offer`, `closed` |
| tailored_cv_path | TEXT | Nullable, path to generated CV PDF |
| tailored_cl_path | TEXT | Nullable, path to generated cover letter PDF |
| status_updated_at | DATETIME | When status last changed |
| created_at | DATETIME | |

#### `llm_prompts`
| Column | Type | Description |
|---|---|---|
| id | TEXT | Primary key. One of: `matching`, `cv_tailoring`, `cover_letter_tailoring`, `ai_detection` |
| system_prompt | TEXT | The system prompt text |
| is_custom | BOOLEAN | Whether user has customized this (false = using default) |
| updated_at | DATETIME | |

Default prompts are shipped with the app. If `is_custom` is false, the app uses the built-in default (so defaults can be improved via app updates). Once the user edits a prompt, `is_custom` becomes true and their version is preserved across updates.

#### `approved_domains`
| Column | Type | Description |
|---|---|---|
| domain | TEXT | Primary key. e.g., "careers.google.com" |
| approved_at | DATETIME | When the user approved this domain |

Used when a job posting redirects to an external company site. The user is asked once per domain; approval persists.

#### `settings`
| Column | Type | Description |
|---|---|---|
| key | TEXT | Primary key |
| value | TEXT | JSON-encoded value |

Settings keys include:
- `cv_path` — Path to user's uploaded CV PDF
- `cover_letter_draft` — User's draft cover letter text/template
- `custom_instructions` — User's custom instructions injected into tailoring prompts (e.g., "Emphasize my leadership experience")
- `match_threshold` — Minimum score (0.0-1.0) to trigger notification (default: 0.7)
- `poll_interval_minutes` — Polling interval (default: 30)
- `first_run_complete` — Whether setup wizard has been completed
- `llm_provider` — Active LLM provider identifier (default: "claude")
- `llm_api_key_name` — Keychain entry name for the active provider's API key

### 3.2 File Storage

Tailored CVs and cover letters are stored as PDF files:
- macOS: `~/Library/Application Support/HamsterWheel/documents/`
- Windows: `%APPDATA%\HamsterWheel\documents\`

File naming: `{job_id}_cv.pdf`, `{job_id}_cover_letter.pdf`

---

## 4. Feature Specifications

### 4.1 First-Launch Setup Wizard

A step-by-step guided wizard that appears on first run:

**Step 1 — Welcome**
- Brief explanation of what Hamster Wheel does
- "Get started" button

**Step 2 — LLM Provider Setup**
- Explanation: "Hamster Wheel uses AI to match jobs and tailor your documents"
- Provider selector dropdown (Claude is default and recommended; extensible for future providers)
- Per-provider instructions with link to get an API key (e.g., console.anthropic.com for Claude)
- Paste field for API key
- "Test Connection" button to validate the key against the selected provider
- "Skip for now" option (app works without it, but no matching/tailoring)
- Key is stored in OS keychain immediately upon entry, keyed by provider name

**Step 3 — Upload CV**
- File picker for PDF
- Preview of parsed content (so user can verify it parsed correctly)
- Explanation of what the CV is used for

**Step 4 — Cover Letter Draft**
- Text area for user to enter their base cover letter
- Explanation: "Provide a rough template — mention your key strengths and what you want to highlight. The AI will tailor this for each specific job."
- Can be skipped

**Step 5 — First Search Filter**
- Keywords field
- Location field
- Source dropdown (only "Reed UK" initially)
- Name for this filter
- "Add more filters" option

**Step 6 — Done**
- Summary of configuration
- "Start monitoring" button
- App begins polling immediately

### 4.2 Background Polling & Job Discovery

**Scheduler:**
- Runs on a configurable interval (default: 30 minutes)
- Each enabled search filter is polled independently via its adapter
- Polling happens in separate goroutines (concurrent per filter)
- On first poll, fetches currently visible jobs and marks them as the baseline
- Subsequent polls only process jobs not yet in the database (deduplication via `source_id`)

**Flow per poll cycle:**
1. For each enabled filter, call the adapter's `FetchNewJobs()` method
2. Adapter returns list of new job summaries (title, company, location, URL, snippet)
3. For each new job, fetch the full description (adapter's `FetchJobDetails()`)
4. Store job in `jobs` table
5. If an LLM provider is configured with a valid API key:
   a. Send job description + user's CV to LLM matcher (via active provider)
   b. Receive match score (0.0-1.0) and explanation
   c. Store in `job_matches` table
   d. If score >= threshold: trigger OS notification
6. If no LLM provider configured: store job but skip matching, no notification

**Deduplication:**
- Each job gets a `source_id` derived from its unique URL or ID from the source
- Before processing, check if `source_id` already exists in the database
- Skip if it does

### 4.3 Job Matching (LLM)

See [Section 6](#6-llm-integration) for detailed LLM prompt design.

**Inputs:**
- Full job description text
- User's parsed CV text
- User's cover letter draft (if available)

**Outputs:**
- `match_score`: Float 0.0 - 1.0
- `match_summary`: 2-3 sentence explanation of why this job is/isn't a good match
- `key_matches`: List of specific skills/experiences that align
- `gaps`: List of requirements the user may not meet

### 4.4 Notifications

When a job exceeds the match threshold:
- **OS-native notification** appears:
  - Title: "Hamster Wheel - New Match!"
  - Body: "{Job Title} at {Company} — {match_score}% match"
  - Click action: Opens the app window and navigates to that job

- **Tray icon badge** (if supported): Shows count of unreviewed matches

### 4.5 Main Dashboard (Split View)

**Left Panel — Job List:**
- Tabs or filter bar: "New Matches" | "Accepted" | "All" | by status
- Each item shows:
  - Job title
  - Company name
  - Location
  - Match score (as percentage badge, color-coded: green >80%, amber 60-80%, red <60%)
  - Time since discovered ("2h ago")
  - Current status pill
- Sorted by: match score (default), date discovered, or status
- Search/filter within the list

**Right Panel — Job Detail:**
- Full job title + company + location
- Match score + LLM match summary
- "Key matches" and "Gaps" sections
- Full job description (scrollable)
- Action buttons:
  - "Accept" (green) — moves to `accepted` status
  - "Reject" (red) — moves to `rejected` status
  - "Open in Browser" — opens original job URL
  - "Generate Documents" — triggers CV + cover letter tailoring (only shown if LLM provider configured with valid API key)
- Generated documents section (after tailoring):
  - Preview of tailored CV
  - Preview of tailored cover letter
  - "Download CV" / "Download Cover Letter" buttons
  - "Regenerate" button if user wants a different version

### 4.6 Job Status Tracking

Users can move jobs between statuses manually. Statuses form a pipeline:

```
matched → accepted → applied → interview → offer
                  ↘ rejected                ↘ closed
                                → closed
```

- Status is changed via dropdown or drag-and-drop on a Kanban-style board view (optional secondary view)
- Status history is preserved (timestamp of each transition stored)
- Status data is available for future features (analytics, patterns, etc.)

### 4.7 Settings Page

- **Search Filters**: List of saved filters. Add / edit / delete / enable / disable.
- **CV Management**: Upload new CV, preview parsed content, re-upload.
- **Cover Letter Draft**: Edit the base template.
- **LLM Provider**: Select active provider. Per-provider API key management — update, remove, show masked version, "Test" button.
- **Custom Instructions**: Text field for plain-language preferences injected into all prompts.
- **Match Threshold**: Slider from 0% to 100% (default 70%).
- **Poll Interval**: Dropdown (15 min, 30 min, 1 hour, 2 hours).
- **Notifications**: Enable/disable.
- **Approved Domains**: List of external domains approved for job description scraping. Remove individual approvals.
- **Advanced → Prompt Customization**: Edit system prompts for matching, CV tailoring, cover letter tailoring, and AI detection. "Reset to Default" per prompt.
- **Data**: Export all data as JSON. Clear all jobs. Reset application.

### 4.8 System Tray

- **Icon**: Hamster wheel icon (idle state)
- **Left-click**: Open/focus the main window
- **Right-click context menu**:
  - "Open Hamster Wheel"
  - "Pause Monitoring" / "Resume Monitoring"
  - Separator
  - "Quit"
- **Tooltip**: "Hamster Wheel — Monitoring (next poll in X min)" or "Hamster Wheel — Paused"

---

## 5. Job Source Adapter System

### 5.1 Adapter Interface (Go)

```go
// internal/adapter/adapter.go

type JobSummary struct {
    SourceID    string
    Title       string
    Company     string
    Location    string
    URL         string
    Snippet     string    // Short description / preview
    PostedAt    time.Time
}

type JobDetails struct {
    JobSummary
    FullDescription string
    Salary          string // If available
    JobType         string // Full-time, Part-time, Contract, etc.
}

type SearchParams struct {
    Keywords string
    Location string
}

// Adapter is the interface that all job source integrations must implement.
type Adapter interface {
    // Name returns the unique identifier for this adapter (e.g., "reed_uk")
    Name() string

    // DisplayName returns the human-readable name (e.g., "Reed UK")
    DisplayName() string

    // FetchNewJobs retrieves job listings matching the given search parameters.
    // The adapter is responsible for making HTTP requests and parsing responses.
    // Returns only jobs, deduplication is handled by the caller.
    FetchNewJobs(ctx context.Context, params SearchParams) ([]JobSummary, error)

    // FetchJobDetails retrieves the full details for a specific job.
    // This is called after FetchNewJobs to get the complete description.
    FetchJobDetails(ctx context.Context, job JobSummary) (*JobDetails, error)

    // Validate checks if the adapter is functional (e.g., can reach the site).
    Validate(ctx context.Context) error
}
```

### 5.2 Adapter Registry

```go
// internal/adapter/registry.go

type Registry struct {
    adapters map[string]Adapter
}

func NewRegistry() *Registry { ... }
func (r *Registry) Register(adapter Adapter) { ... }
func (r *Registry) Get(name string) (Adapter, bool) { ... }
func (r *Registry) List() []Adapter { ... }
```

New job sources are added by:
1. Creating a new package under `internal/adapter/` (e.g., `internal/adapter/glassdoor/`)
2. Implementing the `Adapter` interface
3. Registering it in the app startup

### 5.3 Reed UK Adapter (Implemented)

**Approach: Reed REST API**

Indeed decommissioned their RSS feeds, so the first adapter uses Reed.co.uk's official REST API instead. Reed provides a JSON API with HTTP Basic Auth (API key as username, empty password).

- **API Base URL:** `https://www.reed.co.uk/api/1.0`
- **Auth:** HTTP Basic (API key as username, empty password)
- **API Key:** Users get a free key at https://www.reed.co.uk/developers
- **Adapter Name:** `reed_uk`
- **Implementation:** `internal/adapter/reed/reed.go`

**`FetchNewJobs` implementation:**
1. Construct search URL: `/search?keywords={keywords}&location={location}`
2. HTTP GET with Basic Auth header
3. Parse JSON response into `JobSummary` structs
4. `SourceID` = deterministic hash of job ID from Reed

**`FetchJobDetails` implementation:**
1. HTTP GET `/jobs/{jobId}` with Basic Auth
2. Parse JSON response for full description, salary, job type
3. Uses `externalUrl` from Reed if available (links to company site)
4. Return `JobDetails`

**Rate limiting:**
- Minimum 2-second delay between HTTP requests
- Configurable via adapter internals

---

## 6. LLM Integration

### 6.1 LLM Provider Abstraction

Similar to the job source adapter system, LLM providers are abstracted behind an interface so new providers can be added without changing core logic.

```go
// internal/llm/provider.go

type MatchResult struct {
    Score       float64  // 0.0 - 1.0
    Summary     string   // 2-3 sentence assessment
    KeyMatches  []string // Skills/experience that align
    Gaps        []string // Requirements candidate may not meet
}

type TailorResult struct {
    Content         string  // The tailored text
    AIDetectionScore float64 // 0.0 - 1.0 from self-check
    Warnings        []string // Any flags from AI detection
}

type DetectionResult struct {
    AIProbability float64  // 0.0 - 1.0
    Flags         []string // Specific AI-sounding phrases
    Suggestions   []string // Rewrites to sound more human
}

// LLMProvider is the interface all LLM integrations must implement.
type LLMProvider interface {
    // Name returns the unique identifier (e.g., "claude", "openai")
    Name() string

    // DisplayName returns the human-readable name (e.g., "Claude (Anthropic)")
    DisplayName() string

    // APIKeyURL returns the URL where users can get an API key
    APIKeyURL() string

    // SendMessage sends a system + user prompt and returns the raw text response.
    // All prompt construction is handled by the caller (matcher.go / tailor.go).
    // This keeps the provider implementation simple — just handle the API transport.
    SendMessage(ctx context.Context, systemPrompt string, userPrompt string, maxTokens int) (string, error)

    // Validate checks if the API key is valid and the provider is reachable.
    Validate(ctx context.Context) error
}
```

**Provider Registry** (same pattern as adapter registry):
```go
// internal/llm/registry.go

type ProviderRegistry struct {
    providers map[string]LLMProvider
}

func NewProviderRegistry() *ProviderRegistry { ... }
func (r *ProviderRegistry) Register(provider LLMProvider) { ... }
func (r *ProviderRegistry) Get(name string) (LLMProvider, bool) { ... }
func (r *ProviderRegistry) List() []LLMProvider { ... }
```

New LLM providers are added by:
1. Creating a new package under `internal/llm/` (e.g., `internal/llm/openai/`)
2. Implementing the `LLMProvider` interface (just the `SendMessage` transport)
3. Registering it in the app startup

The matching and tailoring logic (`matcher.go`, `tailor.go`) is **provider-agnostic** — they construct prompts and parse responses, calling `provider.SendMessage()` for the actual API call.

### 6.2 Claude Provider (First Implementation)

Direct HTTP integration with `https://api.anthropic.com/v1/messages`.

**Configuration:**
- Model: `claude-sonnet-4-5-20250929` (good balance of cost/quality)
- Max tokens: Passed through from caller (1024 for matching, 4096 for tailoring)
- API key: Retrieved from OS keychain at runtime via key name `hamsterwheel_claude`

**Error handling (common across all providers):**
- Rate limit (429): Exponential backoff with jitter
- API errors: Log and mark job as "matching failed", retry on next cycle
- Network errors: Retry with backoff, pause polling if persistent
- Invalid key: Notify user, pause LLM features

### 6.3 Default Prompts

The app ships with default prompts for each operation. These are the starting point; users can customize them in Settings (see 6.4).

**Template variables** available in all prompts:
- `{cv_text}` — User's parsed CV text
- `{job_title}` — Job title
- `{company}` — Company name
- `{location}` — Job location
- `{job_description}` — Full job description text
- `{cover_letter_draft}` — User's draft cover letter
- `{custom_instructions}` — User's custom instructions (see 6.4)

#### Default: Job Matching Prompt

```
System: You are a job matching assistant. You will be given a candidate's CV and a
job posting. Analyze how well the candidate matches the job requirements.

You MUST respond in valid JSON with this exact structure:
{
  "match_score": <float 0.0 to 1.0>,
  "match_summary": "<2-3 sentence overall assessment>",
  "key_matches": ["<skill/experience that aligns>", ...],
  "gaps": ["<requirement candidate may not meet>", ...]
}

Scoring guide:
- 0.9-1.0: Near-perfect match, meets almost all requirements
- 0.7-0.89: Strong match, meets most key requirements
- 0.5-0.69: Partial match, meets some requirements but has notable gaps
- 0.3-0.49: Weak match, significant gaps in key areas
- 0.0-0.29: Poor match, candidate's profile doesn't align

Be realistic and objective. Consider transferable skills.
Do not inflate scores.

{custom_instructions}

User:
## Candidate CV:
{cv_text}

## Job Posting:
Title: {job_title}
Company: {company}
Location: {location}
Description:
{job_description}
```

#### Default: CV Tailoring Prompt

```
System: You are an expert CV writer. Given a candidate's original CV and a specific
job posting, rewrite the CV to be tailored for this role.

Rules:
- Keep all factual information accurate — do not invent experience or skills
- Reorder and emphasize relevant experience and skills for this specific role
- Use natural, professional language — vary sentence structure and vocabulary
- Mirror key terminology from the job description where the candidate genuinely has that skill
- Do NOT use generic AI-sounding phrases like "leveraged", "spearheaded", "synergized"
- Do NOT include a summary/objective section unless the original CV had one
- Maintain the overall structure and length of the original CV
- Write as a human professional would — slightly imperfect, personal, specific
- Output the full tailored CV text (this will be converted to PDF)

{custom_instructions}

User:
## Original CV:
{cv_text}

## Target Job:
Title: {job_title}
Company: {company}
Description:
{job_description}
```

#### Default: Cover Letter Tailoring Prompt

```
System: You are helping a job candidate write a cover letter. You are given their
draft cover letter template, their CV, and the job they are applying for.

Rules:
- Use the draft as a structural and tonal guide — keep the candidate's voice
- Incorporate specific details from the job description
- Reference concrete experience from the CV that relates to this role
- Keep it concise — no more than one page worth of text
- Sound human and genuine — not generic or AI-generated
- Vary sentence length and structure naturally
- Do NOT use cliches like "I am writing to express my interest" unless the draft uses them
- Do NOT use words like "synergy", "leverage", "spearhead", "delve"
- Output the complete cover letter text

{custom_instructions}

User:
## Draft Cover Letter Template:
{cover_letter_draft}

## Candidate CV:
{cv_text}

## Target Job:
Title: {job_title}
Company: {company}
Description:
{job_description}
```

#### Default: AI Detection Self-Check Prompt

```
System: You are an AI content detector. Analyze the following text and assess
whether it appears to be written by AI or by a human.

Respond in JSON:
{
  "ai_probability": <float 0.0 to 1.0>,
  "flags": ["<specific phrases or patterns that seem AI-generated>"],
  "suggestions": ["<specific rewrites to sound more human>"]
}

Be strict. If the probability is above 0.5, the text needs revision.

User:
{generated_text}
```

If `ai_probability > 0.5`, automatically apply the suggestions and re-check (max 2 iterations). If still high, present to user with a warning.

### 6.4 User-Customizable Prompts & Instructions

Two levels of customization:

**Level 1 — Custom Instructions (Simple)**
A single text field in Settings where users describe their preferences in plain language. This text is injected into all prompts via the `{custom_instructions}` variable.

Examples:
- "Emphasize my leadership and people management experience"
- "I'm transitioning from finance to tech — highlight transferable skills"
- "I prefer a formal, conservative tone in cover letters"
- "Don't mention my first job at McDonald's — it's not relevant"

This is the recommended approach for most users. No prompt engineering knowledge needed.

**Level 2 — Full Prompt Editing (Advanced)**
Under Settings → Advanced → Prompt Customization:
- Separate text area for each of the 4 prompts (matching, CV tailoring, cover letter tailoring, AI detection)
- Shows the current prompt (default or user-modified)
- "Reset to Default" button per prompt to revert to the shipped version
- Template variable reference shown alongside the editor
- Warning: "Modifying prompts is advanced. The default prompts are optimized for best results."

When a user edits a prompt, `is_custom` is set to true in `llm_prompts` table. On app updates, only non-custom prompts receive updated defaults.

---

## 7. Security

### 7.1 API Key Storage

| Platform | Storage | Access |
|---|---|---|
| macOS | Keychain Services | App-specific, requires user session |
| Windows | Credential Manager | App-specific, requires user session |

- Key is NEVER written to disk in plaintext
- Key is NEVER logged
- Key is retrieved from keychain only when needed for API calls, not cached in memory long-term

### 7.2 Local Data Security

- SQLite database is stored in the OS user-specific application data directory
- File permissions set to user-only read/write (0600 on macOS)
- No encryption at rest for v1 (the OS user session is the security boundary)
- Generated documents stored with user-only permissions

### 7.3 Network Security

- All HTTP requests use TLS (HTTPS only)
- No HTTP fallback
- Claude API key sent only in `x-api-key` header over HTTPS
- No other external network calls except job source scraping and Claude API
- No telemetry, analytics, or phoning home

### 7.4 Application Security

- No remote code execution capabilities
- No eval or dynamic code loading
- Wails security model: frontend cannot access Go functions not explicitly bound
- Input sanitization on all user inputs before database storage
- Parameterized SQL queries only (no string concatenation)
- Content Security Policy headers in the webview

### 7.5 Open Source Security

- No secrets in the repository
- `.gitignore` excludes all local data, configs, and build artifacts
- Dependency audit on each release (Go and npm)
- Clear documentation of what data flows where

---

## 8. UI/UX Design

### 8.1 Design Principles

- Clean, minimal, professional
- Light and dark mode support
- Responsive within reasonable window sizes
- Accessible (keyboard navigation, screen reader labels)

### 8.2 Views

1. **Setup Wizard** — Full-screen stepper (only on first run or reset)
2. **Dashboard** — Split view (primary view, see section 4.5)
3. **Status Board** — Kanban-style view of jobs by status (optional secondary view)
4. **Settings** — Full-page settings panel

### 8.3 Navigation

- Sidebar or top-bar navigation: Dashboard | Board | Settings
- System tray for background access

### 8.4 Color Palette (Indicative)

- Primary: A warm, friendly tone (amber/orange — hamster wheel theme)
- Match score badges: Green (>80%), Amber (60-80%), Red (<60%)
- Status pills: Distinct color per status
- Dark mode: Full dark theme variant

---

## 9. Distribution & Installation

### 9.1 Build Outputs

| Platform | Format | Tool |
|---|---|---|
| macOS | `.dmg` with `.app` bundle | Wails build + `create-dmg` |
| Windows | `.msi` installer | Wails build + WiX toolset or NSIS |

### 9.2 GitHub Releases

- Each version tagged as a GitHub release
- Pre-built binaries for macOS (universal) and Windows (x64) attached
- Changelog in release notes
- SHA256 checksums for all artifacts

### 9.3 Signing (Future, but noted)

- macOS: Code signing + notarization (requires Apple Developer account, $99/yr) — without this, users must right-click → Open on first launch. Document this clearly.
- Windows: Code signing (requires certificate) — without this, SmartScreen warning appears. Document workaround.

### 9.4 First Run Experience

1. User downloads installer from GitHub releases
2. Runs installer (standard OS install flow)
3. App launches into the setup wizard
4. Wizard guides through API key, CV upload, and first filter
5. App begins monitoring, minimizes to system tray

---

## 10. Project Phases

### Phase 1: Foundation (MVP)

**Goal:** App runs, polls Reed UK via API, stores jobs, displays them in UI.

- [x] Initialize Wails project with Go + React + TypeScript
- [x] Set up project structure (directories, Go modules, npm packages)
- [x] Implement SQLite database layer with migrations
- [x] Implement adapter interface and Reed UK API adapter
- [x] Implement background scheduler (goroutine with ticker)
- [x] Build React frontend: job list, detail, filter panel, polling controls
- [x] Wire up Wails bindings (Go ↔ React)
- [x] Basic settings storage (Reed API key via DB + env var)
- [x] **Wails v2 → v3 migration** (for native system tray support)
- [x] **System tray with context menu** (Open / Pause-Resume / Quit, hide-to-tray)
- [ ] Initial git commit (all code is currently untracked)

**Deliverable:** ✅ App polls Reed UK, shows new jobs in a three-pane layout, runs in system tray. No matching, no tailoring yet.

### Phase 2: LLM Matching

**Goal:** Jobs are scored against user's CV, high matches trigger notifications.

- [ ] Implement keychain manager (store/retrieve API keys per provider)
- [ ] Implement LLMProvider interface and provider registry
- [ ] Implement Claude provider (first implementation)
- [ ] Implement CV PDF parser
- [ ] Implement job matching logic (provider-agnostic, uses LLMProvider.SendMessage)
- [ ] Integrate matching into poll cycle
- [ ] OS-native notifications for high matches
- [ ] Match score display in job list and detail view
- [ ] Match threshold configuration

**Deliverable:** App matches jobs against CV, notifies on strong matches.

### Phase 3: Document Tailoring

**Goal:** App generates tailored CVs and cover letters.

- [ ] Implement CV tailoring prompt and flow
- [ ] Implement cover letter tailoring prompt and flow
- [ ] Implement AI detection self-check loop
- [ ] PDF generation for tailored documents
- [ ] Document preview in the UI
- [ ] Download buttons for generated documents
- [ ] "Regenerate" functionality

**Deliverable:** User can generate tailored documents for accepted jobs.

### Phase 4: Full UI & Polish

**Goal:** Complete, polished user experience.

- [ ] Setup wizard (all steps)
- [ ] Split view dashboard with all features
- [ ] Status tracking pipeline (status transitions, history)
- [ ] Kanban board view
- [ ] Settings page (all settings, including LLM provider selection)
- [ ] Custom instructions field in settings
- [ ] Advanced prompt customization UI (edit/reset prompts)
- [ ] Approved external domains management
- [ ] Search/filter within job list
- [ ] Dark mode
- [ ] Keyboard shortcuts
- [ ] Error states and empty states
- [ ] Loading states and skeletons

**Deliverable:** Polished, feature-complete v1.0.

### Phase 5: Distribution & Documentation

**Goal:** Ready for public release.

- [ ] macOS `.dmg` build and test
- [ ] Windows `.msi` build and test
- [ ] GitHub repository setup (LICENSE, README, CONTRIBUTING.md)
- [ ] User documentation (setup guide, FAQ)
- [ ] GitHub Actions CI (build + test on push)
- [ ] First GitHub release with binaries

**Deliverable:** v1.0 public release on GitHub.

---

## 11. Out of Scope (Future)

These features are explicitly deferred but the architecture should not prevent them:

- **Linux support** — Add when demand exists. Keychain fallback (encrypted file) needed.
- **Auto-updates** — Check GitHub releases API, notify user, download in-app.
- **Additional job sources** — Indeed, Glassdoor, Totaljobs, LinkedIn, etc. via new adapters.
- **Additional LLM providers** — OpenAI (GPT-4o), Google (Gemini), local models via Ollama. Architecture supports this via the LLMProvider interface.
- **Auto-apply** — Automatically submit applications on behalf of user (significant legal and ethical considerations).
- **Analytics dashboard** — Success rates, application funnel metrics from status history.
- **Multiple CV profiles** — Different CVs for different types of roles.
- **Salary insights** — Aggregate salary data from matched jobs.
- **Job alerts via email** — Send email notifications (would require optional email config).
- **Browser extension companion** — Mark jobs as "applied" directly from Indeed.

---

## 12. Open Questions / Risks

### 12.1 Risks

| Risk | Severity | Mitigation |
|---|---|---|
| Reed API changes or is discontinued | Medium | Adapter pattern allows swapping sources. Monitor API changelog. |
| Reed API rate limits tighten | Low | Built-in 2s delay between requests. Stagger filters if many. |
| Claude API costs for users | Low | Sonnet is cheap (~$3/M input tokens). Typical user: <$1/month. Document expected costs. |
| macOS/Windows unsigned binary warnings | Medium | Document workaround. Consider signing in future. |
| PDF parsing inaccuracy | Medium | Some CVs have complex layouts. Show parsed preview so user can verify. Allow plain text CV paste as fallback. |

### 12.2 Open Questions

1. **CV PDF generation layout**: Tailored CVs should attempt to match the original CV's visual layout. This is complex but important for user trust and professionalism.
2. **Rate limiting on Reed**: What happens if a user has 20+ filters? Built-in 2s delay per request helps, but may need staggering across filters.
3. **Job description availability**: Reed provides full descriptions via API for most jobs. Some jobs link to external company sites via `externalUrl`. Per-domain approval for external fetching is supported (approved_domains table exists).
4. ~~**License choice**~~: Resolved — Apache 2.0.

---

*This document is the single source of truth for the Hamster Wheel project. All implementation decisions should reference this plan.*
