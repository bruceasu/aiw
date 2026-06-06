#!/usr/bin/env sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(CDPATH= cd -- "$script_dir/../.." && pwd)
out_dir="$repo_dir/plugins/aiw-cz"

mkdir -p "$out_dir"
cd "$script_dir"
go build -o "$out_dir/cz" .
