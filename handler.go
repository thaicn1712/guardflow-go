package guardflow

import "context"

// Handler produces text from an input string — the shape most LLM call
// wrappers in Go already have. guardflow decorates Handlers the way
// net/http middleware decorates http.Handler: wrap once, get the behavior
// on every call.
type Handler func(ctx context.Context, input string) (string, error)

// WithGuard wraps next so its output is checked by guard on every call,
// retrying up to maxAttempts times until a check passes. If no attempt
// passes, the last attempt's output is returned rather than an error —
// callers that need to distinguish that case should check guard.Check on
// the result themselves.
func WithGuard(next Handler, guard *Guard, maxAttempts int) Handler {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	return func(ctx context.Context, input string) (string, error) {
		var last string
		for i := 0; i < maxAttempts; i++ {
			output, err := next(ctx, input)
			if err != nil {
				return "", err
			}
			last = output
			if guard.Check(output).Passed {
				return output, nil
			}
		}
		return last, nil
	}
}

// WithAsyncGuard is WithGuard for an AsyncGuard, propagating any error a
// validator itself returns (as opposed to a failed check, which just moves
// to the next attempt).
func WithAsyncGuard(next Handler, guard *AsyncGuard, maxAttempts int) Handler {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	return func(ctx context.Context, input string) (string, error) {
		var last string
		for i := 0; i < maxAttempts; i++ {
			output, err := next(ctx, input)
			if err != nil {
				return "", err
			}
			last = output
			outcome, err := guard.Check(ctx, output)
			if err != nil {
				return "", err
			}
			if outcome.Passed {
				return output, nil
			}
		}
		return last, nil
	}
}
