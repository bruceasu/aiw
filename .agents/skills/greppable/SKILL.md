---
name: greppable
description: Find and improve code paths using domain terms.
disable-model-invocation: true
---

# Greppable

Use this reference when the user invokes `greppable` for a code review, design,
refactor, or implementation. A code path is greppable when a search for its
domain term leads to its owner, production use, contract, and available
verification evidence without first reconstructing the repository.

Apply the rules to the concepts in the requested scope. Do not turn a focused
change into a repository-wide naming or module cleanup. This skill does not
start an implementation, create tests, or authorize runtime checks on its own.

## Search addresses

- Use the repository's established term for each concept in paths, symbols,
  operational strings, and nearby documentation. Record a non-obvious canonical
  term in repository guidance when future searches would otherwise miss it.
- Give a public symbol enough domain context to distinguish it in its effective
  namespace. A qualified name may already provide that context; a generic
  method such as `process` usually needs an object. Stop adding words once the
  search result is clear.
- Keep names true to behavior. When a behavior changes meaning, update its
  name and callers in the same authorized change. Avoid aliases that give one
  concept unrelated names.
- Put each behavior at one searchable owner. Remove a replaced definition, or
  mark a retained compatibility path at its definition and point to the owner.
- Use domain terms in filenames and explicit exports when a generic path or
  wildcard re-export hides the owner.
- Keep externally visible event names, flags, and error codes as complete
  literals. Give diagnostic messages a stable literal phrase that can be
  searched from an observed value.

## Contracts and wiring

- Make the boundary readable from its names, signatures, and nearby notes.
  State a constraint at the definition when the type cannot express it, such
  as units, timezone, ordering, authority, or source of truth. Add the ordinary
  phrase a reader would search for if the symbol uses compressed naming.
- Represent distinct identities and states explicitly when mixing them would
  permit a real error. Validate untrusted input at its entry boundary and pass
  the resulting domain value to downstream code.
- Keep a cohesive concept at one named boundary. Split a file when unrelated
  domain questions make search results hard to interpret; keep private helpers
  with their owner. File length alone does not decide the boundary.
- Give a feature a visible composition site that connects its parts to a
  production entry point. Keep side effects and dependency direction clear
  enough to trace the call path from that site.

## Evidence and dead ends

- Use the same domain term in available tests, fixtures, and verification notes
  so a search can connect them to production code. Follow repository test rules;
  greppability does not require adding or running a test.
- Record an intentional absence at the boundary where a reader would look for
  the behavior. State the reason when it affects use or safety.
- When code moves, trace callers to the new owner. A stale implementation or
  unexplained deprecated path is a false search result.

## How to use this reference

Start with one domain phrase from the user's task. Search for the definition,
production wiring, contract, and available verification evidence. Follow only
the paths needed to explain the scoped behavior. Identify the search dead ends,
then propose or make the smallest authorized change that removes them.

Finish when each changed concept has a clear owner and production path, its
relevant contract can be found at the boundary, and any remaining search dead
end is reported with its location. Report the terms searched, files inspected,
findings or edits, and checks actually run. If evidence is missing, report
`%% NEEDS_INPUT: ...` and mark the result `INCOMPLETE` when the gap prevents a
valid conclusion. Read-only review changes no files; edits and validation
follow the user's request and repository authorization rules.
