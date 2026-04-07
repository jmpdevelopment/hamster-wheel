# Hamster Wheel

Job search monitoring with LLM-powered matching

## Overview

Hamster Wheel is a self-hosted desktop application (macOS & Windows) built with Wails v3. It runs in the background, polls job boards at configurable intervals, deduplicates and stores results locally in SQLite, and scores each posting against your CV using an AI model (cloud or local LLM). When a high-quality match is found, you get a native desktop notification. CV and cover-letter tailoring workflows are planned for a future phase.

## Architecture

```mermaid
flowchart TD
    User("👤 User")

    subgraph Desktop["Desktop App (Wails v3)"]
        Frontend["Frontend\nReact + TypeScript"]
        subgraph Services["Go Services (Wails-bound)"]
            FilterSvc["FilterService"]
            JobSvc["JobService"]
            PollSvc["PollingService"]
            SettingsSvc["SettingsService"]
        end
    end

    subgraph Internal["Internal Packages"]
        Scheduler["Scheduler\nscheduler/"]
        Adapter["Job Adapters\nadapter/"]
        Matcher["Match Worker\nmatcher/"]
        Keychain["Keychain\nkeychain/"]
        DB[("SQLite\ndb/")]
    end

    LLM["LLM Provider\n(OpenAI / Local / Compatible)"]
    JobAPI["External Job APIs\n(e.g. Reed UK)"]
    Notify["Native OS Notification"]

    User -->|UI interactions| Frontend
    Frontend <-->|Wails bindings| Services
    PollSvc --> Scheduler
    SettingsSvc --> Keychain
    Scheduler -->|trigger poll| Adapter
    Adapter -->|fetch listings| JobAPI
    Adapter -->|persist & deduplicate| DB
    Scheduler -->|queue new jobs| Matcher
    Matcher -->|score against CV| LLM
    Matcher -->|store match result| DB
    Matcher -->|high-score match| Notify
    DB -->|retrieve jobs & scores| JobSvc
    DB -->|retrieve filters| FilterSvc
```

## Features

- Background job-board polling with configurable intervals
- LLM-powered job matching (OpenAI cloud, local models, or OpenAI-compatible endpoints)
- Native desktop notifications for high-score matches
- Local SQLite storage — no cloud backend required
- OS keychain integration for API key storage
- Extensible adapter pattern for job sources
- Privacy-first: no telemetry, no user accounts

## Prerequisites

- Go 1.25+
- Node.js (for frontend)
- Wails v3 CLI

## Getting Started

```bash
# Clone the repository
git clone https://github.com/jmpdevelopment/hamster-wheel.git
cd hamster-wheel

# Install frontend dependencies
cd frontend && npm install && cd ..

# Run in development mode
./dev-run.sh
```

## Building Installers

Windows (NSIS) and macOS (pkg/dmg) installer instructions are in [`scripts/installers/README.md`](scripts/installers/README.md).

## Documentation

Detailed architecture, product, and engineering docs live in [`docs/`](docs/). For AI agent context (documentation contract, read order, and minimal context packs) see [`docs/AI_CONTEXT.md`](docs/AI_CONTEXT.md).

## Current Status

Phase 1 (foundation + polling) is complete. Phase 2 (LLM matching) is in progress. See [`docs/execution/status.md`](docs/execution/status.md) and [`docs/execution/roadmap.md`](docs/execution/roadmap.md) for details.

## License

Apache 2.0
