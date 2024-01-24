release:
    #!/usr/bin/env bash
    set -euo pipefail

    git checkout main > /dev/null 2>&1
    git diff-index --quiet HEAD || (echo "Git directory is dirty" && exit 1)

    version=v$(semver bump prerelease beta.. $(git describe --abbrev=0))

    echo "Detected version: ${version}"
    read -n 1 -p "Is that correct (y/N)? " answer
    echo

    case ${answer:0:1} in
        y|Y )
            echo "Tagging release with version ${version}"
        ;;
        * )
            echo "Aborting"
            exit 1
        ;;
    esac

    git tag -m "Release ${version}" $version
    git push origin $version
