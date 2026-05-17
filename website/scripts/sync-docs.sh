#!/usr/bin/env bash
# sync-docs.sh — Copy ../documentation/ into content/docs/, injecting Hugo front matter
# and converting MkDocs admonitions (!!!  note/warning/etc.) to callout shortcodes.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SRC_DIR="$(realpath "$SCRIPT_DIR/../../documentation")"
DST_DIR="$SCRIPT_DIR/../content/docs"

if [[ ! -d "$SRC_DIR" ]]; then
  echo "ERROR: docs source not found at $SRC_DIR" >&2
  exit 1
fi

echo "Syncing docs: $SRC_DIR → $DST_DIR"
mkdir -p "$DST_DIR"

# ── Helpers ──────────────────────────────────────────────────────────────────

inject_frontmatter() {
  local file="$1"
  local title="$2"
  local weight="$3"

  # Skip if front matter already present
  if head -1 "$file" | grep -q '^---'; then
    return
  fi

  # Derive title from first H1 heading if present, then strip it from content.
  # The Hugo template renders the title as its own <h1> — keeping the markdown
  # H1 produces a duplicate heading on every page.
  local h1
  h1="$(grep -m1 '^# ' "$file" | sed 's/^# //' || true)"
  if [[ -n "$h1" ]]; then
    title="$h1"
    # Remove the first H1 line (and the blank line immediately after it, if any)
    sed -i '0,/^# /{/^# /d}' "$file"
    # Remove leading blank lines left behind
    sed -i '/./,$!d' "$file"
  fi

  local tmp
  tmp="$(mktemp)"
  printf -- '---\ntitle: "%s"\nweight: %s\n---\n\n' "$title" "$weight" > "$tmp"
  cat "$file" >> "$tmp"
  mv "$tmp" "$file"
}

slugify_title() {
  # Convert filename to Title Case, stripping leading numbers and hyphens
  basename "$1" .md \
    | sed 's/^[0-9]*[-_]*//' \
    | sed 's/[-_]/ /g' \
    | awk '{for(i=1;i<=NF;i++) $i=toupper(substr($i,1,1)) substr($i,2); print}'
}

convert_admonitions() {
  local file="$1"
  # MkDocs: !!! note "Optional title"
  #           body line (indented 4 spaces)
  # Hugo callout: {{< callout type="note" title="Optional title" >}} body {{< /callout >}}

  python3 - "$file" <<'PYEOF'
import sys, re

path = sys.argv[1]
with open(path, 'r') as f:
    text = f.read()

# Match admonition blocks: !!! type "optional title"\n(    indented lines)+
admonition_re = re.compile(
    r'^!!![ \t]+(\w+)(?:[ \t]+"([^"]*)")?[ \t]*\n((?:[ \t]{4}[^\n]*\n?)*)',
    re.MULTILINE
)

def replace_admonition(m):
    kind = m.group(1).lower()
    title = m.group(2) or ''
    body_raw = m.group(3)
    body = re.sub(r'^    ', '', body_raw, flags=re.MULTILINE).rstrip()
    title_attr = f' title="{title}"' if title else ''
    return f'{{{{< callout type="{kind}"{title_attr} >}}}}\n{body}\n{{{{< /callout >}}}}\n'

text = admonition_re.sub(replace_admonition, text)

with open(path, 'w') as f:
    f.write(text)
PYEOF
}

# ── Main sync ─────────────────────────────────────────────────────────────────

weight=1

# Walk all .md files under docs/
find "$SRC_DIR" -name '*.md' | sort | while read -r src_file; do
  rel_path="${src_file#$SRC_DIR/}"

  # Skip blog/ and publications/ — they have their own top-level content directories
  if [[ "$rel_path" == blog/* ]] || [[ "$rel_path" == publications/* ]]; then
    continue
  fi

  dst_file="$DST_DIR/$rel_path"

  # Hugo requires _index.md (not index.md) for section branch bundles.
  # Any source index.md becomes _index.md so sub-pages render correctly.
  if [[ "$(basename "$dst_file")" == "index.md" ]]; then
    dst_file="${dst_file%index.md}_index.md"
  fi

  # If destination _index.md already exists with a custom weight, preserve it.
  existing_weight=""
  if [[ -f "$dst_file" ]] && head -1 "$dst_file" | grep -q '^---'; then
    existing_weight="$(grep '^weight:' "$dst_file" | head -1 | awk '{print $2}')"
  fi

  mkdir -p "$(dirname "$dst_file")"
  cp "$src_file" "$dst_file"

  # Rewrite markdown cross-references: foo.md) → foo/) and index.md) → )
  # Hugo serves pages at /path/ not /path/index.md or /path.md
  sed -i \
    -e 's|\(\.\/[^)]*\)\/index\.md)|\1/)|g' \
    -e 's|\(\.\.[^)]*\)\/index\.md)|\1/)|g' \
    -e 's|\(\.\/[^)]*\)\.md)|\1/)|g' \
    -e 's|\(\.\.[^)]*\)\.md)|\1/)|g' \
    "$dst_file"

  convert_admonitions "$dst_file"

  title="$(slugify_title "$src_file")"
  effective_weight="${existing_weight:-$weight}"
  inject_frontmatter "$dst_file" "$title" "$effective_weight"

  ((weight++)) || true
  echo "  synced: $rel_path"
done

# ── Section _index.md files ───────────────────────────────────────────────────

find "$DST_DIR" -mindepth 1 -maxdepth 3 -type d | while read -r dir; do
  index="$dir/_index.md"
  if [[ ! -f "$index" ]]; then
    section_title="$(basename "$dir" | sed 's/[-_]/ /g' | awk '{for(i=1;i<=NF;i++) $i=toupper(substr($i,1,1)) substr($i,2); print}')"
    printf -- '---\ntitle: "%s"\nweight: 1\n---\n' "$section_title" > "$index"
    echo "  created index: ${index#$DST_DIR/}"
  fi
done

echo "Done."
