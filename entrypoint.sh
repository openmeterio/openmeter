#!/usr/bin/env sh

set -euo pipefail

main() {
    # The system trust store is root-owned, so merge custom CAs into a bundle we can write.
    custom_cas="$(find /usr/local/share/ca-certificates -maxdepth 1 -name '*.crt')"
    if [ -n "$custom_cas" ]; then
        bundle="${TMPDIR:-/tmp}/ca-certificates.crt"
        cat /etc/ssl/certs/ca-certificates.crt $custom_cas > "$bundle"
        export SSL_CERT_FILE="$bundle"
    fi

    # Start CMD
    exec "$@"
}

main "$@"
