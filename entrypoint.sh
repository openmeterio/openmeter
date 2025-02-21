#!/usr/bin/env sh

set -euo pipefail

main() {
    # Update CA certificates if needed
    find /usr/local/share/ca-certificates -maxdepth 0 ! -empty -exec update-ca-certificates \;

    # Start CMD
    exec "$@"
}

main "$@"
