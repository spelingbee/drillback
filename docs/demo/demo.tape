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

Type "./scripts/demo.sh"
Enter

# demo.sh stands up Gitea and PostgreSQL, seeds them, dumps and backs up with
# restic, destroys the stack, and asks restored. Ten minutes is generous; the
# @-timeout is a ceiling, not a delay - vhs continues as soon as the regex matches.
Wait+Screen@10m /This backup boots|RESTORE UNUSABLE|restored exited/

Sleep 4s

Type "echo $?"
Enter
Sleep 2s

Type "clear"
Enter
Sleep 500ms

# --- Act 2: a backup that does not. ---------------------------------------------
Type "# Now the same drill, on a backup that is missing its database dump."
Enter
Sleep 1500ms

Type "./scripts/demo-broken.sh"
Enter
Wait+Screen@10m /restored exited|This backup boots/

Sleep 5s

Type "echo $?"
Enter
Sleep 3s
