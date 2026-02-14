# Known Issues and Risks

Last reviewed: 2026-02-14

## Open Bugs

- No confirmed open functional defects are currently tracked in this file.

When a bug is found, record:

- Severity (`critical`, `high`, `medium`, `low`)
- Impacted area
- Reproduction notes
- Planned fix step
- Regression test reference after fix

## Active Watchlist

- Distribution signing/notarization is deferred; users may see trust warnings on unsigned builds.
- LLM matching and tailoring phases are not fully implemented yet; quality depends on upcoming Phase 2-3 completion.
- PDF parsing/generation quality may vary by document structure and will need targeted validation.

## Risk Register

| Risk | Current Mitigation |
| --- | --- |
| Job source API changes or deprecation | Adapter pattern allows source replacement with minimal impact. |
| LLM API cost growth | Keep matching targeted and thresholds configurable; default model choice is cost-aware. |
| Unsigned desktop package warnings | Document user workarounds now; revisit code-signing in distribution phase. |
| PDF parse inaccuracies | Add preview/review steps and fallback text workflows before final tailoring. |

## Reliability Notes

- Poll diagnostics are now exportable for failure triage.
- Poll-cycle errors now distinguish partial failures vs total run failures.
- Secrets are no longer exposed through frontend read paths.
