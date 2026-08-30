# Mealie - no issue drafted

**Status: nothing to file.**

Mealie's backup page recommends stopping the container and copying `/app/data`, calls
that the best way in bold, and it is: the restore brings back the accounts, the recipes,
and the `.secret` file that signs sessions. There is no gap to report.

One thing is recorded in [result.md](result.md) as a limitation of this drill rather than
of Mealie: the integrated backup feature was not tested, because restoring it means
uploading a zip through an authenticated admin session into a running instance, and this
drill's harness cannot express "restore after the application is up, then check". That
is a gap in the result, not a finding.
