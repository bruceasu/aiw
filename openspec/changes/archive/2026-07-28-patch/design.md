# Design: Cross-platform Git Patch Application

## Boundary

aiw patch owns input transport, AI patch syntax conversion, and user-facing patch semantics as an AI support capability. Git owns unified diff parsing, path validation, context matching, reverse application, and three-way merge behavior.

## AI Invocation

AI code-editing workflows and AI support surfaces SHALL call the patch adapter for every generated patch. Direct file writes are not a substitute for patch application when the AI has produced a patch. The adapter returns a machine-readable result containing status, Git exit code, diagnostics, and affected paths.

## Flow

patch file or stdin -> detect encoding -> recognize AI patch syntax -> convert to unified diff -> write an ephemeral patch file -> git apply --check -> git apply or git apply -R -> remove the temporary file

## Command Surface

aiw patch check [--encoding name] patch
aiw patch apply [--3way] [--index] [--encoding name] patch
aiw patch reverse [--encoding name] patch

A patch path of - means standard input. The detector handles UTF-8, UTF-8 BOM, and UTF-16. --encoding is available for ambiguous legacy input. The converter supports Begin Patch, Update File, Add File, Delete File, and Move to File operations when their line content can be represented as a standard unified diff. Conversion errors must identify the source operation and file path.

Arguments must be passed as a list to the subprocess. Shell interpolation is forbidden. Temporary files must be removed on success and failure. AI callers must stop the current workflow when validation or application fails, preserve Git diagnostics, and either revise the patch or ask for user direction.
