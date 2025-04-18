#!/usr/bin/env bash

# Generate a version string to be used by Otter.
#
# When $OTTER_RELEASE is true:
#   The produced version will equal the current commit's tag.
#   If the current commit does not have a tag attached to it,
#   then this script will fail.
#
# When $OTTER_RELEASE is unset:
#   The produced version will equal the latest git tag + a dev suffix.

set -euo pipefail

if [[ ${OTTER_RELEASE:-} == "true" ]]; then
    VERSION=$(git tag --points-at "$(git rev-parse HEAD)" --sort=version:refname | head -n 1)

    if [[ -z $VERSION ]]; then
        echo "ERROR: ./scripts/version.sh: the current commit is not tagged"
        exit 1
    fi
else
    VERSION=$(git tag --sort=version:refname | tail -n 1)

    if [[ -z $VERSION ]]; then
        VERSION="v0.0.0"
    fi

    VERSION+="-dev+$(git rev-parse --short HEAD)"
fi

echo "$VERSION"
