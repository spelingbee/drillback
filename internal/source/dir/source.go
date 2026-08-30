package dir

import (
	"context"
	"path/filepath"

	"github.com/spelingbee/restored/internal/source"
)

// Source is the dir implementation of source.Source: a tree that is already restored
// on disk, which is what makes the first run cost nothing. See DECISIONS.md ADR-063.
type Source struct {
	From string
}

// New returns a dir source rooted at from.
func New(from string) *Source { return &Source{From: from} }

// Kind is the name the --source flag and the report use.
func (s *Source) Kind() string { return "dir" }

// Preflight answers the two questions that are worth answering before a workspace
// exists: was a tree named, and is it one.
func (s *Source) Preflight(_ context.Context) error {
	if s.From == "" {
		return errNoTree
	}
	return Check(s.From)
}

// Fetch materialises nothing: the data is already where it is. It resolves the tree to
// an absolute path so that the report and every later path operation agree about it.
func (s *Source) Fetch(_ context.Context, _ []source.Request, _ string) (source.Fetched, error) {
	var f source.Fetched
	abs, err := filepath.Abs(s.From)
	if err != nil {
		return f, err
	}
	return source.Fetched{
		Descriptor: source.Descriptor{Kind: "dir", Repository: abs},
		Locate:     func(backupPath string) string { return Locate(abs, backupPath) },
		// The tree belongs to the user. Moving files out of somebody's live
		// directory is not a thing a restore drill may do.
		Disposable: false,
	}, nil
}
