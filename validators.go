package guardflow

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// NotEmpty fails on text that is empty or all whitespace.
type NotEmpty struct{}

func (NotEmpty) Validate(text string) ValidationOutcome {
	if strings.TrimSpace(text) == "" {
		return ValidationOutcome{Passed: false, Reason: "text is empty"}
	}
	return ValidationOutcome{Passed: true}
}

// MaxLength fails on text longer than N runes.
type MaxLength int

func (m MaxLength) Validate(text string) ValidationOutcome {
	if n := len([]rune(text)); n > int(m) {
		return ValidationOutcome{
			Passed: false,
			Reason: fmt.Sprintf("text is %d runes, exceeds max length %d", n, int(m)),
		}
	}
	return ValidationOutcome{Passed: true}
}

// MinLength fails on text shorter than N runes.
type MinLength int

func (m MinLength) Validate(text string) ValidationOutcome {
	if n := len([]rune(text)); n < int(m) {
		return ValidationOutcome{
			Passed: false,
			Reason: fmt.Sprintf("text is %d runes, below min length %d", n, int(m)),
		}
	}
	return ValidationOutcome{Passed: true}
}

// MatchesRegex fails on text that doesn't match the given pattern.
type MatchesRegex struct {
	Pattern *regexp.Regexp
}

func NewMatchesRegex(pattern string) (MatchesRegex, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return MatchesRegex{}, err
	}
	return MatchesRegex{Pattern: re}, nil
}

func (m MatchesRegex) Validate(text string) ValidationOutcome {
	if !m.Pattern.MatchString(text) {
		return ValidationOutcome{
			Passed: false,
			Reason: fmt.Sprintf("text does not match pattern %q", m.Pattern.String()),
		}
	}
	return ValidationOutcome{Passed: true}
}

// OneOf fails unless text exactly matches one of Choices.
type OneOf struct {
	Choices []string
}

func (o OneOf) Validate(text string) ValidationOutcome {
	for _, choice := range o.Choices {
		if text == choice {
			return ValidationOutcome{Passed: true}
		}
	}
	return ValidationOutcome{
		Passed: false,
		Reason: fmt.Sprintf("text %q is not one of %v", text, o.Choices),
	}
}

// ValidJSON fails on text that doesn't parse as JSON.
type ValidJSON struct{}

func (ValidJSON) Validate(text string) ValidationOutcome {
	var v any
	if err := json.Unmarshal([]byte(text), &v); err != nil {
		return ValidationOutcome{Passed: false, Reason: fmt.Sprintf("invalid JSON: %s", err)}
	}
	return ValidationOutcome{Passed: true}
}

// Truncate never fails — it always passes, offering a Fixed value truncated
// to N runes when the text is longer than that. Combine with Guard.Fix to
// silently shorten instead of rejecting.
type Truncate int

func (t Truncate) Validate(text string) ValidationOutcome {
	runes := []rune(text)
	if len(runes) <= int(t) {
		return ValidationOutcome{Passed: true}
	}
	fixed := string(runes[:int(t)])
	return ValidationOutcome{Passed: false, Reason: "text exceeds truncation length", Fixed: &fixed}
}

var profanityDefaultWords = []string{"damn", "hell", "crap"}

// Profanity fails if text contains any of Words (case-insensitive). Uses a
// small built-in list by default — pass your own Words for a real deny list.
type Profanity struct {
	Words []string
}

// NewProfanity returns a Profanity validator using a small built-in word list.
func NewProfanity() Profanity {
	return Profanity{Words: profanityDefaultWords}
}

func (p Profanity) Validate(text string) ValidationOutcome {
	lower := strings.ToLower(text)
	for _, word := range p.Words {
		if strings.Contains(lower, strings.ToLower(word)) {
			return ValidationOutcome{
				Passed: false,
				Reason: fmt.Sprintf("text contains disallowed word %q", word),
			}
		}
	}
	return ValidationOutcome{Passed: true}
}

var (
	emailPattern = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	phonePattern = regexp.MustCompile(`\b\d{3}[-.\s]?\d{3}[-.\s]?\d{4}\b`)
)

// NoPII fails if text contains what looks like an email address or a
// US-style phone number. Pattern-based, not exhaustive — pair with an
// AsyncValidator backed by a real PII-detection service for anything that
// needs to be airtight.
type NoPII struct{}

func (NoPII) Validate(text string) ValidationOutcome {
	if emailPattern.MatchString(text) {
		return ValidationOutcome{Passed: false, Reason: "text appears to contain an email address"}
	}
	if phonePattern.MatchString(text) {
		return ValidationOutcome{Passed: false, Reason: "text appears to contain a phone number"}
	}
	return ValidationOutcome{Passed: true}
}
