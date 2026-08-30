# The README GIF, recorded from a real run.
#
# This tape runs scripts/demo.sh and scripts/demo-broken.sh. It does not simulate
# them, it does not replay a transcript, and nothing in the resulting GIF is typed by
# a human: what you see is what the tool printed on the machine that rendered it.
# That is the rule in CLAUDE.md, and a GIF is the easiest place in a project to break
# it.
#
# Render it with:
#   ./scripts/render-demo.sh
# which needs vhs, ttyd, ffmpeg, a docker daemon and restic, and takes a few minutes
# because it stands up two real application stacks and restores two real backups.
#
# The two acts are timed rather than watched for a verdict; the comments at each
# Sleep say why, and what to do if a run outgrows its budget.
#
# Requirements this tape assumes, all of which render-demo.sh checks first:
#   - the working directory is the repository root
#   - bin/restored is built
#   - `docker info` succeeds
#   - `restic version` succeeds

Output "docs/demo/demo.gif"

Require bash
Require docker
Require restic

Set Shell "bash"
Set FontSize 14
Set Width 1000
Set Height 640
Set Padding 24
Set TypingSpeed 45ms
# Two real stack-ups take about four minutes of wall clock, and four minutes of GIF is
# tens of megabytes nobody will wait for. This is a real recording played at double
# speed - the README says so underneath it. Nothing is cut.
Set PlaybackSpeed 2.0
Set Framerate 8
Set LoopOffset 0
Set MarginFill "#12151c"
Set Margin 14
Set BorderRadius 8

# --- Act 1: a backup that restores. ---------------------------------------------
Type "# Does this backup actually restore? Let's find out the only honest way."
Enter
Sleep 1500ms

# The `; echo` is not decoration. Wait+Screen matches against what is currently on
# the screen, and the tail of a long run scrolls the verdict off it - so waiting for
# the word PASS is waiting for something that may already be gone. The exit code is
# printed last, is the thing a cron job actually reads, and cannot scroll away before
# vhs sees it, because nothing follows it.
Type `./scripts/demo.sh; echo "exit=$?"`
Enter

# A fixed budget rather than Wait+Screen. Wait is the obvious tool for this and it
# does not work here: it reported `timeout waiting for "Screen exit=[0-9]"` with a
# last-seen screen containing only the two lines that had been typed, having never
# observed a single line of the run's output - while the recorded frames show that
# output arriving normally. Rather than ship a tape that depends on behaviour I cannot
# explain, both acts wait out a measured budget.
#
# demo.sh stands up Gitea and PostgreSQL, seeds them, dumps and backs up with restic,
# destroys the stack, and asks restored. Measured at 49s on the machine that rendered
# the committed GIF; 110 is that with room. If it ever runs longer, the GIF ends
# mid-drill and the fix is to raise the number - which is visible immediately, because
# the last thing on screen will not be a verdict.
Sleep 110s

Type "clear"
Enter
Sleep 500ms

# --- Act 2: a backup that does not. ---------------------------------------------
Type "# Now the same drill, on a backup that is missing its database dump."
Enter
Sleep 1500ms

Type `./scripts/demo-broken.sh; echo "exit=$?"`
Enter

# demo-broken.sh is the slower of the two: it stands the stack up twice, because the
# restore it is asked to judge has to fail its checks rather than fail to start.
# Measured at about two minutes.
Sleep 170s
