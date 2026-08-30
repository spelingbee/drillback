// Package observe holds the two types that are both a check's output and part of the
// report's public JSON.
//
// They used to live in internal/check, which meant internal/report imported
// internal/check, and therefore internal/compose (which shells out to docker) and
// internal/sqlite (which links a SQL driver). SPEC.md section 13.1 says the opposite:
// "internal/report is a pure function of its input struct ... it never reaches back
// into check or compose".
//
// The compile time was never the problem. The problem was that `checks[].observed` -
// a field the report promises is only ever added to within a major schema version -
// was defined by a struct living in an execution package, where somebody renaming a
// field for an internal reason would break every downstream JSON consumer with a
// green build and green tests. Putting the wire types in a leaf package that both
// sides import means re-typing a report field is a change to a package whose whole
// job is the wire format. See DECISIONS.md ADR-062.
//
// This package imports nothing but the standard library, and should stay that way.
package observe

// Observation is what a check saw. Every field is a pointer or a zero-able scalar so
// that "the check did not produce this" and "the check produced zero" are different
// things in the JSON.
type Observation struct {
	Status    *int   `json:"status,omitempty"`
	BodyBytes *int   `json:"body_bytes,omitempty"`
	Matched   *bool  `json:"matched,omitempty"`
	Value     string `json:"value,omitempty"`
	Rows      *int   `json:"rows,omitempty"`
	ExitCode  *int   `json:"exit_code,omitempty"`
	Count     *int   `json:"count,omitempty"`
	Entries   *int   `json:"entries,omitempty"`
	Bytes     *int64 `json:"bytes,omitempty"`
	Exists    *bool  `json:"exists,omitempty"`
	IsDir     *bool  `json:"is_dir,omitempty"`
	Error     string `json:"error,omitempty"`

	// Not serialised: the raw material the expect keys and the hint matcher read.
	// It is deliberately kept out of the report, because a response body is the
	// most likely place for somebody's data to end up in a document they attach to
	// a bug report.
	Body    string `json:"-"`
	Stdout  string `json:"-"`
	Stderr  string `json:"-"`
	Summary string `json:"-"`
}

// Failure is one unmet expectation, in the "expect / got" shape the report prints.
type Failure struct {
	Expect string `json:"expect"`
	Got    string `json:"got"`
}
