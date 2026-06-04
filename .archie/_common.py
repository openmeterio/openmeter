"""Archie shared utilities — imported by other standalone scripts.

Deduplicates helpers that were copy-pasted across 6+ files.

Zero dependencies beyond Python 3.9+ stdlib.
"""
from __future__ import annotations

import ast
import fnmatch
import json
import os
import re
from pathlib import Path


# ── Shared constants ──────────────────────────────────────────────────────

SOURCE_EXTENSIONS = {
    ".py", ".kt", ".kts", ".java", ".js", ".jsx", ".ts", ".tsx",
    ".swift", ".go", ".rs", ".rb", ".c", ".cpp", ".cc", ".cxx", ".h",
    ".hpp", ".cs", ".php", ".scala", ".m", ".mm",
}

SKIP_DIRS = {
    ".git", "node_modules", "__pycache__", ".venv", "venv", "env",
    ".tox", ".mypy_cache", ".pytest_cache", ".ruff_cache",
    "dist", "build", ".build", ".next", ".nuxt", ".svelte-kit",
    "coverage", ".nyc_output", ".turbo", ".parcel-cache",
    "vendor", "Pods", "DerivedData", ".gradle", ".idea", ".vscode",
    ".archie", ".claude",
    ".devenv",       # Nix devenv
    ".swiftpm",      # Swift PM cache
    ".pub-cache",    # Dart/Flutter
    ".dart_tool",    # Dart
    ".ccache",       # C/C++ compiler cache
}

# ── IgnoreMatcher ─────────────────────────────────────────────────────────


class _IgnorePattern:
    """A single parsed gitignore-style pattern."""

    __slots__ = ("pattern", "is_dir_only", "is_rooted", "is_negation", "scope")

    def __init__(self, raw: str, scope: str = ""):
        """Parse a raw gitignore line into a structured pattern.

        Args:
            raw: The pattern string (already stripped of comments/blanks).
            scope: Relative directory scope (empty string for root-level files).
        """
        self.scope = scope
        self.is_negation = raw.startswith("!")
        if self.is_negation:
            raw = raw[1:]

        self.is_dir_only = raw.endswith("/")
        if self.is_dir_only:
            raw = raw.rstrip("/")

        # A pattern is "rooted" if it contains a / (after stripping trailing /)
        # or starts with /. Leading / is removed after marking as rooted.
        self.is_rooted = raw.startswith("/") or "/" in raw
        if raw.startswith("/"):
            raw = raw[1:]

        # Strip leading **/ — it means "match at any depth" (same as unrooted)
        if raw.startswith("**/"):
            raw = raw[3:]
            self.is_rooted = False

        self.pattern = raw

    def _matches_path(self, name: str, rel_parent: str) -> bool:
        """Check whether this pattern matches the given name under rel_parent."""
        # If the pattern is scoped (from a nested .gitignore), the rel_parent
        # must be equal to or nested under the scope.
        if self.scope:
            if rel_parent != self.scope and not rel_parent.startswith(self.scope + "/"):
                return False

        if self.is_rooted:
            # Rooted patterns match relative to their scope.
            if self.scope:
                # Remove scope prefix to get path relative to .gitignore dir
                if rel_parent == self.scope:
                    check_path = name
                elif rel_parent.startswith(self.scope + "/"):
                    sub = rel_parent[len(self.scope) + 1:]
                    check_path = sub + "/" + name
                else:
                    return False
            else:
                check_path = (rel_parent + "/" + name) if rel_parent else name
            return fnmatch.fnmatch(check_path, self.pattern)
        else:
            # Unrooted: match the basename at any depth
            return fnmatch.fnmatch(name, self.pattern)

    def matches_dir(self, dirname: str, rel_parent: str) -> bool:
        """Check if this pattern matches a directory."""
        # File-only patterns (not dir_only) with a glob like *.ext
        # should not match directories. But plain names without glob
        # can match dirs even without trailing /.
        return self._matches_path(dirname, rel_parent)

    def matches_file(self, filename: str, rel_parent: str) -> bool:
        """Check if this pattern matches a file."""
        if self.is_dir_only:
            return False
        return self._matches_path(filename, rel_parent)


def safe_read_text(path, base=None, *, errors: str = "replace") -> str:
    """Single audited file-read sink for the standalone tools.

    Resolves *path* and, when *base* is given, refuses to read anything that
    escapes that directory (a path-containment check). Routing every read
    through here means the tools have one sanitized sink instead of many bare
    ``read_text()``/``open()`` calls on derived paths — these only ever read
    fixed, project-internal locations, never attacker input.
    """
    p = Path(path).resolve()
    if base is not None:
        b = Path(base).resolve()
        if b != p and b not in p.parents:
            raise ValueError(f"refusing to read outside {b}: {p}")
    return p.read_text(encoding="utf-8", errors=errors)


def _parse_ignore_file(path: Path, scope: str = "") -> list[_IgnorePattern]:
    """Parse a gitignore-format file into a list of _IgnorePattern objects."""
    if not path.exists():
        return []
    try:
        text = safe_read_text(path)
    except OSError:
        return []
    patterns: list[_IgnorePattern] = []
    for line in text.splitlines():
        line = line.rstrip()
        if not line or line.startswith("#"):
            continue
        patterns.append(_IgnorePattern(line, scope=scope))
    return patterns


def _collect_nested_gitignores(root: Path) -> list[_IgnorePattern]:
    """Walk the tree and collect patterns from nested .gitignore files.

    Root .gitignore is excluded (handled separately). Each nested
    .gitignore's patterns are scoped to its directory.
    """
    patterns: list[_IgnorePattern] = []
    for dirpath, dirnames, filenames in os.walk(root):
        # Skip .git and other VCS dirs
        dirnames[:] = [d for d in dirnames if d not in (".git",)]
        rel = os.path.relpath(dirpath, root)
        if rel == ".":
            continue  # root .gitignore handled separately
        if ".gitignore" in filenames:
            scope = rel.replace(os.sep, "/")
            patterns.extend(
                _parse_ignore_file(Path(dirpath) / ".gitignore", scope=scope)
            )
    return patterns


class IgnoreMatcher:
    """Merge .archieignore + .gitignore patterns for file/directory filtering.

    Usage with os.walk::

        matcher = IgnoreMatcher(project_root)
        for root, dirs, files in os.walk(project_root):
            rel = os.path.relpath(root, project_root)
            if rel == ".":
                rel = ""
            dirs[:] = [d for d in dirs if not matcher.should_skip_dir(d, rel)]
            files = [f for f in files if not matcher.should_skip_file(f, rel)]
    """

    def __init__(self, root: str | Path):
        root = Path(root)
        self._root = root
        self._patterns: list[_IgnorePattern] = []

        # .archieignore takes priority and is always loaded
        archieignore = root / ".archieignore"
        self._patterns.extend(_parse_ignore_file(archieignore))

        # .gitignore at root
        gitignore = root / ".gitignore"
        self._patterns.extend(_parse_ignore_file(gitignore))

        # Nested .gitignore files
        self._patterns.extend(_collect_nested_gitignores(root))

    def _check(self, name: str, rel_parent: str, is_dir: bool) -> bool:
        """Evaluate patterns in order — last matching pattern wins (git semantics)."""
        matched = False
        for pat in self._patterns:
            if is_dir:
                hit = pat.matches_dir(name, rel_parent)
            else:
                hit = pat.matches_file(name, rel_parent)
            if hit:
                matched = not pat.is_negation
        return matched

    def should_skip_dir(self, dirname: str, parent_rel: str) -> bool:
        """Should this directory be pruned during os.walk?

        Args:
            dirname: The directory basename (e.g. "node_modules").
            parent_rel: The relative path from project root to the parent
                        directory (e.g. "" for root, "src/pkg" for nested).
        """
        return self._check(dirname, parent_rel, is_dir=True)

    def should_skip_file(self, filename: str, parent_rel: str) -> bool:
        """Should this file be skipped during os.walk?

        Args:
            filename: The file basename (e.g. "main.py").
            parent_rel: The relative path from project root to the parent
                        directory.
        """
        return self._check(filename, parent_rel, is_dir=False)

    def is_ignored(self, rel_path: str) -> bool:
        """Convenience: check if a relative path is ignored.

        For directories, checks as a directory. For files, checks as a file.
        If the path exists under root and is a directory, it's treated as a dir.
        Otherwise treated as a file.

        Gitignore semantics: a path is also ignored when any ANCESTOR directory
        matches a dir pattern (``vendor/`` ignores ``vendor/b.py``). os.walk
        callers get this for free via directory pruning; full-path callers
        (e.g. drift's git-log list, where the file may no longer exist on disk)
        rely on this ancestor walk.
        """
        rel_path = rel_path.replace(os.sep, "/")
        parts = rel_path.rsplit("/", 1)
        if len(parts) == 2:
            parent, name = parts
        else:
            parent, name = "", parts[0]

        # Determine if it's a directory
        full = self._root / rel_path
        is_dir = full.is_dir()

        if self._check(name, parent, is_dir=is_dir):
            return True

        # Walk ancestor directories — an ignored parent dir hides everything
        # beneath it, regardless of whether the leaf itself matches.
        segments = rel_path.split("/")
        for i in range(len(segments) - 1):
            if self._check(segments[i], "/".join(segments[:i]), is_dir=True):
                return True
        return False


# ── BulkMatcher — classify files as bulk-content (visible but not read) ──

def _glob_to_regex(glob: str) -> re.Pattern:
    """Translate a shell-style glob to a regex for full-path matching.

    Differs from fnmatch to give sane `**` semantics across path components:
    - `**/` (with trailing slash) matches zero or more directory segments
    - `**` (without slash) matches any characters including `/`
    - `*` matches any characters except `/`
    - `?` matches a single non-`/` character
    - `[abc]` keeps standard character-class semantics
    """
    out: list[str] = []
    i = 0
    n = len(glob)
    while i < n:
        c = glob[i]
        if c == '*':
            if i + 1 < n and glob[i + 1] == '*':
                if i + 2 < n and glob[i + 2] == '/':
                    out.append(r'(?:[^/]+/)*')
                    i += 3
                else:
                    out.append(r'.*')
                    i += 2
            else:
                out.append(r'[^/]*')
                i += 1
        elif c == '?':
            out.append(r'[^/]')
            i += 1
        elif c == '[':
            j = glob.find(']', i)
            if j == -1:
                out.append(r'\[')
                i += 1
            else:
                out.append(glob[i:j + 1])
                i = j + 1
        else:
            out.append(re.escape(c))
            i += 1
    return re.compile('^' + ''.join(out) + '$')


def _find_archiebulk(root: Path) -> Path | None:
    """Locate the nearest `.archiebulk` walking up from `root`.

    Walk-up resolution (editorconfig/prettier style — NOT gitignore):
    prefer a `.archiebulk` in the project root, otherwise walk up the
    directory tree. Stops at the filesystem root, the user's home directory
    (don't escape into ~), or when no further parent exists.

    Required for monorepo scans: when `archie-deep-scan` runs with
    `PROJECT_ROOT` set to a subpackage (e.g. `<monorepo>/packages/android/`),
    the `.archiebulk` typically lives at the monorepo root. Without this
    walk, `bulk_content_manifest` is empty and `frontend_ratio` collapses
    to 0 — UI Layer never spawns and the agent misreads the project shape.

    The walk deliberately ignores `.git` boundaries: a sub-project may be
    a git submodule with its own `.git`, but the monorepo's `.archiebulk`
    still describes the right bulk patterns for that subtree. `.archiebulk`
    is an Archie convention, not a git one.
    """
    try:
        current = root.resolve()
    except OSError:
        return None
    try:
        home = Path.home().resolve()
    except (OSError, RuntimeError):
        home = None
    while True:
        candidate = current / ".archiebulk"
        if candidate.is_file():
            return candidate
        parent = current.parent
        if parent == current:
            return None  # filesystem root
        # Stop before escaping into the user's home directory.
        if home is not None and current == home:
            return None
        current = parent


class BulkMatcher:
    """Classify files as bulk-content (visible to the scanner, never read).

    Loads `.archiebulk` from the project root, falling back to the nearest
    ancestor with a `.archiebulk` (gitignore-style walk, bounded by the
    enclosing git repo root). Each non-comment line carries three
    whitespace-separated columns: `<glob>  <category>  <framework>`. The
    framework column is optional (use `-` or omit).

    `classify(rel_path)` returns `{category, framework}` for a match, else None.
    Last matching rule wins (gitignore-style precedence).
    """

    def __init__(self, root: str | Path):
        root = Path(root)
        self._root = root
        self._rules: list[tuple[re.Pattern, str, str]] = []
        path = _find_archiebulk(root)
        if path is None:
            return
        try:
            text = safe_read_text(path, root)
        except (OSError, ValueError):
            return
        for raw in text.splitlines():
            line = raw.strip()
            if not line or line.startswith("#"):
                continue
            parts = line.split(None, 2)
            if len(parts) < 2:
                continue
            glob_pat, category = parts[0], parts[1]
            framework = parts[2].strip() if len(parts) == 3 else "-"
            # Allow an inline comment after the framework column.
            if "#" in framework:
                framework = framework.split("#", 1)[0].strip() or "-"
            try:
                rx = _glob_to_regex(glob_pat)
            except re.error:
                continue
            self._rules.append((rx, category, framework))

    def classify(self, rel_path: str) -> dict | None:
        """Return {category, framework} if the path is bulk, else None.

        Last matching rule wins, so later entries in `.archiebulk` override
        earlier ones (mirroring gitignore precedence).
        """
        if not self._rules:
            return None
        norm = rel_path.replace(os.sep, "/")
        result: dict | None = None
        for rx, cat, fw in self._rules:
            if rx.match(norm):
                result = {"category": cat, "framework": fw}
        return result

    def __bool__(self) -> bool:
        return bool(self._rules)


# Regex decision-point patterns for non-Python languages.
# NOTE: bare ``else`` is intentionally excluded — it is NOT a decision point
# in cyclomatic complexity (it's the default path, not an independent one).
DECISION_RE = re.compile(
    r"""(?x)
    \b(?:if|elif|else\s+if|elseif|for|foreach|while|do\b.*\bwhile|
        switch|case|catch|except|when|guard)\b
    | \?\s*[^?]     # ternary  ?:
    | &&             # logical AND
    | \|\|           # logical OR
    """
)


# ── Shared helpers ────────────────────────────────────────────────────────

def _load_json(path: Path, base=None) -> dict | list:
    """Load a JSON file, returning empty dict on failure."""
    if Path(path).exists():
        try:
            return json.loads(safe_read_text(path, base))
        except (json.JSONDecodeError, OSError, ValueError):
            pass
    return {}


def normalize_blueprint(bp: dict) -> dict:
    """Normalize blueprint to canonical schema. Safe to call multiple times.

    Ensures:
    - Dict sections (meta, components, decisions, etc.) are dicts
    - components is always {"components": [...], ...} (wraps plain list)
    - List sections (pitfalls, implementation_guidelines, etc.) are lists
    - architecture_diagram exists as string
    """
    # Sections that must be dicts
    for key in ("meta", "architecture_rules", "decisions",
                "communication", "quick_reference", "technology", "frontend",
                "deployment"):
        val = bp.get(key)
        if not isinstance(val, dict):
            bp[key] = {} if val is None else {}

    # Components: can arrive as list or {"components": [...]}
    comps = bp.get("components")
    if isinstance(comps, list):
        bp["components"] = {"components": comps}
    elif not isinstance(comps, dict):
        bp["components"] = {"components": []}
    elif "components" not in comps:
        bp["components"]["components"] = []

    # Sections that must be lists
    for key in ("pitfalls", "implementation_guidelines", "development_rules"):
        val = bp.get(key)
        if not isinstance(val, list):
            bp[key] = []

    bp.setdefault("architecture_diagram", "")
    return bp


def _read_file(path: str, base=None) -> str | None:
    """Read a file, returning None on failure."""
    try:
        return safe_read_text(path, base)
    except (OSError, UnicodeDecodeError, ValueError):
        return None


# ── Cyclomatic complexity ─────────────────────────────────────────────────

def _cc_python_function(source: str, func_start: int, func_end: int) -> int:
    """Compute cyclomatic complexity of a Python function via AST."""
    lines = source.splitlines(True)
    func_lines = lines[func_start - 1 : func_end]
    if not func_lines:
        return 1

    # Dedent to column 0 so ast.parse works
    min_indent = 9999
    for ln in func_lines:
        stripped = ln.rstrip("\n\r")
        if stripped.strip():
            indent = len(stripped) - len(stripped.lstrip())
            min_indent = min(min_indent, indent)
    if min_indent == 9999:
        min_indent = 0

    dedented = "".join(ln[min_indent:] if len(ln) > min_indent else ln for ln in func_lines)

    try:
        tree = ast.parse(dedented, mode="exec")
    except SyntaxError:
        return 1

    cc = 1
    for node in ast.walk(tree):
        if isinstance(node, (ast.If, ast.For, ast.While, ast.ExceptHandler, ast.Assert)):
            cc += 1
        elif isinstance(node, ast.BoolOp):
            cc += len(node.values) - 1
        elif isinstance(node, ast.comprehension):
            cc += 1
            cc += len(node.ifs)
    return cc


def _cc_regex(lines: list[str]) -> int:
    """Approximate cyclomatic complexity via regex for non-Python files."""
    cc = 1
    for line in lines:
        cleaned = re.sub(r'"(?:[^"\\]|\\.)*"', '""', line)
        cleaned = re.sub(r"'(?:[^'\\]|\\.)*'", "''", cleaned)
        cc += len(DECISION_RE.findall(cleaned))
    return cc
