# Hamster Wheel

Job search monitoring with LLM-powered matching

## Overview

Hamster Wheel is a self-hosted desktop application (macOS & Windows) built with Wails v3. It runs in the background, polls job boards at configurable intervals, deduplicates and stores results locally in SQLite, and scores each posting against your CV using an AI model (cloud or local LLM). When a high-quality match is found, you get a native desktop notification. CV and cover-letter tailoring workflows are planned for a future phase.

## Features

- Background job-board polling with configurable intervals
- LLM-powered job matching (OpenAI cloud, local models, or OpenAI-compatible endpoints)
- Native desktop notifications for high-score matches
- Local SQLite storage — no cloud backend required
- OS keychain integration for API key storage
- Extensible adapter pattern for job sources
- Privacy-first: no telemetry, no user accounts

## Tech Stack

| Layer | Technology |
|---|---|
| Desktop Framework | Wails v3 |
| Backend | Go 1.25 |
| Database | SQLite (modernc.org/sqlite) |
| Frontend | React 18 + TypeScript + Vite + Tailwind |
| Secrets | zalando/go-keyring |
| LLM Integration | Provider interface (OpenAI-first, extensible) |

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

## Project Structure

```
├── main.go                  # Application entry point
├── app.go                   # App lifecycle service
├── *_service.go             # Wails-bound service layer
├── frontend/                # React + TypeScript UI
├── internal/
│   ├── db/                  # Migrations and DB operations
│   ├── adapter/             # Job source adapters
│   ├── scheduler/           # Polling orchestration
│   ├── matcher/             # LLM matching engine
│   └── keychain/            # Key storage abstraction
├── docs/                    # Project documentation & AI context
└── scripts/                 # Build and installer scripts
```

## Building Installers

Windows (NSIS) and macOS (pkg/dmg) installer instructions are in [`scripts/installers/README.md`](scripts/installers/README.md).

## Documentation

Detailed architecture, product, and engineering docs live in [`docs/`](docs/). For AI agent context (documentation contract, read order, and minimal context packs) see [`docs/AI_CONTEXT.md`](docs/AI_CONTEXT.md).

## Current Status

Phase 1 (foundation + polling) is complete. Phase 2 (LLM matching) is in progress. See [`docs/execution/status.md`](docs/execution/status.md) and [`docs/execution/roadmap.md`](docs/execution/roadmap.md) for details.

## License

Apache 2.0
