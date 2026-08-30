# The Homebrew tap

`brew install spelingbee/tap/restored` needs a second repository, `homebrew-tap`,
that this one pushes a cask into at release time. This file is the steps to create it.

It lives in `docs/` rather than in `dist/homebrew/`, where the cask itself is written,
because `goreleaser --clean` empties `dist/` on every run: a checked-in file there is a
file that disappears the next time somebody builds a release, and then gets committed
as a deletion by whoever runs `git add -A` afterwards. Nothing here is wired up yet: creating the repository
is stop point 4 in [CLAUDE.md](../CLAUDE.md), and pushing a cask to it is stop
point 6.

## Why a cask and not a formula

Homebrew wants a *cask* for a prebuilt binary and a *formula* for something it
compiles. `restored` ships prebuilt binaries, so it is a cask. GoReleaser agrees:
its `brews` section is deprecated in v2 in favour of `homebrew_casks`, which is what
[`.goreleaser.yaml`](../.goreleaser.yaml) uses.

Casks are macOS-only. That is the honest scope of this path. Linux users install
with `install.sh` or `go install`, both of which the README documents first.

## Creating the tap, once

1. Create a public repository named exactly **`homebrew-tap`** under the same owner
   as this one. The name matters: `brew` maps `spelingbee/tap` to
   `github.com/spelingbee/homebrew-tap` and will not find it under any other name.
   *(Stop point 4: a human creates this, not a session.)*

2. Give it a `README.md` and a `Casks/` directory. GoReleaser writes
   `Casks/restored.rb` into it; it will not create the directory for you on every
   backend, so create it with a `.gitkeep`.

3. Create a fine-grained personal access token with **contents: read and write** on
   `homebrew-tap` **only** - not on this repository, and not on the account. Store it
   in this repository as the Actions secret `HOMEBREW_TAP_GITHUB_TOKEN`.
   *(Stop point 4: adding an Actions secret is a human's.)*

4. In `.goreleaser.yaml`, change `homebrew_casks[0].skip_upload` from `true` to
   `false`. Until that flip, tagging a release writes the cask into `dist/` and
   pushes nothing, which is deliberate: a tag should never be the thing that
   publishes to a package manager by surprise.

## Verifying it before anyone else does

After the first release that uploads a cask:

```sh
brew tap spelingbee/tap
brew install spelingbee/tap/restored
restored version
brew uninstall restored && brew untap spelingbee/tap
```

To check the generated cask *without* publishing anything, run a snapshot release
and read what it produced:

```sh
goreleaser release --snapshot --clean
cat dist/homebrew/Casks/restored.rb
```

## What the cask contains, and why

- `binaries: [restored]` - the one binary in the archive.
- A `caveats` block naming Docker and restic. Neither is a `dependencies:` entry on
  purpose: `restored version`, `recipe validate` and `recipe show` all work with
  neither installed, and when a command does need one it says which. Declaring a
  hard dependency would install a second Docker on a machine that already runs
  colima.
- A post-install `hooks` block that clears `com.apple.quarantine`. Without it the
  first run of an unsigned, non-browser-downloaded binary dies with "cannot be opened
  because the developer cannot be verified", which reads as *restored is broken*
  rather than as *restored is unsigned*. It is unsigned; notarisation costs an Apple
  Developer account, which is stop point 5.
