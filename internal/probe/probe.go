// Package probe runs the ready probes: the retried questions that say the application
// has come up at all. A probe that never succeeds ends the run as RESTORE UNUSABLE,
// not as a tool error, because "the app did not come up" is a verdict about the
// backup. See DECISIONS.md ADR-023.
package probe

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spelingbee/drillback/internal/check"
	"github.com/spelingbee/drillback/internal/recipe"
)

// Result is one probe, retried until it succeeded or ran out of time.
type Result struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Attempts   int    `json:"attempts"`
	DurationMS int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

// Defaults for a probe that does not set its own.
const (
	DefaultTimeout  = 180 * time.Second
	DefaultInterval = 3 * time.Second
)

// Run retries one probe until it succeeds or its budget runs out.
func Run(ctx context.Context, e *check.Executor, p *recipe.Probe, budget time.Duration) Result {
	timeout := parseDuration(p.Timeout, DefaultTimeout)
	if budget > 0 && budget < timeout {
		timeout = budget
	}
	interval := parseDuration(p.Interval, DefaultInterval)

	start := time.Now()
	deadline := start.Add(timeout)
	res := Result{Name: p.Name, Status: "fail"}

	for attempt := 1; ; attempt++ {
		res.Attempts = attempt
		obs := attemptOnce(ctx, e, p, time.Until(deadline))
		if obs.Error == "" {
			res.Status = "ok"
			res.DurationMS = time.Since(start).Milliseconds()
			return res
		}
		res.Error = obs.Error
		if ctx.Err() != nil {
			res.Error = ctx.Err().Error()
			break
		}
		if time.Now().Add(interval).After(deadline) {
			break
		}
		select {
		case <-ctx.Done():
			res.Error = ctx.Err().Error()
			res.DurationMS = time.Since(start).Milliseconds()
			return res
		case <-time.After(interval):
		}
	}
	res.DurationMS = time.Since(start).Milliseconds()
	if res.Error == "" {
		res.Error = "the probe never succeeded"
	}
	return res
}

func attemptOnce(ctx context.Context, e *check.Executor, p *recipe.Probe, left time.Duration) check.Observation {
	if left <= 0 {
		return check.Observation{Error: "out of time"}
	}
	per := left
	if per > 30*time.Second {
		per = 30 * time.Second
	}
	switch p.Kind {
	case "http":
		obs := e.HTTP(ctx, check.HTTPRequest{URL: p.URL, Timeout: per})
		if obs.Error != "" {
			return obs
		}
		want := p.ExpectStatus
		if want == 0 {
			want = 200
		}
		if obs.Status == nil || *obs.Status != want {
			obs.Error = fmt.Sprintf("%s answered %s, expected %d", p.URL, statusText(obs.Status), want)
		}
		return obs
	case "tcp":
		return e.TCP(ctx, p.Service, p.Port, per)
	case "exec":
		obs := e.Exec(ctx, p.Service, p.User, p.Command, per)
		if obs.Error != "" {
			return obs
		}
		if obs.ExitCode == nil || *obs.ExitCode != 0 {
			obs.Error = fmt.Sprintf("%s exited %s: %s",
				strings.Join(p.Command, " "), statusText(obs.ExitCode), firstLine(obs.Stderr+obs.Stdout))
		}
		return obs
	default:
		return check.Observation{Error: fmt.Sprintf("unknown probe kind %q", p.Kind)}
	}
}

// RunAll runs the probes in order under one shared budget. The first probe that never
// succeeds stops the sequence: there is nothing to learn from probing an application
// whose database never accepted a connection.
func RunAll(ctx context.Context, e *check.Executor, probes []*recipe.Probe, budget time.Duration) []Result {
	results := make([]Result, 0, len(probes))
	deadline := time.Now().Add(budget)
	for _, p := range probes {
		left := time.Until(deadline)
		if left <= 0 {
			results = append(results, Result{
				Name: p.Name, Status: "fail", Error: "the ready budget ran out before this probe started",
			})
			return results
		}
		r := Run(ctx, e, p, left)
		results = append(results, r)
		if r.Status != "ok" {
			return results
		}
	}
	return results
}

func parseDuration(s string, def time.Duration) time.Duration {
	if s == "" {
		return def
	}
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return def
	}
	return d
}

func statusText(p *int) string {
	if p == nil {
		return "nothing"
	}
	return fmt.Sprint(*p)
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		return s[:i]
	}
	return s
}
