# Installation

The native installers support Linux, macOS, and Windows release archives. The
CLI is useful on its own; the optional Skill teaches supported AI coding hosts
when and how to use that CLI.

## Binary-only install

Linux or macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/yangkushu/ai-session-history/master/scripts/install.sh | sh
```

PowerShell:

```powershell
irm https://raw.githubusercontent.com/yangkushu/ai-session-history/master/scripts/install.ps1 | iex
```

This is the default. It installs `ai-history` without changing any Agent Skill.

## Binary and Skill install

Linux or macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/yangkushu/ai-session-history/master/scripts/install.sh | sh -s -- --with-skill
```

PowerShell:

```powershell
$script = irm https://raw.githubusercontent.com/yangkushu/ai-session-history/master/scripts/install.ps1
& ([scriptblock]::Create($script)) -WithSkill
```

The bundle first installs and verifies the binary, then installs the optional
Skill for detected or explicitly selected hosts. Skill installation requires
Node.js and `npx`. Before running the bundle, review the repository source and
[`skills/ai-history/SKILL.md`](../skills/ai-history/SKILL.md). `npx` downloads
and executes the linked third-party `skills` package. Node.js and `npx` are used
only for Skill installation; the Go CLI runtime has no Node.js dependency.

## Version, install directory, and PATH options

Without a version option, the installer resolves the latest release. Pin a
release, select a destination, and leave shell configuration unchanged when
needed:

```sh
curl -fsSL https://raw.githubusercontent.com/yangkushu/ai-session-history/master/scripts/install.sh | sh -s -- --version v1.2.3 --install-dir /opt/ai-history/bin --no-modify-path
```

```powershell
$script = irm https://raw.githubusercontent.com/yangkushu/ai-session-history/master/scripts/install.ps1
& ([scriptblock]::Create($script)) -Version v1.2.3 -InstallDir C:\Tools\ai-history -NoModifyPath
```

The Unix options are `--version`, `--install-dir`, and `--no-modify-path`.
PowerShell uses `-Version`, `-InstallDir`, and `-NoModifyPath`. If PATH changes
are enabled, start a new shell after installation.

## Repeat-to-update behavior

Rerunning a binary-only command updates only the binary; it does not install or
refresh Skills. Rerunning a bundle command refreshes the binary and the Skills
for the selected hosts. A pinned version can also be used for an intentional
downgrade. Repeating an already current binary-only install is safe.

## Agent detection and explicit targets

With `--with-skill` or `-WithSkill`, the installer detects supported Codex,
Claude Code, and Cursor hosts. If detection is incomplete or you want a precise
set, provide targets explicitly:

```sh
curl -fsSL https://raw.githubusercontent.com/yangkushu/ai-session-history/master/scripts/install.sh | sh -s -- --with-skill --agent codex --agent claude-code --agent cursor
```

```powershell
$script = irm https://raw.githubusercontent.com/yangkushu/ai-session-history/master/scripts/install.ps1
& ([scriptblock]::Create($script)) -WithSkill -Agent codex,claude-code,cursor
```

If no supported host is detected and no target is supplied, Skill installation
stops and reports how to select a target. The verified binary remains installed.
Installing a Skill does not grant permission to execute the CLI, read history,
or write exports. It also does not change the host sandbox, allowlist, or
managed policy. Runtime authorization remains under the selected host and
user's control. Codex uses `$ai-history` to invoke the Skill. For Claude Code
and Cursor, use the Skill invocation displayed by the current host UI.

## Releases, source, and manual fallbacks

If the native installer is unsuitable, download the matching archive and
`checksums.txt` from [GitHub Releases](https://github.com/yangkushu/ai-session-history/releases),
verify it, and place the binary on PATH. You can instead clone the repository
and build `./cmd/ai-history` with Go 1.22 or later.

For a manual Skill fallback, use the same canonical `skills/ai-history/` source
of truth and copy that directory in full to each intended host target:

- `$HOME/.agents/skills/ai-history`
- `$HOME/.claude/skills/ai-history`
- `$HOME/.cursor/skills/ai-history`

Review the host documentation before choosing a global or project target. Do
not maintain a separately edited Skill copy, and use the same installation
method for later updates.

## Remote-script review and checksum trust boundary

A piped remote script executes code from the network. Before running it, fetch
the canonical `scripts/install.sh` or `scripts/install.ps1` URL shown above,
inspect the content, and run only the revision you trust. For stronger
repeatability, download a reviewed script revision before executing it.

The installer verifies the selected archive against the release
`checksums.txt`. This detects corruption or a mismatched artifact; it does not
protect against a compromised repository, release publisher, hosting account,
or both the archive and checksum being replaced by the same attacker. The
remote installer and release publisher remain part of the trust boundary.

## Partial-failure recovery

The binary is installed and checked before Skill installation begins. If `npx`
is missing or one Skill target fails, the installer reports a partial failure
and preserves the verified binary and any successful Skill installs. Correct
the reported dependency or target issue, then rerun the bundle command with the
intended explicit targets. A binary-only rerun will not repair or refresh a
Skill.

If download, checksum, extraction, or binary verification fails, the installer
does not replace a known working binary. Resolve the network or release issue
and rerun the same command.

## PATH conflicts

The installer warns when another `ai-history` executable is already earlier on
PATH. Check which executable your shell resolves, then remove the stale entry or
reorder PATH. Use the no-modify-PATH option when PATH is managed centrally.
Do not delete an existing executable until the newly installed one passes the
verification below.

## Verification

Open a new shell when PATH was changed, then run:

```sh
ai-history version
ai-history doctor --json
```

`version` confirms which release is running. `doctor --json` checks source
discovery and reports access or configuration issues without granting new
permissions.

## Uninstall

There is no uninstall command. Remove the `ai-history` executable from the
chosen install directory. On Unix, remove the managed profile line marked
`# ai-history installer`; on Windows, remove the actual install directory entry
from the user PATH. Then start a new shell. If you installed the Skill bundle,
remove `ai-history` from each selected host's Skill directory as well. Remove
only paths confirmed to belong to this installation; exported session files
and source history are not deleted automatically.
