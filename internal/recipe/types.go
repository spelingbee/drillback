// Package recipe holds the recipe types, the loader, the restricted templating
// context, and JSON Schema validation.
//
// It imports nothing else from internal/: everything depends on recipe, and recipe
// depends on nothing. See SPEC.md section 13.1.
package recipe

// Recipe is a parsed recipe.yaml.
type Recipe struct {
	APIVersion string            `yaml:"apiVersion" json:"apiVersion"`
	Kind       string            `yaml:"kind"       json:"kind"`
	Metadata   Metadata          `yaml:"metadata"   json:"metadata"`
	Vars       map[string]any    `yaml:"vars,omitempty"  json:"vars,omitempty"`
	Inputs     map[string]*Input `yaml:"inputs"     json:"inputs"`
	Ready      []*Probe          `yaml:"ready,omitempty" json:"ready,omitempty"`
	Checks     []*Check          `yaml:"checks"     json:"checks"`
	Test       *TestSpec         `yaml:"test,omitempty"  json:"test,omitempty"`

	// Provenance, filled in by the loader. Not part of the recipe file.
	Dir        string   `yaml:"-" json:"-"`
	File       string   `yaml:"-" json:"-"`
	Bundled    bool     `yaml:"-" json:"-"`
	Digest     string   `yaml:"-" json:"-"`
	InputOrder []string `yaml:"-" json:"-"`
	Raw        []byte   `yaml:"-" json:"-"`
}

// Metadata is the recipe's identity. It is prose and is never templated.
type Metadata struct {
	Name        string   `yaml:"name"        json:"name"`
	Title       string   `yaml:"title"       json:"title"`
	Description string   `yaml:"description" json:"description"`
	Maintainers []string `yaml:"maintainers,omitempty" json:"maintainers,omitempty"`
	Upstream    string   `yaml:"upstream,omitempty"    json:"upstream,omitempty"`
	Tags        []string `yaml:"tags,omitempty"        json:"tags,omitempty"`
}

// Input is one logical input the application needs: a directory, a SQLite file, or a
// PostgreSQL dump. The recipe names it and guesses a path; the user owns the path.
type Input struct {
	Kind        string    `yaml:"kind"         json:"kind"`
	Title       string    `yaml:"title"        json:"title"`
	Description string    `yaml:"description,omitempty" json:"description,omitempty"`
	DefaultPath string    `yaml:"default_path" json:"default_path"`
	Required    *bool     `yaml:"required,omitempty" json:"required,omitempty"`
	Within      string    `yaml:"within,omitempty"   json:"within,omitempty"`
	Mount       *Mount    `yaml:"mount,omitempty"    json:"mount,omitempty"`
	Load        *LoadSpec `yaml:"load,omitempty"     json:"load,omitempty"`
}

// IsRequired reports whether a missing input is a hard failure. Inputs are required
// unless a recipe says otherwise.
func (i *Input) IsRequired() bool { return i.Required == nil || *i.Required }

// Mount says which compose service sees a dir input, and under which environment
// variable the recipe compose.yaml refers to it.
type Mount struct {
	Env  string `yaml:"env"  json:"env"`
	Into string `yaml:"into" json:"into"`
}

// LoadSpec describes how an input is made live: loaded into a database service, or,
// for SQLite, verified in place.
type LoadSpec struct {
	Service        string `yaml:"service,omitempty"  json:"service,omitempty"`
	Database       string `yaml:"database,omitempty" json:"database,omitempty"`
	User           string `yaml:"user,omitempty"     json:"user,omitempty"`
	Timeout        string `yaml:"timeout,omitempty"  json:"timeout,omitempty"`
	IntegrityCheck bool   `yaml:"integrity_check,omitempty" json:"integrity_check,omitempty"`
}

// Probe is a ready probe. Probes are retried until they succeed or time out.
type Probe struct {
	Name         string   `yaml:"name" json:"name"`
	Kind         string   `yaml:"kind" json:"kind"`
	Timeout      string   `yaml:"timeout,omitempty"  json:"timeout,omitempty"`
	Interval     string   `yaml:"interval,omitempty" json:"interval,omitempty"`
	URL          string   `yaml:"url,omitempty"      json:"url,omitempty"`
	ExpectStatus int      `yaml:"expect_status,omitempty" json:"expect_status,omitempty"`
	Service      string   `yaml:"service,omitempty"  json:"service,omitempty"`
	Port         int      `yaml:"port,omitempty"     json:"port,omitempty"`
	User         string   `yaml:"user,omitempty"     json:"user,omitempty"`
	Command      []string `yaml:"command,omitempty"  json:"command,omitempty"`
}

// Check is one assertion about the restored application. Checks run once, in order,
// and every check runs even after an earlier one has failed.
type Check struct {
	ID        string   `yaml:"id"    json:"id"`
	Title     string   `yaml:"title" json:"title"`
	Kind      string   `yaml:"kind"  json:"kind"`
	Timeout   string   `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Expect    Expect   `yaml:"expect" json:"expect"`
	URL       string   `yaml:"url,omitempty"        json:"url,omitempty"`
	Method    string   `yaml:"method,omitempty"     json:"method,omitempty"`
	BasicAuth []string `yaml:"basic_auth,omitempty" json:"basic_auth,omitempty"`
	JSONBody  string   `yaml:"json_body,omitempty"  json:"json_body,omitempty"`
	Driver    string   `yaml:"driver,omitempty"     json:"driver,omitempty"`
	Query     string   `yaml:"query,omitempty"      json:"query,omitempty"`
	Service   string   `yaml:"service,omitempty"    json:"service,omitempty"`
	Database  string   `yaml:"database,omitempty"   json:"database,omitempty"`
	User      string   `yaml:"user,omitempty"       json:"user,omitempty"`
	File      string   `yaml:"file,omitempty"       json:"file,omitempty"`
	Path      string   `yaml:"path,omitempty"       json:"path,omitempty"`
	Command   []string `yaml:"command,omitempty"    json:"command,omitempty"`
}

// Expect is the closed vocabulary a check may assert with. A recipe is data, not a
// program: there is no expression language here on purpose. See SPEC.md section 3.3.
type Expect struct {
	Status         *int    `yaml:"status,omitempty"           json:"status,omitempty"`
	StatusIn       []int   `yaml:"status_in,omitempty"        json:"status_in,omitempty"`
	BodyMatches    string  `yaml:"body_matches,omitempty"     json:"body_matches,omitempty"`
	BodyNotMatches string  `yaml:"body_not_matches,omitempty" json:"body_not_matches,omitempty"`
	JSONPath       string  `yaml:"json_path,omitempty"        json:"json_path,omitempty"`
	JSONPathEquals *string `yaml:"json_path_equals,omitempty" json:"json_path_equals,omitempty"`
	JSONPathIntMin *int    `yaml:"json_path_int_min,omitempty" json:"json_path_int_min,omitempty"`
	JSONPathLenMin *int    `yaml:"json_path_len_min,omitempty" json:"json_path_len_min,omitempty"`
	ExitCode       *int    `yaml:"exit_code,omitempty"        json:"exit_code,omitempty"`
	StdoutMatches  string  `yaml:"stdout_matches,omitempty"   json:"stdout_matches,omitempty"`
	StderrMatches  string  `yaml:"stderr_matches,omitempty"   json:"stderr_matches,omitempty"`
	ScalarEquals   *string `yaml:"scalar_equals,omitempty"    json:"scalar_equals,omitempty"`
	ScalarIntMin   *int    `yaml:"scalar_int_min,omitempty"   json:"scalar_int_min,omitempty"`
	ScalarIntMax   *int    `yaml:"scalar_int_max,omitempty"   json:"scalar_int_max,omitempty"`
	RowsMin        *int    `yaml:"rows_min,omitempty"         json:"rows_min,omitempty"`
	RowsMax        *int    `yaml:"rows_max,omitempty"         json:"rows_max,omitempty"`
	Exists         *bool   `yaml:"exists,omitempty"           json:"exists,omitempty"`
	IsDir          *bool   `yaml:"is_dir,omitempty"           json:"is_dir,omitempty"`
	NotEmpty       *bool   `yaml:"not_empty,omitempty"        json:"not_empty,omitempty"`
	SizeMin        *int64  `yaml:"size_min,omitempty"         json:"size_min,omitempty"`
	Glob           string  `yaml:"glob,omitempty"             json:"glob,omitempty"`
	GlobMinCount   *int    `yaml:"glob_min_count,omitempty"   json:"glob_min_count,omitempty"`
}

// TestSpec drives the round-trip harness. It is parsed and validated in this build;
// the harness that executes it is not implemented yet. See PROGRESS.md.
type TestSpec struct {
	Seed   []*Step `yaml:"seed,omitempty"   json:"seed,omitempty"`
	Export []*Step `yaml:"export,omitempty" json:"export,omitempty"`
}

// Step is one harness action.
type Step struct {
	Name         string   `yaml:"name" json:"name"`
	Kind         string   `yaml:"kind" json:"kind"`
	Timeout      string   `yaml:"timeout,omitempty"  json:"timeout,omitempty"`
	Produces     string   `yaml:"produces,omitempty" json:"produces,omitempty"`
	Service      string   `yaml:"service,omitempty"  json:"service,omitempty"`
	User         string   `yaml:"user,omitempty"     json:"user,omitempty"`
	Command      []string `yaml:"command,omitempty"  json:"command,omitempty"`
	URL          string   `yaml:"url,omitempty"      json:"url,omitempty"`
	Method       string   `yaml:"method,omitempty"   json:"method,omitempty"`
	BasicAuth    []string `yaml:"basic_auth,omitempty" json:"basic_auth,omitempty"`
	JSONBody     string   `yaml:"json_body,omitempty"  json:"json_body,omitempty"`
	ExpectStatus int      `yaml:"expect_status,omitempty" json:"expect_status,omitempty"`
}
