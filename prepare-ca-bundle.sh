#!/bin/sh
set -eu

# A Compose volume can outlive an application container. Publish a complete
# bundle only after all input files were read successfully.
output=/var/run/openmeter-ca/ca-certificates.crt
bundle=$(mktemp /var/run/openmeter-ca/.ca-certificates.XXXXXX)
trap 'rm -f "$bundle"' EXIT
{
    cat /etc/ssl/certs/ca-certificates.crt
    for certificate in /usr/local/share/ca-certificates/*.crt; do
        [ -e "$certificate" ] || continue
        printf '\n'
        cat "$certificate"
    done
} > "$bundle"
chmod 644 "$bundle"
mv -f "$bundle" "$output"
