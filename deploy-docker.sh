#!/bin/bash
set -euo pipefail

# Build (and optionally push) the panahon-api container image.
# Usage: ./deploy-docker.sh -U yohlime -p

ENGINE="podman"
REGISTRY="ghcr.io"
PLATFORM="linux/amd64"
ARCH=""
TAG_SUFFIX=""
USERNAME=""
IMG_NAME="panahon-api"
PUSH_BUILD=false
MULTIARCH=false
REV_LENGTH=7

usage() {
  cat <<EOF
Usage: $0 [options]

Builds the image locally by default. Add -p to publish it.

Options:
  -n, --image-name=IMAGE  Image name (default: panahon-api)
  -U, --user=USERNAME     Container registry username (required to push)
  -p, --push              Push image(s) to the registry
  -D, --docker            Use docker instead of podman
  --registry=REGISTRY     Registry hostname (default: ghcr.io)
  --arch=ARCH             Target architecture: amd64, arm64, or linux/amd64
  --multiarch             Build a multi-arch manifest
  --platform=PLATFORMS    Comma-separated platforms (default: linux/amd64,linux/arm64)
  --rev-length=N          Short git revision length (default: 7)
  --suffix=SUFFIX         Tag suffix appended to the revision tag
  -h, --help              Show this help and exit

Examples:
  ./deploy-docker.sh                          # local build
  ./deploy-docker.sh -U yohlime -p            # build and push
  ./deploy-docker.sh -U yohlime --arch=arm64 -p
EOF
  exit 1
}

while [[ $# -gt 0 ]]; do
  case $1 in
  -p | --push) PUSH_BUILD=true; shift ;;
  -D | --docker) ENGINE="docker"; shift ;;
  -U)
    USERNAME="${2:?-U requires a username}"
    shift 2 ;;
  --user=*) USERNAME="${1#*=}"; shift ;;
  -n | --image-name=*) IMG_NAME="${1#*=}"; shift ;;
  --registry=*) REGISTRY="${1#*=}"; shift ;;
  --arch=*) ARCH="${1#*=}"; shift ;;
  --multiarch) MULTIARCH=true; shift ;;
  --platform=*) PLATFORM="${1#*=}"; shift ;;
  --rev-length=*) REV_LENGTH="${1#*=}"; shift ;;
  --suffix=*) TAG_SUFFIX="${1#*=}"; shift ;;
  -h | --help) usage ;;
  *) echo "Unknown argument: $1"; usage ;;
  esac
done

if [[ "$MULTIARCH" == true && -n "$ARCH" ]]; then
  echo "Error: --arch and --multiarch are mutually exclusive."
  exit 1
fi

ARCH_SUFFIX=""
if [[ "$MULTIARCH" == false ]]; then
  if [[ -n "$ARCH" ]]; then
    if [[ "$ARCH" == */* ]]; then PLATFORM="$ARCH"; else PLATFORM="linux/${ARCH}"; fi
    # Only suffix the tags for an explicit cross-build so `:latest` isn't clobbered.
    ARCH_SUFFIX="${PLATFORM//\//-}"
  else
    case "$(uname -m)" in
    x86_64 | amd64) PLATFORM="linux/amd64" ;;
    aarch64 | arm64) PLATFORM="linux/arm64" ;;
    *) PLATFORM="" ;;
    esac
  fi
fi

BASE_TAG="${REGISTRY}/${USERNAME:-yohlime}/${IMG_NAME}"

if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  REV="$(git rev-parse --short="$REV_LENGTH" HEAD)"
else
  REV="$(date +%Y%m%d)"
fi

TAG_SUFFIXES=""
[[ -n "$TAG_SUFFIX" ]] && TAG_SUFFIXES="-${TAG_SUFFIX}"
[[ -n "$ARCH_SUFFIX" ]] && TAG_SUFFIXES="${TAG_SUFFIXES}-${ARCH_SUFFIX}"

VER_TAG="${REV}${TAG_SUFFIXES}"
LATEST_TAG="${BASE_TAG}:latest${TAG_SUFFIXES}"
FULL_TAG="${BASE_TAG}:${VER_TAG}"

if [[ "$MULTIARCH" == true ]]; then
  if [[ "$ENGINE" == "docker" ]]; then
    if ! docker buildx inspect buildx-builder &>/dev/null; then
      docker buildx create --use --name buildx-builder
    else
      docker buildx use buildx-builder
    fi
    docker buildx inspect --bootstrap

    PUSH_FLAG=""
    [[ "$PUSH_BUILD" == true ]] && PUSH_FLAG="--push"

    echo "Multi-arch build for ${PLATFORM}..."
    docker buildx build --platform "${PLATFORM}" \
      "${PUSH_FLAG}" \
      -t "${LATEST_TAG}" \
      -t "${FULL_TAG}" \
      .
  else
    podman manifest rm "${FULL_TAG}" 2>/dev/null || true
    podman manifest create "${FULL_TAG}"

    IFS=',' read -ra ARCHES <<<"${PLATFORM}"
    for p in "${ARCHES[@]}"; do
      echo "Building ${p}..."
      podman build --platform "${p}" --manifest "${FULL_TAG}" .
    done

    podman tag "${FULL_TAG}" "${LATEST_TAG}"

    if [[ "$PUSH_BUILD" == true ]]; then
      podman manifest push --all "${FULL_TAG}" "docker://${FULL_TAG}"
      podman manifest push --all "${LATEST_TAG}" "docker://${LATEST_TAG}"
    else
      echo "Built local manifest(s) (not pushed): ${LATEST_TAG}  ${FULL_TAG}"
    fi
  fi
else
  PLATFORM_ARGS=()
  [[ -n "$PLATFORM" ]] && PLATFORM_ARGS=(--platform "${PLATFORM}")

  echo "Building ${PLATFORM:-host platform} with ${ENGINE}..."
  "${ENGINE}" build "${PLATFORM_ARGS[@]}" -t "${LATEST_TAG}" -t "${FULL_TAG}" .

  if [[ "$PUSH_BUILD" == true ]]; then
    echo "Pushing..."
    "${ENGINE}" push "${LATEST_TAG}"
    "${ENGINE}" push "${FULL_TAG}"
  else
    echo "Built local image(s) (not pushed): ${LATEST_TAG}  ${FULL_TAG}"
  fi
fi
