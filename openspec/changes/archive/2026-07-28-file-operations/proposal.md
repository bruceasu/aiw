# Proposal: Shared AI File Operations

## Problem Statement

Skills that read and write files through console commands can mis-handle UTF-8 BOM, UTF-16, GB18030, Windows-31J, or line endings on Windows. Different Skills may also overwrite files non-atomically.

## Solution

Provide aiw file read, info, and write commands with shared encoding detection, newline preservation, and atomic writes. Make these commands the default file access path for AI Skills. Keep search, Git inspection, tests, and builds on their existing tools.

## User Stories

1. As an AI Skill, I want to read text with detected encoding, so that context is not corrupted.
2. As an AI Skill, I want to write text while preserving encoding and line endings, so that existing files remain stable.
3. As a developer, I want file metadata and confidence, so that ambiguous encodings are visible.
4. As a developer, I want failed writes to leave the original file unchanged, so that uncertain detection is safe.

## Implementation Decisions

- Support UTF-8, UTF-16, GB18030, and Windows-31J.
- Prefer deterministic BOM and strict UTF-8 detection.
- Require explicit encoding when legacy encodings are ambiguous.
- Preserve BOM and newline style by default.
- Use temporary-file plus atomic replacement for writes.
- Make AI Skills prefer aiw file for file content access and aiw patch for code changes.

## Testing Decisions

- Test encoding detection, confidence, BOM, newline preservation, atomic writes, missing files, and ambiguous legacy input.
- Use temporary directories and external behavior assertions.
- Keep command integration tests separate from codec unit tests.

## Out of Scope

- Binary file editing.
- Automatic conversion of an entire repository.
- Replacing rg, Git, test runners, or build tools.
- Inferring arbitrary historical code pages without an explicit option.

## Further Notes

%% Legacy encoding detection can be ambiguous for short or mostly-ASCII files.