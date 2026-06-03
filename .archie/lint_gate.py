#!/usr/bin/env python3
"""Archie lint gate — run project-native linters on files the agent writes.

Opt-in via ``.archie/enforcement.json``. Invoked as a PostToolUse hook after
Write/Edit/MultiEdit. Detects the right linter for the file type based on
project config (pyproject.toml, package.json, .semgrep.yml), runs it on the
single changed file, and exits 2 when severity=error so Claude Code blocks.

Zero dependencies beyond Python 3.9+ stdlib. Fails open on every error path —
missing config, missing linter, ambiguous file type — so a misconfigured gate
never breaks the agent's flow.

Config schema (``.archie/enforcement.json``)::

    {
      "enabled": true,
      "severity": "error",         # error → exit 2 (block), warn → exit 0 with message
      "linters": {                 # optional overrides; omit to auto-detect
        "python": {"command": "ruff check --quiet"},
        "js":     {"command": "eslint --quiet"}
      }
    }
"""

from __future__ import annotations

import json
import os
import shutil
import subprocess
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from _common import safe_read_text  # noqa: E402


# File extension → logical linter kind.
EXT_TO_KIND = {
    ".py": "python",
    ".js": "js",
    ".jsx": "js",
    ".ts": "js",
    ".tsx": "js",
    ".mjs": "js",
    ".cjs": "js",
    ".go": "go",
}


def load_config(project_root: Path) -> dict | None:
    """Return config dict if enabled, else None (fail open)."""
    cfg_path = project_root / ".archie" / "enforcement.json"
    if not cfg_path.is_file():
        return None
    try:
        cfg = json.loads(safe_read_text(cfg_path, project_root))
    except (json.JSONDecodeError, OSError, ValueError):
        return None
    if not cfg.get("enabled"):
        return None
    return cfg


def detect_linter(file_path: Path, project_root: Path, config: dict) -> dict | None:
    """Pick the linter for this file. Returns a dict with ``command`` and
    ``kind``, or None if nothing applies (fail open).

    Resolution:
    1. Extension maps to a kind ("python", "js", ...)
    2. Explicit config override wins
    3. Otherwise auto-detect based on project config files + executable on PATH
    4. Always check for ``.semgrep.yml`` / ``.semgrep.yaml`` / ``semgrep.yml``
       as a universal fallback when the kind linter is absent
    """
    ext = file_path.suffix.lower()
    kind = EXT_TO_KIND.get(ext)

    linters_cfg = config.get("linters", {})

    if kind:
        # Explicit override from config.
        override = linters_cfg.get(kind)
        if isinstance(override, dict) and override.get("command"):
            target = override.get("target", _default_target(kind))
            return {"kind": kind, "command": override["command"], "target": target}

        # Auto-detect per kind.
        if kind == "python":
            auto = _detect_python_linter(project_root)
            if auto:
                return {"kind": kind, "command": auto, "target": "file"}
        elif kind == "js":
            auto = _detect_js_linter(project_root)
            if auto:
                return {"kind": kind, "command": auto, "target": "file"}
        elif kind == "go":
            auto = _detect_go_linter(project_root)
            if auto:
                # golangci-lint is package-aware — pass the containing
                # directory, not the single file, or it errors with "no go
                # files found".
                return {"kind": kind, "command": auto, "target": "parent"}

    # Semgrep fallback — applies to any file type when the user has a semgrep
    # config checked in.
    if _has_semgrep_config(project_root) and shutil.which("semgrep"):
        return {"kind": "semgrep", "command": "semgrep --error --quiet", "target": "file"}

    return None


def _default_target(kind: str) -> str:
    """File-oriented vs package-oriented linters."""
    return "parent" if kind == "go" else "file"


def _detect_python_linter(project_root: Path) -> str | None:
    """Return a ruff command when the project is configured for ruff."""
    if not shutil.which("ruff"):
        return None
    pyproject = project_root / "pyproject.toml"
    if pyproject.is_file():
        try:
            text = safe_read_text(pyproject, project_root)
            if "[tool.ruff]" in text or "[tool.ruff." in text:
                return "ruff check --quiet"
        except (OSError, ValueError):
            pass
    if (project_root / "ruff.toml").is_file():
        return "ruff check --quiet"
    if (project_root / ".ruff.toml").is_file():
        return "ruff check --quiet"
    return None


def _detect_js_linter(project_root: Path) -> str | None:
    """Return an eslint command when the project is configured for eslint."""
    # Local eslint binary is preferred so version matches the repo's config.
    local = project_root / "node_modules" / ".bin" / "eslint"
    for cfg_name in (
        ".eslintrc", ".eslintrc.json", ".eslintrc.js", ".eslintrc.cjs",
        ".eslintrc.yaml", ".eslintrc.yml", "eslint.config.js", "eslint.config.mjs",
    ):
        if (project_root / cfg_name).is_file():
            if local.is_file():
                return f"{local} --quiet"
            if shutil.which("eslint"):
                return "eslint --quiet"
    # package.json with eslint dep is also a signal.
    pkg = project_root / "package.json"
    if pkg.is_file():
        try:
            data = json.loads(safe_read_text(pkg, project_root))
            deps = {**data.get("dependencies", {}), **data.get("devDependencies", {})}
            if "eslint" in deps:
                if local.is_file():
                    return f"{local} --quiet"
                if shutil.which("eslint"):
                    return "eslint --quiet"
        except (json.JSONDecodeError, OSError, ValueError):
            pass
    return None


def _detect_go_linter(project_root: Path) -> str | None:
    """Return a golangci-lint command when the project is configured for it."""
    if not shutil.which("golangci-lint"):
        return None
    for cfg_name in (
        ".golangci.yaml",
        ".golangci.yml",
        ".golangci.toml",
        ".golangci.json",
    ):
        if (project_root / cfg_name).is_file():
            return "golangci-lint run --fast"
    return None


def _has_semgrep_config(project_root: Path) -> bool:
    for name in (".semgrep.yml", ".semgrep.yaml", "semgrep.yml", "semgrep.yaml"):
        if (project_root / name).is_file():
            return True
    return False


def run_linter(
    command: str,
    file_path: Path,
    cwd: Path,
    timeout: int = 30,
) -> tuple[int, str]:
    """Run the linter on a single file. Returns (exit_code, combined_output).

    Uses shell=True so commands like ``ruff check --quiet`` stay readable.
    Wraps any spawn error as exit 0 (fail open — we will not pretend a missing
    linter is a lint failure).
    """
    try:
        proc = subprocess.run(
            f"{command} {_shquote(str(file_path))}",
            shell=True,
            cwd=str(cwd),
            capture_output=True,
            text=True,
            timeout=timeout,
        )
    except (FileNotFoundError, subprocess.TimeoutExpired, OSError):
        return 0, ""
    output = (proc.stdout or "") + (proc.stderr or "")
    return proc.returncode, output.strip()


def _shquote(s: str) -> str:
    """Minimal shell-quoting for a single arg."""
    if not s or any(c in s for c in " '\"\\$`"):
        return "'" + s.replace("'", "'\\''") + "'"
    return s


def gate(
    project_root: Path,
    file_path: Path,
) -> tuple[int, str]:
    """Top-level entrypoint used by the hook. Returns (exit_code, message).

    - exit_code 0 with empty message: all clear, nothing to report
    - exit_code 0 with message: warn-mode finding, surfaced to the agent
    - exit_code 2: block — severity=error and linter found issues
    """
    cfg = load_config(project_root)
    if cfg is None:
        return 0, ""

    linter = detect_linter(file_path, project_root, cfg)
    if linter is None:
        return 0, ""

    # Some linters are package-aware (golangci-lint) and must be pointed at
    # the containing directory, not the single file.
    target_path = file_path.parent if linter.get("target") == "parent" else file_path

    code, output = run_linter(linter["command"], target_path, project_root)
    if code == 0:
        return 0, ""

    severity = cfg.get("severity", "error")
    prefix = "[Archie lint-gate BLOCKED]" if severity == "error" else "[Archie lint-gate WARNING]"
    rel = _relative(file_path, project_root)
    msg = f"{prefix} {linter['kind']} reported issues in {rel}:\n{output}"
    return (2 if severity == "error" else 0), msg


def _relative(p: Path, root: Path) -> str:
    try:
        return str(p.relative_to(root))
    except ValueError:
        return str(p)


def _main(argv: list[str]) -> int:
    """CLI entrypoint used by post-lint.sh.

    Reads the Claude Code tool-call JSON from stdin, resolves the project root
    and file path, and calls ``gate()``. Exits 0 on fail-open paths and on
    non-Write/Edit/MultiEdit tools.
    """
    try:
        data = json.load(sys.stdin)
    except Exception:
        return 0
    if data.get("tool_name") not in ("Write", "Edit", "MultiEdit"):
        return 0
    ti = data.get("tool_input") or {}
    fp = ti.get("file_path") or ti.get("path")
    if not fp:
        return 0

    # Resolve project root. Prefer env, fallback to git, fallback to cwd.
    root_env = os.environ.get("_ARCHIE_ROOT")
    if root_env:
        root = Path(root_env)
    else:
        try:
            out = subprocess.check_output(
                ["git", "rev-parse", "--show-toplevel"],
                stderr=subprocess.DEVNULL,
                text=True,
            ).strip()
            root = Path(out) if out else Path.cwd()
        except (subprocess.CalledProcessError, FileNotFoundError):
            root = Path.cwd()

    file_path = Path(fp)
    if not file_path.is_file():
        return 0

    code, msg = gate(root, file_path)
    if msg:
        print(msg)
    return code


if __name__ == "__main__":
    sys.exit(_main(sys.argv[1:]))
