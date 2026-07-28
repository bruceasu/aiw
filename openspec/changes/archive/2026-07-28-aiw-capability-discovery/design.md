# Design: AIW Capability Discovery

AIW uses four layers:

1. Core workflow: tasks, specs, worktrees, registry, context, and archive behavior.
2. Capability layer: Skills and their installation/discovery surface.
3. AI support layer: flow automation, session inspection, and interactive loop behavior.
4. Plugin layer: external commands and their runtime discovery metadata.

This change covers discovery for the capability, AI support, and plugin layers. The runtime must resolve plugin paths at execution time. Metadata should include name, description, invocation, source path, readOnly, mutatesFiles, requiresConfirmation, and outputFormat.

Existing META dictionaries remain valid. Missing fields receive conservative defaults: mutatesFiles=true for unknown commands, requiresConfirmation=true for unknown mutating commands, and outputFormat=text.

Templates should instruct AI to inspect runtime metadata before unfamiliar or mutating operations and to prefer aiw file and aiw patch for file changes.
