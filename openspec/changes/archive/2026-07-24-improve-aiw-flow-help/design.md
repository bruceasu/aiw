## Context

`build_parser()` currently creates the complete aiw-flow argparse tree with minimal metadata. Generated HELP therefore shows syntax but omits command purpose, option meaning, workflow order, examples, and important constraints such as placing `--root` before the command. The README already contains this knowledge, but terminal HELP is the first and often only interface users consult.

The change must remain compatible with Python 3.9 and preserve the exact argparse destinations, required flags, defaults, and dispatch behavior.

## Goals / Non-Goals

**Goals:**

- Make top-level HELP useful as a command map and first-run guide.
- Make every command's HELP sufficient to construct a valid invocation.
- Keep HELP concise by presenting overview information at the top level and details at the selected command.
- Use Easy English, consistent Session terminology, explicit metavariables, and realistic examples.
- Verify that improved metadata does not change parsed namespaces.

**Non-Goals:**

- Change any command name, option, default, validation, or runtime behavior.
- Replace the README or duplicate all operational documentation in HELP.
- Add shell completion, interactive prompts, localization, or a documentation generator.
- Add aliases or deprecate one-shot commands.

## Decisions

1. Use argparse's `RawDescriptionHelpFormatter`.
   - Descriptions and epilogs can contain readable workflow diagrams and multiline examples.
   - A custom formatter was rejected because standard argparse behavior is stable and familiar.

2. Add a small `_add_command()` construction helper.
   - The helper applies a summary, detailed description, examples, and the common formatter consistently.
   - It only configures parser metadata and does not abstract argument definitions or dispatch.

3. Keep HELP layered.
   - Top-level HELP explains the product, the common lifecycle, quick starts, command summaries, and how to request more detail.
   - Command HELP explains the selected action, every argument, relevant constraints, and two or three examples.
   - Parent commands for `memory`, `handoff`, and `daemon` summarize their child actions; child HELP contains action-specific examples.

4. Treat examples as tested interface documentation.
   - Tests assert representative examples, summaries, argument descriptions, and nested navigation.
   - Parser-compatibility tests compare important parsed values before and after metadata changes rather than executing commands.

5. Keep HELP in Easy English.
   - The CLI and existing runtime messages are English, and simple wording is portable across terminals.
   - Localization is out of scope.

## Risks / Trade-offs

- [Risk] Top-level output becomes visually dense. → Keep only the lifecycle, two quick starts, and one-line command summaries at the top level.
- [Risk] Examples drift from accepted syntax. → Parse representative example argument lists in tests.
- [Risk] Refactoring parser construction changes a default or destination accidentally. → Leave all `add_argument` calls and dispatch logic intact, and test representative namespaces.
- [Trade-off] HELP repeats a small amount of README content. → Prefer terminal discoverability while keeping detailed architecture and safety discussion in the README.
