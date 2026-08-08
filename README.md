# guardflow-go

[![CI](https://github.com/thaicn1712/guardflow-go/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/thaicn1712/guardflow-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/thaicn1712/guardflow-go.svg)](https://pkg.go.dev/github.com/thaicn1712/guardflow-go)
[![license](https://img.shields.io/github/license/thaicn1712/guardflow-go.svg?style=flat)](LICENSE)

Validators and a retry-until-valid guard for LLM output, in Go. The Go port of [`guardflow`](https://crates.io/crates/guardflow) (Rust) — same idea, adapted to Go idioms: no graph-framework dependency, just an `http`-middleware-style decorator around whatever `func(ctx, input) (string, error)` you already call your LLM with.

## Install

```bash
go get github.com/thaicn1712/guardflow-go
```

## Usage

Wrap any handler so its output is checked, retried until valid:

```go
guard := guardflow.New().
    With(guardflow.NotEmpty{}).
    With(guardflow.MaxLength(2000))

guarded := guardflow.WithGuard(myLLMHandler, guard, 3) // up to 3 attempts

output, err := guarded(ctx, "what does the fox say?")
```

`myLLMHandler` is any `func(ctx context.Context, input string) (string, error)` — the shape most LLM call wrappers in Go already have. No interface to implement, no framework to adopt.

## Built-in validators

`NotEmpty`, `MaxLength`, `MinLength`, `MatchesRegex`, `OneOf`, `ValidJSON`, `NoPII` (email/phone pattern match), `Profanity` (word-list match), `Truncate` (never fails — offers a truncated `Fixed` value instead).

## Fixing instead of rejecting

Some validators (like `Truncate`) can repair what they'd otherwise reject. `Guard.Fix` applies that repair and re-checks:

```go
guard := guardflow.New().With(guardflow.Truncate(280))
fixed, outcome := guard.Fix(tooLongText) // fixed is truncated to 280 runes, outcome.Passed == true
```

## Async validators

For checks that need network access — a moderation API, an LLM-as-judge call — implement `AsyncValidator` instead of `Validator`, and use `AsyncGuard` / `WithAsyncGuard`:

```go
type ModerationCheck struct{ client *moderation.Client }

func (m ModerationCheck) Validate(ctx context.Context, text string) (guardflow.ValidationOutcome, error) {
    result, err := m.client.Check(ctx, text)
    if err != nil {
        return guardflow.ValidationOutcome{}, err
    }
    if result.Flagged {
        return guardflow.ValidationOutcome{Passed: false, Reason: result.Reason}, nil
    }
    return guardflow.ValidationOutcome{Passed: true}, nil
}

guard := guardflow.NewAsync().With(ModerationCheck{client: client})
guarded := guardflow.WithAsyncGuard(myLLMHandler, guard, 3)
```

## Examples

```bash
go run ./examples/retry_until_valid
```

## Benchmarks

`go test -bench=.`:

| Scenario | Time |
|---|---|
| `Guard.Check`, 3 validators (`NotEmpty` + `MaxLength` + `NoPII`) | ~7.5 µs |
| `WithGuard`-wrapped handler, single passing attempt | ~60 ns |

## License

MIT
