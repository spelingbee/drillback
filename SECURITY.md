# Security policy

## Reporting a vulnerability

**Do not open a public issue.**

Use GitHub's private reporting: **Security → Report a vulnerability** on
<https://github.com/spelingbee/restored/security/advisories/new>. That opens a private
advisory only the maintainers can see.

What to expect:

| | |
|---|---|
| First response | within 72 hours |
| Assessment | within 7 days |
| Fix or a public explanation of why not | within 30 days for anything in the table below |

Credit in the advisory and the release notes unless you would rather not be named.

---

## What counts as a vulnerability here

`restored` runs somebody else's backup — data of unknown provenance — inside
containers, on a machine that also holds things that backup should never touch. The
isolation is the product. Anything that gets through it is a vulnerability.

**In scope:**

- **Escape from the run workspace.** Anything that lets a recipe, a compose file, or
  the *contents of a backup* read, write, or delete a path outside
  `<tmp>/restored-<runid>`. The archived-symlink case is the obvious one, and it is
  why `internal/workspace.Sanitise` exists.
- **Escape from the run's isolation.** A recipe that reaches the host network,
  publishes a port, obtains the host PID or IPC namespace, gains a capability, or gets
  a privileged container, without `restored recipe validate` refusing it. The rules are
  in `schema/compose-safety.schema.json` and `internal/recipe/safety`; a way round any
  of them is a bug in scope.
- **Command injection.** A recipe field that reaches a shell. Recipes are data:
  `command:` is an argv and is passed through unchanged, and any path from recipe text
  to `sh -c` on the host is in scope.
- **Secret disclosure.** `RESTIC_PASSWORD`, `RESTIC_PASSWORD_FILE`,
  `RESTIC_PASSWORD_COMMAND`, cloud credentials, or the contents of a restored backup
  appearing in a report, a log, a process argument list, or a JSON document. restic's
  environment is passed through and is never parsed or logged; a leak is a bug.
- **Path traversal in a restored tree.** `..` components or absolute paths in an
  archive escaping the extraction directory.
- **Anything that makes `restored` write outside its workspace**, including the
  `--report`, `--workspace` and `--hints` flags being made to write somewhere they
  should not.
- **A malicious recipe from a pull request** doing any of the above while passing
  `recipe validate` and `recipe test`. Recipes come from strangers by design.

**Out of scope**, deliberately, and documented in SPEC.md § 9.4:

- **The docker socket is root-equivalent.** `restored` needs the daemon, and anyone who
  can run `restored` can already run `docker run -v /:/host`. `restored` does not, and
  cannot, protect a machine from its own operator.
- **A recipe you wrote yourself, running on your own machine.** The isolation rules
  protect you from *other people's* recipes and *other people's* backup contents. They
  are not a sandbox against yourself.
- **Denial of service by a large backup.** A restore drill on a 900 GB snapshot will
  fill the disk. Use `--workspace` to point at somewhere with room.
- **The images a recipe pulls.** A recipe naming a malicious upstream image is a supply
  chain problem for that image, not a flaw in `restored` — though a recipe pinning an
  image by digest instead of a tag is always better, and `--strict` says so.
- **The throwaway credentials in the bundled recipes.** `restored-throwaway` and
  `restored-recipe-test` are literals in a public repository on purpose. They belong to
  databases that exist for ninety seconds, on an internal network, with no published
  port, and are destroyed with `compose down -v`. If one of them ever protects
  something real, that is the bug.
- **Vulnerabilities in the applications being restored.** A drill starts a real Gitea
  or a real Nextcloud. If that version has a CVE, the drill has it too, for ninety
  seconds, unreachable from anywhere. Report it upstream.

## Supported versions

There has been no release. Until v0.1.0, `main` is the only supported version.
