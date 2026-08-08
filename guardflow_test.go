package guardflow

import (
	"context"
	"errors"
	"testing"
)

func TestGuardCheckPassesWhenAllValidatorsPass(t *testing.T) {
	g := New().With(NotEmpty{}).With(MaxLength(10))
	outcome := g.Check("hello")
	if !outcome.Passed {
		t.Fatalf("expected pass, got failure: %s", outcome.Reason)
	}
}

func TestGuardCheckStopsAtFirstFailure(t *testing.T) {
	g := New().With(NotEmpty{}).With(MaxLength(3))
	outcome := g.Check("hello")
	if outcome.Passed {
		t.Fatal("expected failure")
	}
}

func TestGuardFixAppliesTruncateAndRechecks(t *testing.T) {
	g := New().With(Truncate(5))
	fixed, outcome := g.Fix("hello world")
	if fixed != "hello" {
		t.Fatalf("expected \"hello\", got %q", fixed)
	}
	if !outcome.Passed {
		t.Fatalf("expected the fixed text to pass, got: %s", outcome.Reason)
	}
}

func TestGuardFixReturnsOriginalWhenNoFixAvailable(t *testing.T) {
	g := New().With(NotEmpty{})
	fixed, outcome := g.Fix("")
	if fixed != "" {
		t.Fatalf("expected original text back, got %q", fixed)
	}
	if outcome.Passed {
		t.Fatal("expected failure to persist with no fix available")
	}
}

func TestValidators(t *testing.T) {
	cases := []struct {
		name      string
		validator Validator
		text      string
		wantPass  bool
	}{
		{"NotEmpty passes", NotEmpty{}, "hi", true},
		{"NotEmpty fails on blank", NotEmpty{}, "   ", false},
		{"MaxLength passes under limit", MaxLength(5), "hi", true},
		{"MaxLength fails over limit", MaxLength(1), "hi", false},
		{"MinLength passes at limit", MinLength(2), "hi", true},
		{"MinLength fails under limit", MinLength(5), "hi", false},
		{"OneOf passes on match", OneOf{Choices: []string{"a", "b"}}, "a", true},
		{"OneOf fails on no match", OneOf{Choices: []string{"a", "b"}}, "c", false},
		{"ValidJSON passes on object", ValidJSON{}, `{"a":1}`, true},
		{"ValidJSON fails on garbage", ValidJSON{}, `not json`, false},
		{"NoPII fails on email", NoPII{}, "contact me at a@b.com", false},
		{"NoPII fails on phone", NoPII{}, "call 555-123-4567", false},
		{"NoPII passes on clean text", NoPII{}, "hello there", true},
		{"Profanity fails on match", NewProfanity(), "well damn", false},
		{"Profanity passes on clean text", NewProfanity(), "well shoot", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			outcome := c.validator.Validate(c.text)
			if outcome.Passed != c.wantPass {
				t.Fatalf("Validate(%q) passed=%v, want %v (reason: %s)", c.text, outcome.Passed, c.wantPass, outcome.Reason)
			}
		})
	}
}

func TestMatchesRegex(t *testing.T) {
	v, err := NewMatchesRegex(`^\d+$`)
	if err != nil {
		t.Fatal(err)
	}
	if !v.Validate("12345").Passed {
		t.Fatal("expected digits-only text to pass")
	}
	if v.Validate("12a45").Passed {
		t.Fatal("expected mixed text to fail")
	}
}

func TestWithGuardRetriesUntilPass(t *testing.T) {
	attempts := 0
	flaky := Handler(func(ctx context.Context, input string) (string, error) {
		attempts++
		if attempts < 2 {
			return "no", nil
		}
		return "this response is long enough", nil
	})

	guarded := WithGuard(flaky, New().With(MinLength(15)), 3)
	output, err := guarded(context.Background(), "irrelevant")
	if err != nil {
		t.Fatal(err)
	}
	if output != "this response is long enough" {
		t.Fatalf("unexpected output: %q", output)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestWithGuardReturnsLastAttemptWhenNoneEverPass(t *testing.T) {
	always := Handler(func(ctx context.Context, input string) (string, error) {
		return "no", nil
	})
	guarded := WithGuard(always, New().With(MinLength(50)), 2)
	output, err := guarded(context.Background(), "irrelevant")
	if err != nil {
		t.Fatal(err)
	}
	if output != "no" {
		t.Fatalf("expected last attempt's output, got %q", output)
	}
}

func TestWithGuardPropagatesHandlerError(t *testing.T) {
	boom := errors.New("boom")
	failing := Handler(func(ctx context.Context, input string) (string, error) {
		return "", boom
	})
	guarded := WithGuard(failing, New(), 3)
	_, err := guarded(context.Background(), "irrelevant")
	if !errors.Is(err, boom) {
		t.Fatalf("expected boom error, got %v", err)
	}
}

type llmAsJudge struct{ reject bool }

func (j llmAsJudge) Validate(ctx context.Context, text string) (ValidationOutcome, error) {
	if j.reject {
		return ValidationOutcome{Passed: false, Reason: "judge rejected it"}, nil
	}
	return ValidationOutcome{Passed: true}, nil
}

func TestAsyncGuardCheck(t *testing.T) {
	g := NewAsync().With(llmAsJudge{reject: false})
	outcome, err := g.Check(context.Background(), "anything")
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.Passed {
		t.Fatal("expected pass")
	}

	g = NewAsync().With(llmAsJudge{reject: true})
	outcome, err = g.Check(context.Background(), "anything")
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Passed {
		t.Fatal("expected failure")
	}
}

func TestWithAsyncGuardRetriesUntilPass(t *testing.T) {
	handlerCalls := 0
	handler := Handler(func(ctx context.Context, input string) (string, error) {
		handlerCalls++
		return "output", nil
	})
	judge := &countingJudge{failUntil: 2}
	guarded := WithAsyncGuard(handler, NewAsync().With(judge), 3)

	output, err := guarded(context.Background(), "irrelevant")
	if err != nil {
		t.Fatal(err)
	}
	if output != "output" {
		t.Fatalf("unexpected output: %q", output)
	}
	if handlerCalls != 2 {
		t.Fatalf("expected 2 handler calls, got %d", handlerCalls)
	}
}

type countingJudge struct {
	calls     int
	failUntil int
}

func (j *countingJudge) Validate(ctx context.Context, text string) (ValidationOutcome, error) {
	j.calls++
	if j.calls < j.failUntil {
		return ValidationOutcome{Passed: false, Reason: "not yet"}, nil
	}
	return ValidationOutcome{Passed: true}, nil
}
