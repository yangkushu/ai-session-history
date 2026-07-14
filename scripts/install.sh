#!/bin/sh
set -eu

REPOSITORY="yangkushu/ai-session-history"
VERSION=${AI_HISTORY_VERSION:-}
INSTALL_DIR=${AI_HISTORY_INSTALL_DIR:-"$HOME/.local/bin"}
RELEASE_BASE_URL=${AI_HISTORY_RELEASE_BASE_URL:-"https://github.com/$REPOSITORY/releases/download"}
LATEST_RELEASE_URL=${AI_HISTORY_LATEST_RELEASE_URL:-"https://api.github.com/repos/$REPOSITORY/releases/latest"}
MODIFY_PATH=1
WITH_SKILL=0
AGENTS=""

usage() {
    cat <<'EOF'
Usage: install.sh [options]

Options:
  --version VALUE       Install a specific version (for example, v1.2.3)
  --install-dir DIR     Install ai-history into DIR
  --no-modify-path      Do not update the shell profile
  --with-skill          Install the ai-history Skill bundle
  --agent NAME          Install the Skill for NAME (repeatable)
  --help                Show this help
EOF
}

die() {
    printf 'Error: %s\n' "$*" >&2
    exit 1
}

argument_error() {
    printf 'Error: %s\n' "$*" >&2
    usage >&2
    exit 2
}

binary_identity_version() {
    BINARY_OUTPUT=$("$1" version 2>/dev/null) || return 1
    printf '%s\n' "$BINARY_OUTPUT" | awk '
        NR == 1 {
            if ($1 == "ai-history" && $2 ~ /^v[0-9]+\.[0-9]+\.[0-9]+$/) {
                print $2
                valid = 1
            }
            exit
        }
        END { if (!valid) exit 1 }
    '
}

json_process() {
    JSON_MODE=$1
    awk -v mode="$JSON_MODE" '
        function skip_whitespace(    c) {
            while (position <= length(json)) {
                c = substr(json, position, 1)
                if (c !~ /[ \t\r\n]/) break
                position++
            }
        }

        function hex_digit(c) {
            if (c ~ /^[0-9]$/) return c + 0
            return index("abcdef", tolower(c)) + 9
        }

        function decode_unicode(hex,    code, i) {
            code = 0
            for (i = 1; i <= 4; i++) code = code * 16 + hex_digit(substr(hex, i, 1))
            if (code >= 32 && code <= 126) return sprintf("%c", code)
            return "\\u" hex
        }

        function parse_string(    c, escaped, hex, i) {
            if (substr(json, position, 1) != "\"") return 0
            position++
            parsed_string = ""
            while (position <= length(json)) {
                c = substr(json, position, 1)
                if (c == "\"") {
                    position++
                    return 1
                }
                if (c ~ /[\001-\037]/) return 0
                if (c == "\\") {
                    position++
                    if (position > length(json)) return 0
                    escaped = substr(json, position, 1)
                    if (escaped == "u") {
                        hex = ""
                        for (i = 1; i <= 4; i++) {
                            position++
                            if (substr(json, position, 1) !~ /^[0-9A-Fa-f]$/) return 0
                            hex = hex substr(json, position, 1)
                        }
                        parsed_string = parsed_string decode_unicode(hex)
                    } else if (escaped != "\"" && escaped != "\\" && escaped != "/" &&
                               escaped != "b" && escaped != "f" && escaped != "n" &&
                               escaped != "r" && escaped != "t") {
                        return 0
                    } else if (escaped == "\"" || escaped == "\\" || escaped == "/") {
                        parsed_string = parsed_string escaped
                    } else {
                        parsed_string = parsed_string "\\" escaped
                    }
                } else {
                    parsed_string = parsed_string c
                }
                position++
            }
            return 0
        }

        function parse_number(    c) {
            if (substr(json, position, 1) == "-") position++
            c = substr(json, position, 1)
            if (c == "0") {
                position++
                if (substr(json, position, 1) ~ /^[0-9]$/) return 0
            } else if (c ~ /^[1-9]$/) {
                do {
                    position++
                    c = substr(json, position, 1)
                } while (c ~ /^[0-9]$/)
            } else {
                return 0
            }
            if (substr(json, position, 1) == ".") {
                position++
                if (substr(json, position, 1) !~ /^[0-9]$/) return 0
                while (substr(json, position, 1) ~ /^[0-9]$/) position++
            }
            c = substr(json, position, 1)
            if (c == "e" || c == "E") {
                position++
                c = substr(json, position, 1)
                if (c == "+" || c == "-") position++
                if (substr(json, position, 1) !~ /^[0-9]$/) return 0
                while (substr(json, position, 1) ~ /^[0-9]$/) position++
            }
            return 1
        }

        function parse_array(    c) {
            if (substr(json, position, 1) != "[") return 0
            position++
            skip_whitespace()
            if (substr(json, position, 1) == "]") {
                position++
                return 1
            }
            while (1) {
                if (!parse_value()) return 0
                skip_whitespace()
                c = substr(json, position, 1)
                if (c == "]") {
                    position++
                    return 1
                }
                if (c != ",") return 0
                position++
                skip_whitespace()
            }
        }

        function parse_object(top_level,    c, key) {
            if (substr(json, position, 1) != "{") return 0
            position++
            skip_whitespace()
            if (substr(json, position, 1) == "}") {
                position++
                return 1
            }
            while (1) {
                if (!parse_string()) return 0
                key = parsed_string
                skip_whitespace()
                if (substr(json, position, 1) != ":") return 0
                position++
                skip_whitespace()
                if (top_level && key == "tag_name") {
                    tag_count++
                    if (substr(json, position, 1) != "\"") {
                        tag_invalid = 1
                        if (!parse_value()) return 0
                    } else {
                        if (!parse_string()) return 0
                        tag_value = parsed_string
                    }
                } else if (!parse_value()) {
                    return 0
                }
                skip_whitespace()
                c = substr(json, position, 1)
                if (c == "}") {
                    position++
                    return 1
                }
                if (c != ",") return 0
                position++
                skip_whitespace()
            }
        }

        function parse_value(    c) {
            skip_whitespace()
            c = substr(json, position, 1)
            if (c == "\"") return parse_string()
            if (c == "{") return parse_object(0)
            if (c == "[") return parse_array()
            if (substr(json, position, 4) == "true") {
                position += 4
                return 1
            }
            if (substr(json, position, 5) == "false") {
                position += 5
                return 1
            }
            if (substr(json, position, 4) == "null") {
                position += 4
                return 1
            }
            if (c == "-" || c ~ /^[0-9]$/) return parse_number()
            return 0
        }

        { json = json $0 "\n" }
        END {
            position = 1
            skip_whitespace()
            if (mode == "array") {
                if (substr(json, position, 1) != "[") exit 1
                if (!parse_array()) exit 1
            } else if (mode == "tag") {
                if (substr(json, position, 1) != "{") exit 1
                if (!parse_object(1)) exit 1
            } else {
                exit 1
            }
            skip_whitespace()
            if (position <= length(json)) exit 1
            if (mode == "tag") {
                if (tag_count != 1 || tag_invalid) exit 1
                print tag_value
            }
        }
    '
}

update_profile() {
    if [ ! -f "$PROFILE" ]; then
        printf '\n%s\n' "$LINE" >> "$PROFILE"
        printf 'Updated PATH in %s\n' "$PROFILE"
        return
    fi

    MARKER_COUNT=$(grep -Fc '# ai-history installer' "$PROFILE" || true)
    if [ "$MARKER_COUNT" -eq 1 ] && grep -Fqx "$LINE" "$PROFILE"; then
        return
    fi
    if [ "$MARKER_COUNT" -eq 0 ]; then
        printf '\n%s\n' "$LINE" >> "$PROFILE"
        printf 'Updated PATH in %s\n' "$PROFILE"
        return
    fi

    PROFILE_TMP=$(mktemp "$PROFILE.ai-history.XXXXXX" 2>/dev/null) || die "could not create temporary profile beside $PROFILE"
    cp -p "$PROFILE" "$PROFILE_TMP" || die "could not preserve profile permissions for $PROFILE"
    if ! AI_HISTORY_MANAGED_LINE=$LINE awk '
        BEGIN { marker = "# ai-history installer"; line = ENVIRON["AI_HISTORY_MANAGED_LINE"] }
        index($0, marker) {
            if (!written) print line
            written = 1
            next
        }
        { print }
    ' "$PROFILE" > "$PROFILE_TMP"; then
        die "could not update PATH in $PROFILE"
    fi
    mv -f "$PROFILE_TMP" "$PROFILE"
    PROFILE_TMP=""
    printf 'Updated PATH in %s\n' "$PROFILE"
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        --version)
            [ "$#" -ge 2 ] || argument_error "--version requires a value"
            case "$2" in --*) argument_error "--version requires a value" ;; esac
            VERSION=$2
            shift 2
            ;;
        --install-dir)
            [ "$#" -ge 2 ] || argument_error "--install-dir requires a directory"
            case "$2" in --*) argument_error "--install-dir requires a directory" ;; esac
            INSTALL_DIR=$2
            shift 2
            ;;
        --no-modify-path)
            MODIFY_PATH=0
            shift
            ;;
        --with-skill)
            WITH_SKILL=1
            shift
            ;;
        --agent)
            [ "$#" -ge 2 ] || argument_error "--agent requires a name"
            case "$2" in --*) argument_error "--agent requires a name" ;; esac
            AGENTS="$AGENTS $2"
            shift 2
            ;;
        --help)
            usage
            exit 0
            ;;
        *)
            argument_error "unknown option: $1"
            ;;
    esac
done

RAW_OS=${AI_HISTORY_TEST_OS:-$(uname -s)}
RAW_ARCH=${AI_HISTORY_TEST_ARCH:-$(uname -m)}
case "$RAW_OS" in
    Linux|linux) OS=linux ;;
    Darwin|darwin) OS=darwin ;;
    *) die "unsupported operating system: $RAW_OS (supported: linux, darwin)" ;;
esac
case "$RAW_ARCH" in
    x86_64|amd64) ARCH=amd64 ;;
    arm64|aarch64) ARCH=arm64 ;;
    *) die "unsupported architecture: $RAW_ARCH (supported: amd64, arm64)" ;;
esac

TARGET=$INSTALL_DIR/ai-history
EXISTING_VERSION=""
if [ -e "$TARGET" ]; then
    [ -x "$TARGET" ] || die "existing target is not executable: $TARGET"
    EXISTING_VERSION=$(binary_identity_version "$TARGET") || die "existing target is not a recognized ai-history binary: $TARGET"
fi

if [ -z "$VERSION" ]; then
    command -v curl >/dev/null 2>&1 || die "required tool is unavailable: curl"
    TMP_LATEST=$(mktemp -d 2>/dev/null) || die "could not create temporary directory"
    trap 'rm -rf "$TMP_LATEST"' 0 HUP INT TERM
    LATEST_JSON=$TMP_LATEST/latest.json
    if ! curl -fsSL "$LATEST_RELEASE_URL" -o "$LATEST_JSON"; then
        die "could not resolve latest release from $LATEST_RELEASE_URL; use --version vX.Y.Z"
    fi
    VERSION=$(json_process tag < "$LATEST_JSON") || die "latest release response from $LATEST_RELEASE_URL has invalid or non-unique tag_name; use --version vX.Y.Z"
    [ -n "$VERSION" ] || die "latest release response from $LATEST_RELEASE_URL has no tag_name; use --version vX.Y.Z"
    rm -rf "$TMP_LATEST"
    trap - 0 HUP INT TERM
fi

if ! printf '%s\n' "$VERSION" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
    die "invalid version '$VERSION'; expected vX.Y.Z"
fi

SKIP_BINARY=0
if [ -n "$EXISTING_VERSION" ] && [ "$EXISTING_VERSION" = "$VERSION" ]; then
    printf 'ai-history %s is already installed at %s\n' "$VERSION" "$TARGET"
    SKIP_BINARY=1
fi

TMP_DIR=""
NEW_TARGET=""
PROFILE_TMP=""
cleanup() {
    [ -z "$TMP_DIR" ] || rm -rf "$TMP_DIR"
    [ -z "$NEW_TARGET" ] || rm -f "$NEW_TARGET"
    [ -z "$PROFILE_TMP" ] || rm -f "$PROFILE_TMP"
}
trap cleanup 0 HUP INT TERM

if [ "$SKIP_BINARY" -eq 0 ]; then
    for tool in curl tar; do
        command -v "$tool" >/dev/null 2>&1 || die "required tool is unavailable: $tool"
    done
    if command -v sha256sum >/dev/null 2>&1; then
        CHECKSUM_TOOL=sha256sum
    elif command -v shasum >/dev/null 2>&1; then
        CHECKSUM_TOOL=shasum
    else
        die "required checksum tool is unavailable (need sha256sum or shasum)"
    fi

    TMP_DIR=$(mktemp -d 2>/dev/null) || die "could not create temporary directory"
    ARCHIVE_NAME="ai-history_${VERSION#v}_${OS}_${ARCH}.tar.gz"
    CHECKSUMS=$TMP_DIR/checksums.txt
    ARCHIVE=$TMP_DIR/$ARCHIVE_NAME
    RELEASE_URL=$RELEASE_BASE_URL/$VERSION

    if ! curl -fsSL "$RELEASE_URL/checksums.txt" -o "$CHECKSUMS"; then
        die "download failed for $RELEASE_URL/checksums.txt (release $VERSION)"
    fi
    if ! curl -fsSL "$RELEASE_URL/$ARCHIVE_NAME" -o "$ARCHIVE"; then
        die "download failed for $RELEASE_URL/$ARCHIVE_NAME (release $VERSION)"
    fi

    EXPECTED=$(awk -v name="$ARCHIVE_NAME" '
        $2 == name && NF == 2 { count++; hash=$1 }
        END { if (count == 1) print hash; else exit 1 }
    ' "$CHECKSUMS") || die "checksum entry for $ARCHIVE_NAME must appear exactly once"
    case "$CHECKSUM_TOOL" in
        sha256sum) ACTUAL=$(sha256sum "$ARCHIVE" | awk '{print $1}') ;;
        shasum) ACTUAL=$(shasum -a 256 "$ARCHIVE" | awk '{print $1}') ;;
    esac
    [ "$ACTUAL" = "$EXPECTED" ] || die "checksum verification failed for $ARCHIVE_NAME"

    STAGE=$TMP_DIR/stage
    mkdir -p "$STAGE"
    if ! tar -xzf "$ARCHIVE" -C "$STAGE"; then
        die "could not extract downloaded archive $ARCHIVE_NAME"
    fi
    STAGED_BINARY=$STAGE/ai-history
    [ -f "$STAGED_BINARY" ] && [ -x "$STAGED_BINARY" ] || die "downloaded archive has no executable ai-history binary"
    STAGED_VERSION=$(binary_identity_version "$STAGED_BINARY") || die "downloaded binary has an invalid ai-history identity"
    [ "$STAGED_VERSION" = "$VERSION" ] || die "downloaded binary version does not match $VERSION"

    mkdir -p "$INSTALL_DIR"
    NEW_TARGET=$(mktemp "$INSTALL_DIR/.ai-history.new.XXXXXX" 2>/dev/null) || die "could not create staging file in $INSTALL_DIR"
    cp "$STAGED_BINARY" "$NEW_TARGET"
    chmod 0755 "$NEW_TARGET"
    mv -f "$NEW_TARGET" "$TARGET"
    INSTALLED_VERSION=$(binary_identity_version "$TARGET") || die "installed binary has an invalid ai-history identity: $TARGET"
    [ "$INSTALLED_VERSION" = "$VERSION" ] || die "installed binary version does not match $VERSION"
    printf 'Installed ai-history %s at %s\n' "$VERSION" "$TARGET"
fi

if [ "$MODIFY_PATH" -eq 1 ]; then
    case ":${PATH:-}:" in
        *:"$INSTALL_DIR":*) ;;
        *)
            QUOTED_DIR=$(printf '%s' "$INSTALL_DIR" | sed "s/'/'\\\\''/g")
            SHELL_NAME=$(basename "${SHELL:-}")
            case "$SHELL_NAME" in
                bash) PROFILE=$HOME/.bashrc ; LINE="export PATH='$QUOTED_DIR':\"\$PATH\" # ai-history installer" ;;
                zsh) PROFILE=$HOME/.zshrc ; LINE="export PATH='$QUOTED_DIR':\"\$PATH\" # ai-history installer" ;;
                fish) PROFILE=$HOME/.config/fish/config.fish ; LINE="fish_add_path '$QUOTED_DIR' # ai-history installer" ;;
                *)
                    printf 'Add %s to PATH: export PATH='"'"'%s'"'"':"$PATH"\n' "$INSTALL_DIR" "$QUOTED_DIR"
                    PROFILE=""
                    ;;
            esac
            if [ -n "$PROFILE" ]; then
                mkdir -p "$(dirname "$PROFILE")"
                update_profile
            fi
            ;;
    esac
fi

DOCTOR_OUTPUT=""
if ! DOCTOR_OUTPUT=$("$TARGET" doctor --json 2>/dev/null); then
    die "ai-history doctor --json failed"
fi
if ! printf '%s' "$DOCTOR_OUTPUT" | json_process array; then
    die "ai-history doctor returned invalid JSON diagnostics"
fi

PATH_BINARY=$(command -v ai-history 2>/dev/null || true)
if [ -n "$PATH_BINARY" ] && [ "$PATH_BINARY" != "$TARGET" ]; then
    printf 'Warning: PATH resolves ai-history to %s instead of installed %s\n' "$PATH_BINARY" "$TARGET" >&2
fi

if [ "$WITH_SKILL" -eq 1 ]; then
    die "Skill bundle not implemented"
fi

exit 0
