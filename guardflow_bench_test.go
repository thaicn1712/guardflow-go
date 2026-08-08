package guardflow

import (
	"context"
	"testing"
)

func BenchmarkGuardCheckThreeValidators(b *testing.B) {
	g := New().With(NotEmpty{}).With(MaxLength(500)).With(NoPII{})
	text := "this is a perfectly ordinary response with no PII in it at all"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Check(text)
	}
}

func BenchmarkWithGuardSingleAttempt(b *testing.B) {
	handler := Handler(func(ctx context.Context, input string) (string, error) {
		return "a perfectly fine response", nil
	})
	guarded := WithGuard(handler, New().With(NotEmpty{}).With(MaxLength(500)), 3)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = guarded(ctx, "irrelevant")
	}
}
