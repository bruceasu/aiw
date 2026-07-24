## 1. Python 3.9 Compatibility

- [x] 1.1 Remove Python 3.10-only dataclass options from the guide models while
  preserving their fields and defaults
- [x] 1.2 Extend dispatcher regression coverage to require discovery of the
  bundled guide subcommand
- [x] 1.3 Register dynamically loaded subcommands during execution so Python
  3.9 dataclasses can resolve postponed annotations

## 2. Verification

- [x] 2.1 Run the aiw-git dispatcher tests under Python 3.9
- [x] 2.2 Run the real aiw-git help/discovery path under Python 3.9
- [x] 2.3 Run repository-standard verification applicable to the change
