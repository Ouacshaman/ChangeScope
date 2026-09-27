# ChangeScope

ChangeScope is a Go CLI tool that helps developers understand the potential blast radius of an unstaged code change before it is committed, reviewed, or released.

A Git diff shows which lines changed, but it does not always show what those changes may affect elsewhere in the codebase. ChangeScope analyzes the diff, identifies changed Go functions and methods, searches for potential references, finds related tests, and produces a concise impact report.

The core question ChangeScope is designed to answer is:

> What else could this change affect?

## Overview

When developers modify a function, method, or package, they often need to manually search through the repository to determine:

- Which functions or files depend on the changed code
- Which tests are related to the change
- Which packages may be affected
- Which integration points should be reviewed
- Whether additional testing may be required

ChangeScope automates the first stage of that investigation.

The current prototype focuses on Go repositories and unstaged Git changes.

## Features

- Detects unstaged changes using `git diff`
- Identifies changed Go files
- Parses changed line ranges from unified Git diff output
- Uses the Go AST to identify changed functions and methods
- Searches the repository for potential references to changed symbols
- Identifies related `_test.go` files
- Produces a readable terminal impact report
- Uses only the Go standard library
- Works as a lightweight CLI without requiring a web interface

## How It Works

ChangeScope uses a deterministic analysis pipeline:

```text
git diff
   |
   v
Changed Go files
   |
   v
Changed line ranges
   |
   v
Go AST analysis
   |
   v
Changed functions and methods
   |
   v
Potential reference search
   |
   v
Related tests
   |
   v
ChangeScope Impact Report
