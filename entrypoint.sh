#!/bin/sh

set -e

# Check if Pangolin environment variables are set
if [ -n "$PANGOLIN_ENDPOINT" ] && [ -n "$CLIENT_ID" ] && [ -n "$CLIENT_SECRET" ]; then
    # Run pangolin-cli up --attach with the provided credentials
    exec pangolin-cli up client --attach --id "$CLIENT_ID" --secret "$CLIENT_SECRET" --endpoint "$PANGOLIN_ENDPOINT" "$@"
fi

# Check if site (Newt) environment variables are set. SITE_ID/SITE_SECRET
# (along with PANGOLIN_ENDPOINT and any other newt flags/env vars) are read
# directly by `up site`, so nothing needs to be translated into flags here.
if [ -n "$SITE_ID" ] && [ -n "$SITE_SECRET" ]; then
    exec pangolin-cli up site "$@"
fi

# If no arguments provided, run pangolin-cli with default behavior
if [ $# -eq 0 ]; then
    exec pangolin-cli
fi

# If first arg is a flag (starts with -), prepend pangolin-cli
if [ "${1#-}" != "$1" ]; then
    exec pangolin-cli "$@"
fi

# If first arg is a shell or absolute path, execute as-is
case "$1" in
    sh|bash|/*)
        exec "$@"
        ;;
    *)
        # Otherwise, treat as a pangolin-cli subcommand
        exec pangolin-cli "$@"
        ;;
esac