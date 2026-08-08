// A flaky "LLM" that answers too short to be useful on its first attempt,
// wrapped with WithGuard so it retries until the response is long enough.
package main

import (
	"context"
	"fmt"

	"github.com/thaicn1712/guardflow-go"
)

func main() {
	attempt := 0
	flakyLLM := guardflow.Handler(func(ctx context.Context, input string) (string, error) {
		attempt++
		if attempt < 2 {
			return "not sure", nil
		}
		return "the quick brown fox jumps over the lazy dog", nil
	})

	guard := guardflow.New().With(guardflow.MinLength(15))
	guarded := guardflow.WithGuard(flakyLLM, guard, 3)

	output, err := guarded(context.Background(), "what does the fox say?")
	if err != nil {
		panic(err)
	}

	fmt.Printf("accepted response after %d attempt(s): %q\n", attempt, output)
}
