package runner

import (
	"testing"
	"time"
)

// The defaults used to guarantee the bug in ARCH-01: a 15-minute run budget with a
// 10-minute restore budget and a 5-minute ready budget inside it left nothing at all
// for COMPOSE UP, LOAD DUMPS and the checks, so a large restore ran the whole run out
// of time and the report blamed the backup.
func TestDefaultRunBudgetExceedsTheStagesInsideIt(t *testing.T) {
	var o Options
	o.applyDefaults()

	inner := o.RestoreTimeout + o.ReadyTimeout
	if inner >= o.Timeout {
		t.Fatalf("the stage budgets (%s restore + %s ready = %s) leave nothing inside "+
			"the run budget of %s", o.RestoreTimeout, o.ReadyTimeout, inner, o.Timeout)
	}
}

// A user who shortens --timeout means it. The stage budgets have to come down with
// it, or the first stage consumes the whole run and every stage after it is judged
// against a deadline it never had a chance against.
func TestStageBudgetsAreClampedToTheRunBudget(t *testing.T) {
	o := Options{Timeout: 2 * time.Minute}
	o.applyDefaults()

	if o.RestoreTimeout > o.Timeout/2 {
		t.Errorf("restore budget %s is more than half of the %s run budget",
			o.RestoreTimeout, o.Timeout)
	}
	if o.ReadyTimeout > o.Timeout/4 {
		t.Errorf("ready budget %s is more than a quarter of the %s run budget",
			o.ReadyTimeout, o.Timeout)
	}
	if o.CheckTimeout > o.Timeout/8 {
		t.Errorf("check budget %s is more than an eighth of the %s run budget",
			o.CheckTimeout, o.Timeout)
	}
	if o.RestoreTimeout+o.ReadyTimeout >= o.Timeout {
		t.Errorf("the clamped stage budgets still fill the whole run budget")
	}
}

// Clamping is downward only: an explicit budget smaller than its share stays where
// the user put it.
func TestAnExplicitStageBudgetIsNotRaised(t *testing.T) {
	o := Options{Timeout: time.Hour, RestoreTimeout: 30 * time.Second, ReadyTimeout: time.Second}
	o.applyDefaults()

	if o.RestoreTimeout != 30*time.Second {
		t.Errorf("restore budget was changed from 30s to %s", o.RestoreTimeout)
	}
	if o.ReadyTimeout != time.Second {
		t.Errorf("ready budget was changed from 1s to %s", o.ReadyTimeout)
	}
}
