# History Rewrite and Next Step - 2026-07-08

## Summary

The repository history was rewritten to remove local machine-specific paths
from committed documentation. The current public branch tips are:

- `master`: `e36c1925169b4c9a8ac2895acdf78d0b2d545870`
- `bootstrap-ai-session-history-cli`: `a5cf97f94e8fe8ef30f930f9bee0f5e85c8d33af`

A pre-rewrite backup bundle was created at:

- `/tmp/ai-session-history-pre-rewrite-20260708.bundle`

## Verification

The rewritten local history was checked with:

- a working-tree scan for local user names, local home paths, old workspace
  paths, and machine-specific Go installation paths;
- a full-history pickaxe scan for the local home path;
- a full-history pickaxe scan for the local user name;
- `git cat-file -e 74f1c38d9253904105c32eca1437fd22a4d2661b`
- `git ls-remote origin refs/heads/master refs/heads/bootstrap-ai-session-history-cli`

The first three checks returned no matches, the old commit object was not
available locally after pruning, and the remote branch tips matched the rewritten
history.

## Current Development State

OpenSpec change `bootstrap-ai-session-history-cli` remains open. The remaining
implementation work is Cursor latest macOS validation:

- capture a minimized macOS Cursor fixture from a real local sample;
- add macOS fixture tests;
- implement or confirm the macOS reader behavior against the real storage shape.

This work is blocked in the current environment because no macOS Cursor sample
is available. The change must not be archived until that validation is complete.
