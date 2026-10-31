# ASUS IME Mode

Get or force the Chinese/English mode of ASUS Bopomofo (華碩注音) in the foreground window. ASUS Bopomofo only has a hotkey to toggle the mode; with this program a hotkey can set it to Chinese or to English.

[繁體中文 README](./README.md)

## How It Works

ASUS Bopomofo ignores the IMM conversion mode that most tools (e.g. AutoHotkey `IME_SetConvMode`, `im-select`) rely on. Instead:

- Its real mode is written to `%ProgramFiles%\ASUS\AsusIME\zhtw\data\%UserName%\AsusIMEConfig.ini` on every switch, under `[DirectOutputMode]`, keyed by process name (`0` = Chinese, `1` = English). Processes without an entry start in Chinese mode.
- Any change of the IMM open status (`WM_IME_CONTROL` / `IMC_SETOPENSTATUS`) flips the mode, regardless of the value written.

So the program reads the mode of the foreground process from the ini, and flips the open status only when the mode differs from the requested one. No key presses are simulated, so held modifier keys of the triggering hotkey don't interfere.

All Chinese input methods share the same keyboard layout language, so the active one can't be identified from outside the target process. The program refuses to run when any Chinese input method other than ASUS Bopomofo is enabled.

## Requirements

- Windows 10 or later with ASUS Bopomofo as the only enabled Chinese input method
- Hotkey use needs Windows 11 24H2 (build 26100) or later. On older Windows a hotkey launch opens a console window that takes the focus, so the console window gets switched instead (see Limitations). Shells and scripts work on any version.

## Installation

Download the zip for your architecture from GitHub Releases: `windows_amd64` for most PCs, `windows_arm64` for ARM laptops (e.g. Snapdragon). If unsure, run `$env:PROCESSOR_ARCHITECTURE` in PowerShell: it prints `AMD64` or `ARM64`.

Extract the zip to `%LOCALAPPDATA%\Programs\asus-ime-mode`, the usual location for per-user programs on Windows, which needs no administrator rights. The zip holds a single program, `asusimemode.exe`, for shells, scripts and hotkeys alike. Its manifest sets [`consoleAllocationPolicy`](https://learn.microsoft.com/windows/console/console-allocation-policy), so a hotkey launch opens no console window and the focus stays on the window to switch, while a shell runs it like any command-line program, with its output and exit code.

To run `asusimemode` by name in a terminal, add the folder to your user PATH environment variable. If you already have `%UserProfile%\.local\bin` on PATH (e.g. from installing uv or Claude Code), you can put the program there instead.

To update, extract the new version to the same folder, overwriting the old files. Hotkeys store the full path, so they keep working as long as the location stays the same.

Updating from v0.1.0: v0.1.0 also shipped `asusimemodew.exe` for hotkeys, which later versions no longer include. Point your hotkeys at `asusimemode.exe`, then delete the old `asusimemodew.exe`.

## Usage

```
asusimemode [zh|en|toggle]
```

- No argument: print the current mode, `zh` or `en`.
- `zh`: force Chinese mode.
- `en`: force English mode.
- `toggle`: toggle the mode.

The resulting mode is printed after switching.

Exit codes:

| Code | Meaning |
| ---- | ------- |
| 0 | Success |
| 2 | Invalid argument |
| 3 | No IME window for the foreground window |
| 4 | The active keyboard is not a Chinese input method (e.g. the US keyboard) |
| 5 | The mode did not change within 500 ms, usually because the focused element has no input context (see Limitations) |
| 6 | Unsupported environment: another Chinese input method is enabled, ASUS Bopomofo is not enabled, or its state can't be read |

### Hotkeys with PowerToys

In PowerToys Keyboard Manager, select "Remap a shortcut", add a shortcut and choose the "Run program" action:

- App: `%LOCALAPPDATA%\Programs\asus-ime-mode\asusimemode.exe`
- Args: `zh` or `en`

## Limitations

- Switching needs an input context on the focused element. Text inputs always have one; whether other elements do depends on the program: some keep one even outside text inputs, while in others a button or the body of a web page has none, and switching there fails with exit code 5. Moving the focus to an element without an input context can also make ASUS Bopomofo switch to English by itself, so the mode read then doesn't tell the mode restored on returning to a text input.
- The mode is tracked per process by ASUS Bopomofo itself, so windows of the same program share one mode.
- It doesn't work on windows of elevated processes unless the caller is also elevated.
- It depends on undocumented behavior of ASUS Bopomofo and may break after an update.
- Windows before 11 24H2 doesn't know `consoleAllocationPolicy`, so a hotkey launch still opens a console window and the console gets switched instead of the original window. Don't use it from hotkeys there.
- Double-clicking `asusimemode.exe`, or starting it with PowerShell's `Start-Process` without `-NoNewWindow`, opens no window either, so its output isn't visible.

## Development

Requirements:

- Go 1.27 or later
- [Task](https://taskfile.dev): every command runs through `task`; `task --list` lists them
- [golangci-lint](https://golangci-lint.run) v2: `task fmt` to format, `task lint` to lint
- Node.js and pnpm: run `pnpm install` once to install cspell, [husky](https://typicode.github.io/husky/), commitlint and GoReleaser. The pre-commit hook checks spelling and, when Go files are staged, runs `golangci-lint run`; the commit-msg hook checks with commitlint that the commit message follows [Conventional Commits](https://www.conventionalcommits.org/) (Angular types; rules in `.commitlintrc.yaml`)

`task build` builds the program into `%UserProfile%\.local\bin` (override with `task build BIN_DIR=<dir>`). Builds are skipped when the sources haven't changed; add `--force` to rebuild anyway.

The manifest is `winres/asusimemode.manifest`. Go's linker can't embed a manifest itself, so it is compiled into `rsrc_windows_amd64.syso` and `rsrc_windows_arm64.syso`, which are committed: `go build`, `go install` and GoReleaser link them in automatically, and building needs no other tool. After changing the manifest, run `task winres` to regenerate the `.syso` files; that step needs the C++ workload of Visual Studio Build Tools (it uses `rc.exe` and `cvtres.exe`). `task winres:check` checks that the committed `.syso` files match the manifest, and the pre-commit hook runs it when `winres/` or a `.syso` file is staged.

### Releasing

Branches follow GitHub Flow (configured in `.gitflow` for [git-flow-next](https://git-flow.sh)): only `main` and `feature/*`, no release branches; a release is a tag on `main`.

1. On `main`, run `task release VERSION=0.1.0` ([SemVer](https://semver.org), without a leading `v`). It first checks that:
   - you are on `main`, the working tree is clean, and `main` matches `origin/main`
   - the tag `v0.1.0` doesn't exist yet, locally or on `origin`

   It then runs the linters, the spell check and `task release:snapshot`, and only if all pass creates the signed tag `v0.1.0` and pushes it.
2. On the pushed tag, GitHub Actions checks again (the tag is on `main`, lint, spelling), then builds with GoReleaser and creates a GitHub release with one zip for amd64, one for arm64, and `checksums.txt`.

To check the release contents without publishing, run `task release:snapshot`; the output goes to `dist/`.

## License

[MIT](./LICENSE)
