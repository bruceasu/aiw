#!/usr/bin/env bash
next-agent() {
  local temp_dir=$(python3 -c 'import tempfile; print(tempfile.gettempdir())')
  local latest_file=$(ls -t "${temp_dir}"/agent_handoff_*.md 2>/dev/null | head -n 1)
  if [ -n "$latest_file" ]; then
    cat "$latest_file"
  else
    echo "⚠️ No agent handoff document found in temporary directory."
  fi
}

next-agent "$@"