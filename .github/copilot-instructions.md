# hd-smart-idle Copilot Guide
- hd-smart-idle is a Go daemon that keeps mechanical HDDs awake unless a scheduled cron run reapplies the hdparm standby timer.
- Motivation: plain `hdparm -S <value>` can cycle disks between standby and active all day; this daemon sets the timer once at the configured schedule and, after any disk wakes, disables further standby until the next scheduled window.
## Architecture
- CLI currently exposes a `run` command that discovers rotational disks, keeps them awake when active, and reapplies a daily standby timer according to the cron expression.
- Extend capabilities by wiring new flags through `cmd/run/run.go` into `internal/daemon` and keeping disk interactions behind the `HDDControl` interface.
## Key Patterns
- Always inject behavior through the `HDDControl` interface so dry-run and tests can wrap or stub the hardware layer.
- `Daemon` methods lock `mu` only around the shared `last` map; avoid long blocking work while holding the mutex.
- Polling uses `time.NewTicker` plus `time.After(time.Until(nextScheduledTime))`; update both when modifying scheduling logic.
- `CronExpr.Parse` expects space-delimited hour/min strings (`"22 00"`); passing `"22:00"` disables scheduling and is how the CLI currently behaves.
- Enable dry-runs via `Daemon.Config.DryRun` which wraps the controller with `hw.NewDryRunHDDControl` and only logs `hdparm` commands.
## Build & Test Workflow
- Preferred commands live in `Makefile`: `make` builds `bin/hd-smart-idle`, `make test` runs `go test ./...`, `make lint` installs (via official script) and runs `golangci-lint` from `bin/`.
## Writing Unittest
- Use table-driven tests for functions with multiple scenarios
- Mock external dependencies using interfaces
- Use `github.com/stretchr/testify` for assertions and mocking
## External Integration
- The daemon shells out to `/sbin/hdparm`; configure an alternate path with the `HDPARM_PATH` environment variable when testing.
- Disk detection depends on `/sys/block/*/queue/rotational`; ensure CI or reproductions provide these files or mock via `fstest`.
## CLI & Logging
- CLI built with Cobra; add flags or subcommands by updating `cmd/run/run.go` and mapping inputs into `daemon.Config`.
- Logging uses `logrus`; keep text output on stdout, honor the configured log level, and log hardware actions at info/debug appropriately.
## Git Commit Guidelines
- Follow Conventional Commits format: `type(scope): description` (scope is optional)
- Types: feat, fix, docs, refactor, perf, test, build, chore, revert
- Title must be 50 characters or less
- For complex commits, add detailed explanation in the body
- Examples:
  - `feat: add heartbeat monitoring for pods`
  - `feat(controller): add heartbeat monitoring for pods`
  - `fix: resolve race condition in status updates`
