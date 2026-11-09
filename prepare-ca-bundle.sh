#!/bin/sh
set -eu

# The init container sees Alpine's original bundle at this path.
{
    cat /etc/ssl/certs/ca-certificates.crt
    for certificate in /usr/local/share/ca-certificates/*.crt; do
        printf '\n'
        cat "$certificate"
    done
} > /var/run/openmeter-ca/ca-certificates.crt
