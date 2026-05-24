#!/usr/bin/env bash
# sync-docs.sh — Copy ../documentation/ into content/docs/, injecting Hugo front matter
# and converting MkDocs admonitions (!!!  note/warning/etc.) to callout shortcodes.
# Performs a clean sync: destination directories are wiped before each run so stale
# files from deleted source pages never survive.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SRC_DIR="$(realpath "$SCRIPT_DIR/../../documentation")"
DST_DIR="$SCRIPT_DIR/../content/docs"
BLOG_DIR="$SCRIPT_DIR/../content/blog"
PUB_DIR="$SCRIPT_DIR/../content/publications"

if [[ ! -d "$SRC_DIR" ]]; then
  echo "ERROR: docs source not found at $SRC_DIR" >&2
  exit 1
fi

# ── Helpers ──────────────────────────────────────────────────────────────────

inject_frontmatter() {
  local file="$1"
  local title="$2"
  local weight="$3"
  local src_file="${4:-}"

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

  # Get date from the source file's git history; fall back to today
  local date_str=""
  if [[ -n "$src_file" ]]; then
    date_str="$(git log --follow -1 --pretty=format:"%as" -- "$src_file" 2>/dev/null || true)"
  fi
  date_str="${date_str:-$(date +%Y-%m-%d)}"

  local tmp
  tmp="$(mktemp)"
  printf -- '---\ntitle: "%s"\ndate: %s\nweight: %s\n---\n\n' "$title" "$date_str" "$weight" > "$tmp"
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

clean_navigation() {
  local file="$1"
  python3 - "$file" <<'PYEOF'
import sys, re

path = sys.argv[1]
with open(path, 'r') as f:
    text = f.read()

# Remove lines that start with "→ Next:" or "→ Previous:"
text = re.sub(r'^→ Next:.*\n?', '', text, flags=re.MULTILINE)
text = re.sub(r'^→ Previous:.*\n?', '', text, flags=re.MULTILINE)

# Remove lines where the entire line is "→ See [something](something)"
# (starts with "→ See [") — avoids removing inline mid-paragraph cross-refs
text = re.sub(r'^→ See \[.*\n?', '', text, flags=re.MULTILINE)

# Remove bare "## Next" sections (heading alone, not "## Next Steps" etc.)
text = re.sub(r'^## Next[ \t]*\n.*?(?=^##|\Z)', '', text, flags=re.MULTILINE | re.DOTALL)

# Remove entire "## See also" sections (heading + all lines until next ## or EOF)
text = re.sub(r'^## See also\b.*?(?=^##|\Z)', '', text, flags=re.MULTILINE | re.DOTALL)

# Strip .md from markdown link display text: [something.md](url) → [something](url)
text = re.sub(r'\[([^\]]+?)\.md\](\([^)]*\))', r'[\1]\2', text)

with open(path, 'w') as f:
    f.write(text)
PYEOF
}

rewrite_links() {
  local file="$1"
  # Rewrite markdown cross-references: foo.md) → foo/) and index.md) → )
  # Hugo serves pages at /path/ not /path/index.md or /path.md
  # Anchor links (file.md#section) are handled first so the # isn't consumed
  # by the greedy non-anchor patterns. Bare paths (no ./ or ../ prefix) are
  # handled last; the [a-zA-Z] anchor and : exclusion prevent matching URLs.
  sed -i \
    -e 's|\(\.\/[^)#]*\)\/index\.md\(#[^)]*\))|\1/\2)|g' \
    -e 's|\(\.\.[^)#]*\)\/index\.md\(#[^)]*\))|\1/\2)|g' \
    -e 's|\([a-zA-Z][^):#]*\)\/index\.md\(#[^)]*\))|\1/\2)|g' \
    -e 's|\(\.\/[^)#]*\)\.md\(#[^)]*\))|\1/\2)|g' \
    -e 's|\(\.\.[^)#]*\)\.md\(#[^)]*\))|\1/\2)|g' \
    -e 's|\([a-zA-Z][^):#]*\)\.md\(#[^)]*\))|\1/\2)|g' \
    -e 's|\(\.\/[^)]*\)\/index\.md)|\1/)|g' \
    -e 's|\(\.\.[^)]*\)\/index\.md)|\1/)|g' \
    -e 's|\([a-zA-Z][^):]*\)\/index\.md)|\1/)|g' \
    -e 's|\(\.\/[^)]*\)\.md)|\1/)|g' \
    -e 's|\(\.\.[^)]*\)\.md)|\1/)|g' \
    -e 's|\([a-zA-Z][^):]*\)\.md)|\1/)|g' \
    "$file"
}

sync_file() {
  local src_file="$1"
  local dst_file="$2"
  local weight="$3"
  local src_for_date="${4:-$src_file}"

  mkdir -p "$(dirname "$dst_file")"
  cp "$src_file" "$dst_file"
  rewrite_links "$dst_file"
  clean_navigation "$dst_file"
  convert_admonitions "$dst_file"
  local title
  title="$(slugify_title "$src_file")"
  inject_frontmatter "$dst_file" "$title" "$weight" "$src_for_date"
}

# ── Clean destinations ────────────────────────────────────────────────────────
# Wipe and recreate docs so stale files from deleted pages are removed.
# For blog and publications, preserve the manually maintained _index.md.

echo "Cleaning content/docs ..."
rm -rf "${DST_DIR:?}"
mkdir -p "$DST_DIR"

echo "Cleaning content/blog (preserving _index.md) ..."
find "${BLOG_DIR:?}" -name '*.md' ! -name '_index.md' -delete 2>/dev/null || true

echo "Cleaning content/publications (preserving _index.md) ..."
find "${PUB_DIR:?}" -name '*.md' ! -name '_index.md' -delete 2>/dev/null || true

# ── Sync docs ─────────────────────────────────────────────────────────────────

echo "Syncing docs: $SRC_DIR → $DST_DIR"

weight=1
find "$SRC_DIR" -name '*.md' | sort | while read -r src_file; do
  rel_path="${src_file#$SRC_DIR/}"

  # Blog and publications are synced separately below
  if [[ "$rel_path" == blog/* ]] || [[ "$rel_path" == publications/* ]]; then
    continue
  fi

  dst_file="$DST_DIR/$rel_path"

  # Hugo requires _index.md (not index.md) for section branch bundles.
  if [[ "$(basename "$dst_file")" == "index.md" ]]; then
    dst_file="${dst_file%index.md}_index.md"
  fi

  sync_file "$src_file" "$dst_file" "$weight" "$src_file"
  ((weight++)) || true
  echo "  docs: $rel_path"
done

# ── Auto-create missing section _index.md files ───────────────────────────────

find "$DST_DIR" -mindepth 1 -maxdepth 4 -type d | while read -r dir; do
  index="$dir/_index.md"
  if [[ ! -f "$index" ]]; then
    section_title="$(basename "$dir" | sed 's/[-_]/ /g' | awk '{for(i=1;i<=NF;i++) $i=toupper(substr($i,1,1)) substr($i,2); print}')"
    printf -- '---\ntitle: "%s"\nweight: 1\n---\n' "$section_title" > "$index"
    echo "  created index: ${index#$DST_DIR/}"
  fi
done

# ── Sync blog ─────────────────────────────────────────────────────────────────

echo "Syncing blog: $SRC_DIR/blog → $BLOG_DIR"

weight=1
find "$SRC_DIR/blog" -name '*.md' 2>/dev/null | sort | while read -r src_file; do
  dst_file="$BLOG_DIR/$(basename "$src_file")"
  sync_file "$src_file" "$dst_file" "$weight" "$src_file"
  ((weight++)) || true
  echo "  blog: $(basename "$src_file")"
done

# ── Sync publications ─────────────────────────────────────────────────────────

echo "Syncing publications: $SRC_DIR/publications → $PUB_DIR"

weight=1
find "$SRC_DIR/publications" -name '*.md' 2>/dev/null | sort | while read -r src_file; do
  dst_file="$PUB_DIR/$(basename "$src_file")"
  sync_file "$src_file" "$dst_file" "$weight" "$src_file"
  ((weight++)) || true
  echo "  pub: $(basename "$src_file")"
done

echo "Done."
