// Package guardflow provides validators and a retry-until-valid guard for
// LLM output, plus an HTTP-middleware-style decorator for wrapping any
// text-generating handler with one.
package guardflow

import "context"

// ValidationOutcome is the result of running one Validator against a piece
// of text.
type ValidationOutcome struct {
	Passed bool
	Reason string
	// Fixed holds a corrected version of the text, if the validator that
	// produced this outcome can repair what it rejects (e.g. Truncate).
	// Nil means no fix is available.
	Fixed *string
}

// Validator checks a piece of text synchronously.
type Validator interface {
	Validate(text string) ValidationOutcome
}

// AsyncValidator checks a piece of text against something that needs
// network access — a moderation API, an LLM-as-judge call, anything a
// synchronous Validator can't do.
type AsyncValidator interface {
	Validate(ctx context.Context, text string) (ValidationOutcome, error)
}

// Guard runs a sequence of synchronous Validators against text, stopping at
// the first failure.
type Guard struct {
	validators []Validator
}

// New creates an empty Guard.
func New() *Guard {
	return &Guard{}
}

// With appends a Validator and returns the Guard for chaining.
func (g *Guard) With(v Validator) *Guard {
	g.validators = append(g.validators, v)
	return g
}

// Check runs every validator in order, returning the first failing outcome,
// or a passing outcome if all validators pass.
func (g *Guard) Check(text string) ValidationOutcome {
	for _, v := range g.validators {
		if outcome := v.Validate(text); !outcome.Passed {
			return outcome
		}
	}
	return ValidationOutcome{Passed: true}
}

// Fix runs Check, and if it fails with a Fixed value available, returns the
// fixed text re-checked; otherwise returns the original text and outcome.
func (g *Guard) Fix(text string) (string, ValidationOutcome) {
	outcome := g.Check(text)
	if outcome.Passed || outcome.Fixed == nil {
		return text, outcome
	}
	fixed := *outcome.Fixed
	return fixed, g.Check(fixed)
}

// AsyncGuard runs a sequence of AsyncValidators against text, stopping at
// the first failure.
type AsyncGuard struct {
	validators []AsyncValidator
}

// NewAsync creates an empty AsyncGuard.
func NewAsync() *AsyncGuard {
	return &AsyncGuard{}
}

// With appends an AsyncValidator and returns the AsyncGuard for chaining.
func (g *AsyncGuard) With(v AsyncValidator) *AsyncGuard {
	g.validators = append(g.validators, v)
	return g
}

// Check runs every validator in order, returning the first failing outcome,
// or a passing outcome if all validators pass. Returns an error only if a
// validator itself errors (e.g. the network call behind it failed).
func (g *AsyncGuard) Check(ctx context.Context, text string) (ValidationOutcome, error) {
	for _, v := range g.validators {
		outcome, err := v.Validate(ctx, text)
		if err != nil {
			return ValidationOutcome{}, err
		}
		if !outcome.Passed {
			return outcome, nil
		}
	}
	return ValidationOutcome{Passed: true}, nil
}
