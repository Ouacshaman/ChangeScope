# AGENTS.md

This file provides persistent project guidance for IBM Bob and other agents working in this repository.

## Project

- **Name:** ChangeScope
- **Language:** Go
- **Module:** `github.com/Ouacshaman/changescope`
- **Go version:** 1.27.1
- **Status:** Early-stage hackathon prototype
- **Primary interface:** CLI

## Purpose

ChangeScope is a developer workflow tool built for the IBM Bob 2.0 Hackathon.

Its goal is to help developers understand the potential blast radius of a code change before the change is merged or released.

Developers usually know which files they changed, but they may not know which other functions, packages, tests, APIs, database logic, configuration, or documentation could be affected.

ChangeScope should help answer:

> "What else could this change affect?"

## MVP

The minimum viable product should:

1. Analyze changes in a local Git repository.
2. Identify changed Go files.
3. Determine which Go functions or methods were changed.
4. Find references to those changed symbols elsewhere in the repository.
5. Identify potentially affected packages.
6. Identify related `_test.go` files and tests.
7. Produce a clear terminal impact report.

Example:

```text
ChangeScope Impact Report

Changed:
  internal/task/service.go

Changed symbols:
  UpdateTask

Potential impact:
  internal/http/task_handler.go
  internal/task/service_test.go

Related tests:
  TestUpdateTask

Suggested review:
  - Review callers of UpdateTask
  - Run task package tests
  - Verify API behavior