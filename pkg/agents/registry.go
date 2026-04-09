package agents

import (
	"fmt"
	"strings"
)

var knownFrameworks = map[string]SpecFramework{
	FrameworkGSD: GSDFramework{},
}

// FrameworkFor returns the SpecFramework registered under name.
// Returns an error naming the invalid value and listing valid identifiers.
func FrameworkFor(name string) (SpecFramework, error) {
	fw, ok := knownFrameworks[name]
	if !ok {
		return nil, fmt.Errorf("unknown agentic framework %q; valid values: %s", name, validFrameworkList())
	}

	return fw, nil
}

// FrameworksFor resolves a slice of framework names to their implementations.
// Returns an error on the first unrecognised name.
func FrameworksFor(names []string) ([]SpecFramework, error) {
	result := make([]SpecFramework, 0, len(names))

	for _, name := range names {
		fw, err := FrameworkFor(name)
		if err != nil {
			return nil, err
		}

		result = append(result, fw)
	}

	return result, nil
}

func validFrameworkList() string {
	names := make([]string, 0, len(knownFrameworks))
	for name := range knownFrameworks {
		names = append(names, fmt.Sprintf("%q", name))
	}

	return strings.Join(names, ", ")
}
