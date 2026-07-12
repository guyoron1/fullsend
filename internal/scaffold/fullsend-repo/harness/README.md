# Do not edit directly

The canonical editing source for harness configs is
[`fullsend-ai/agents`](https://github.com/fullsend-ai/agents).
Make changes there — these files should be kept in sync with the
upstream source.

These configs are actively embedded into the Go binary via
`//go:embed all:fullsend-repo` and must remain in this repository.
The CLI uses them to compute integrity hashes for base URLs served
via `raw.githubusercontent.com`.
