#!/usr/bin/env bash

set -e

export PATH=$PATH:.

if [ "${1#-}" != "$1" ]; then
    set -- envoy-init "$@"
fi

exec "$@"
