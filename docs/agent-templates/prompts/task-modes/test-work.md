# Test Work

## Goal
- test behavior, not implementation details
- protect the public contract or failure mode that matters

## Design
- keep tests deterministic
- prefer small fixtures and focused setup
- create a failing test first when practical

## Coverage
- cover the changed path
- cover important edge cases and error handling
- avoid broad golden outputs when a smaller assertion is enough

## Report
- state what behavior is now protected
- state what still is not covered
