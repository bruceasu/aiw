# AI File Operations

All Skills SHALL use aiw file read, aiw file info, and aiw file write for project text content. AI-generated code changes SHALL use aiw patch for patch application.

Use rg, git, and shell commands for search, repository inspection, tests, builds, and external commands. Do not use PowerShell redirection, echo, console output, or ad hoc scripts to write project files.

If the shared tools are unavailable or the file is binary, use a documented fallback and state why. Encoding is utf-8, gb18030, or windows-31j; preserve existing BOM and newline style by default.