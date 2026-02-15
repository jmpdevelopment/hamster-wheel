# Product Definition

## Mission

Hamster Wheel helps users discover and act on relevant jobs faster by automating job-board polling, matching jobs to a user CV with AI, and assisting document tailoring.

## Problem

Manual job searching is repetitive and time-sensitive. Early applicants have a meaningful advantage, but repeated monitoring is costly.

## Solution

A self-hosted desktop app (macOS and Windows) that:

1. Polls job sources in the background.
2. Deduplicates and stores discovered jobs locally.
3. Scores fit against the user profile when LLM is configured.
4. Notifies on high-quality matches.
5. Supports CV and cover-letter tailoring workflows.

## Product Principles

- Self-hosted first: no required cloud backend, no user accounts.
- Privacy first: no telemetry; external calls limited to configured job APIs and LLM APIs.
- Extensible by design: adapter pattern for job sources and provider abstraction for LLMs.
- Practical quality bar: deterministic behavior, explicit errors, and test-backed changes.
- Open and portable: Apache 2.0, native desktop binaries.
- Easy setup first: non-technical users should configure matching successfully without networking concepts.
- Progressive disclosure: advanced provider controls (for example manual endpoint overrides) are hidden unless explicitly enabled.
- Local autonomy: users who prefer local models should be able to run them from guided in-app workflows.

## Scope by Phase

- Phase 1: Foundation and reliable polling core (complete).
- Phase 1.5: UX standards and interaction consistency (complete).
- Phase 2: LLM matching and threshold-driven notifications (in progress).
- Phase 3: Document tailoring and AI-detection self-check loop.
- Phase 4: Setup wizard and full dashboard/status workflows.
- Phase 5: Distribution hardening (CI and packaged releases).

## Out of Scope for Current Plan

- Linux packaging.
- Auto-apply workflows.
- Full analytics dashboards.
- Multi-profile CV orchestration.
- Salary intelligence features.

## Success Criteria

- Reliable background polling with clear diagnostics.
- Low-friction setup and key management.
- Actionable matching outputs with realistic scoring.
- Stable build and test workflows that support incremental delivery.
