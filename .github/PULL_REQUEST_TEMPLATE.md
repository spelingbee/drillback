<!--
Thank you. If this adds a recipe, the checklist below is the whole review: a round
trip that passes means a maintainer does not have to understand your application to
trust it.

Delete whichever section does not apply.
-->

## What this changes

<!-- One or two sentences. The diff already says what; say why. -->

---

## Adding or changing a recipe

- [ ] **`restored recipe test ./recipes/<name>` passes locally.** Paste the tail of the
      real output below — that is what makes this reviewable.
- [ ] At least one check **fails** against an empty stack, and it is a check about
      *data*, not about the application starting. (Stage A proves this; if it says
      PASS-BY-STARTUP-REFUSAL, say why the application refuses to start empty.)
- [ ] Every image in `compose.yaml` carries a **pinned tag**, not `latest`.
- [ ] `README.md` says **which of my directories is this input** for the two or three
      ways the application is commonly deployed.
- [ ] `metadata.maintainers` names a GitHub handle — yours, unless you say otherwise.
- [ ] Nothing in `compose.yaml` publishes a port, uses a host namespace, or binds a
      path outside the workspace. (`restored recipe validate --strict` checks this.)
- [ ] Anything the round trip does **not** prove is stated in the README or in a
      comment on the step.

```text
$ restored recipe test ./recipes/<name>

<paste the real output, not a summary>
```

Round-trip time on your machine: <!-- e.g. 2m53s -->

---

## Changing the Go code

- [ ] `gofmt -l .` prints nothing, `go vet ./...` and `golangci-lint run` are clean.
- [ ] `go test ./...` passes **without docker or restic installed**.
- [ ] `go test -tags integration ./...` passes, or you have said which part you could
      not run and why.
- [ ] New behaviour has a test that fails without the change.
- [ ] A new dependency has a line in `DECISIONS.md` justifying it.
- [ ] An architectural decision has an ADR in `DECISIONS.md`, rather than quietly
      reversing an existing one.
- [ ] Any terminal output shown in the docs was **captured by
      `scripts/capture-demo.sh`**, not typed.

---

## Anything a reviewer should know

<!--
Something you were unsure about, a tradeoff you made, a thing you could not test on
your machine. An admitted gap is worth more than an unexamined claim.
-->
