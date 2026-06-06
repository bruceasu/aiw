#!/usr/bin/env python3
"""General plugin usage doc generator for all aiw plugins.

Scans a plugins directory for `aiw-*.py`, imports each module and
extracts a `META` dict to generate a Markdown usage document. If the
target doc already exists it is skipped and logged.

Options:
  -d, --dir    plugins directory (default: plugins)
  -n, --name   plugin name or substring to match (optional)
  -o, --output output file or directory, '-' for console (optional)

This is a moved and generalized replacement for the previous
`plugins/generate_git_docs.py` script.
"""
import os
import glob
import importlib.util
import argparse
import sys


def find_plugins(plugins_dir, name_filter=None):
    pattern = os.path.join(plugins_dir, 'aiw-*.py')
    matches = sorted(glob.glob(pattern))
    out = []
    for p in matches:
        base = os.path.basename(p)
        if base in ('aiw-git-core.py', 'generate_git_docs.py'):
            # skip old helpers or generator
            continue
        if name_filter:
            if name_filter in base or name_filter in base.replace('.py', ''):
                out.append(p)
        else:
            out.append(p)
    return out


def load_meta(plugin_path):
    try:
        name = os.path.splitext(os.path.basename(plugin_path))[0]
        spec = importlib.util.spec_from_file_location('mod_'+name, plugin_path)
        m = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(m)
        return getattr(m, 'META', None)
    except Exception as e:
        print('failed to import', plugin_path, e, file=sys.stderr)
        return None


def generate_from_meta(meta, out_path, core_module=None):
    if core_module and hasattr(core_module, 'generate_md_from_meta'):
        core_module.generate_md_from_meta(meta, out_path)
    else:
        # fallback simple renderer
        with open(out_path, 'w', encoding='utf-8') as f:
            f.write(f"# {meta.get('name', '')}\n\n")
            f.write(meta.get('short', '') + "\n\n")
            f.write('## Usage\n\n')
            f.write(meta.get('usage', '') + "\n\n")
            if meta.get('long'):
                f.write('## Description\n\n')
                f.write(meta.get('long') + "\n\n")
            if meta.get('examples'):
                f.write('## Examples\n\n')
                for ex in meta.get('examples'):
                    f.write(f'- `{ex}`\n')


def main(argv=None):
    parser = argparse.ArgumentParser(description='Generate plugin usage docs from META')
    parser.add_argument('-d', '--dir', default=os.path.join(os.path.dirname(__file__), '..', 'plugins'), help='plugins directory')
    parser.add_argument('-n', '--name', help='plugin name or substring to match')
    parser.add_argument('-o', '--output', help="output file or directory, '-' for console")
    args = parser.parse_args(argv)

    plugins_dir = os.path.abspath(args.dir)
    if not os.path.isdir(plugins_dir):
        print('plugins dir not found:', plugins_dir, file=sys.stderr)
        return 2

    # try to load core generator if present
    core = None
    core_path = os.path.join(plugins_dir, 'aiw-git-core.py')
    if os.path.exists(core_path):
        try:
            cspec = importlib.util.spec_from_file_location('aiw_git_core', core_path)
            core = importlib.util.module_from_spec(cspec)
            cspec.loader.exec_module(core)
        except Exception as e:
            print('warning: failed to load core generator:', e, file=sys.stderr)

    # default docs dir (when output not provided or is directory)
    default_docs_dir = os.path.normpath(os.path.join(plugins_dir, '..', 'docs', 'usage'))
    os.makedirs(default_docs_dir, exist_ok=True)

    plugins = find_plugins(plugins_dir, args.name)
    if not plugins:
        print('no plugins found', file=sys.stderr)
        return 0

    multiple = len(plugins) > 1
    for p in plugins:
        base = os.path.basename(p)
        short = base[:-3]
        meta = load_meta(p)
        if not meta:
            print('no META in', base)
            continue

        # determine output
        if args.output == '-':
            # render to console
            if core and hasattr(core, 'generate_md_from_meta'):
                core.generate_md_from_meta(meta, '-')
            else:
                # simple console render
                print('#', meta.get('name', ''))
                print(meta.get('short', ''))
                print('\nUsage:\n', meta.get('usage', ''))
            print('---')
            continue

        out_path = None
        if args.output:
            out_arg = args.output
            if os.path.isdir(out_arg) or out_arg.endswith(os.sep):
                out_dir = out_arg
                os.makedirs(out_dir, exist_ok=True)
                out_path = os.path.join(out_dir, f'{short}.md')
            else:
                if multiple:
                    out_dir = out_arg
                    os.makedirs(out_dir, exist_ok=True)
                    out_path = os.path.join(out_dir, f'{short}.md')
                else:
                    out_path = out_arg
        else:
            out_path = os.path.join(default_docs_dir, f'{short}.md')

        # skip if exists
        if os.path.exists(out_path):
            print('skip existing', out_path)
            continue

        try:
            if core and hasattr(core, 'generate_md_from_meta'):
                core.generate_md_from_meta(meta, out_path)
            else:
                generate_from_meta(meta, out_path, core_module=None)
            print('wrote', out_path)
        except Exception as e:
            print('failed', base, e, file=sys.stderr)

    return 0


if __name__ == '__main__':
    rc = main()
    sys.exit(rc)
