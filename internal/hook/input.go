package hook

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// RefUpdate represents one line from the pre-receive stdin.
type RefUpdate struct {
	OldValue string // old commit SHA (all-zeros for new branch)
	NewValue string // new commit SHA (all-zeros for deleted branch)
	RefName  string // full ref name, e.g. refs/heads/dev/login
}

// ParseStdin reads all pre-receive stdin lines and returns []RefUpdate.
// Lines that are blank or malformed are skipped with a warning returned in errs.
func ParseStdin(r io.Reader) ([]RefUpdate, []error) {
	var refs []RefUpdate
	var errs []error

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			errs = append(errs, fmt.Errorf("malformed stdin line (expected 3 fields, got %d): %q", len(fields), line))
			continue
		}
		refs = append(refs, RefUpdate{
			OldValue: fields[0],
			NewValue: fields[1],
			RefName:  fields[2],
		})
	}
	if err := scanner.Err(); err != nil {
		errs = append(errs, fmt.Errorf("reading stdin: %w", err))
	}
	return refs, errs
}
