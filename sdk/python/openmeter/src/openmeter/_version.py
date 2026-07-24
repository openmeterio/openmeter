"""Package version, sourced from installed distribution metadata."""

from __future__ import annotations

from importlib import metadata

try:
    __version__ = metadata.version("openmeter")
except metadata.PackageNotFoundError:  # pragma: no cover - unbuilt source checkout
    __version__ = "0.0.0+unknown"
