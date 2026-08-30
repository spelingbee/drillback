# TEMPLATE

This directory is the skeleton to copy, not a recipe. It is not in the registry:
`drillback check --recipe TEMPLATE` will tell you there is no such thing.

    cp -r recipes/TEMPLATE recipes/myapp
    $EDITOR recipes/myapp/recipe.yaml recipes/myapp/compose.yaml
    drillback recipe validate ./recipes/myapp --strict
    drillback recipe test ./recipes/myapp

The full walk-through is [CONTRIBUTING.md](../../CONTRIBUTING.md), under
**Add a recipe in 10 minutes**.

If you already have a `docker-compose.yml` for the application, do not start here.
Start from what you have:

    drillback recipe init myapp --compose ~/docker/myapp/docker-compose.yml

That reads your file, turns its volumes into inputs, recognises a PostgreSQL or SQLite
service, takes the container side of your published port for the ready probe, and
leaves a TODO everywhere the answer is yours. The result validates as it comes out.

## What to write in a recipe README

Each recipe ships one, and it answers three questions for somebody who has never seen
the application's directory layout:

1. **Which of my directories is this input?** Name the two or three ways the
   application is commonly deployed - the upstream compose file, a named volume, a
   package install - and give the `--input name=path` for each.
2. **How do I produce the database dump?** The exact command, with the flags that
   matter and why.
3. **Which of these checks would fail if the backup were empty?** A table, one row per
   check. If the honest answer for every row is "no", the recipe is not finished.

A fourth section is worth writing when it applies: **what this recipe does not prove**.
A recipe that seeds through the database rather than the application's own API, or
that skips a slow part of the restore, should say so plainly. An admitted gap is worth
more than an unexamined claim.
