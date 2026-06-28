package hook

import (
	"strings"
	"testing"
)

func TestParseStdin(t *testing.T) {
	t.Run("normal two refs", func(t *testing.T) {
		input := "aaaa bbbb refs/heads/dev\n0000000000000000000000000000000000000000 cccc refs/heads/feature/login\n"
		refs, errs := ParseStdin(strings.NewReader(input))
		if len(errs) != 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}
		if len(refs) != 2 {
			t.Fatalf("got %d refs, want 2", len(refs))
		}
		if refs[0].OldValue != "aaaa" || refs[0].NewValue != "bbbb" || refs[0].RefName != "refs/heads/dev" {
			t.Errorf("refs[0] = %+v", refs[0])
		}
		if refs[1].OldValue != "0000000000000000000000000000000000000000" {
			t.Errorf("refs[1].OldValue = %q, want zero SHA", refs[1].OldValue)
		}
	})

	t.Run("blank lines skipped", func(t *testing.T) {
		input := "\n  \naaaa bbbb refs/heads/dev\n"
		refs, errs := ParseStdin(strings.NewReader(input))
		if len(errs) != 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}
		if len(refs) != 1 {
			t.Fatalf("got %d refs, want 1", len(refs))
		}
	})

	t.Run("malformed line reported as error", func(t *testing.T) {
		input := "aaaa bbbb\n"
		refs, errs := ParseStdin(strings.NewReader(input))
		if len(errs) == 0 {
			t.Error("expected parse error for malformed line")
		}
		if len(refs) != 0 {
			t.Errorf("expected no refs, got %d", len(refs))
		}
	})

	t.Run("empty stdin returns no refs", func(t *testing.T) {
		refs, errs := ParseStdin(strings.NewReader(""))
		if len(errs) != 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}
		if len(refs) != 0 {
			t.Errorf("expected 0 refs, got %d", len(refs))
		}
	})
}
