#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$repo_root"

fail() {
  printf 'docker runtime resources test failed: %s\n' "$1" >&2
  exit 1
}

normalized_file() {
  tr -d '\015' < "$1"
}

assert_line() {
  file=$1
  line=$2
  normalized_file "$file" | grep -Fqx "$line" || fail "$file is missing: $line"
}

assert_absent() {
  file=$1
  text=$2
  if normalized_file "$file" | grep -Fq "$text"; then
    fail "$file unexpectedly contains: $text"
  fi
}

test -s backend/resources/model-pricing/model_prices_and_context_window.json || \
  fail 'fallback pricing data is missing or empty'

assert_line Dockerfile 'COPY --from=backend-builder --chown=sub2api:sub2api /app/backend/resources /app/resources'
assert_line deploy/Dockerfile 'COPY --from=backend-builder --chown=sub2api:sub2api /app/backend/resources /app/resources'
assert_line .goreleaser.yaml '      - src: backend/resources/model-pricing/model_prices_and_context_window.json'
assert_line .goreleaser.yaml '        dst: resources/model-pricing'
assert_line .goreleaser.yaml '      - linux'
assert_line .goreleaser.yaml '      - arm64'
assert_absent .goreleaser.yaml '      - amd64'
assert_absent .goreleaser.yaml '      - darwin'
assert_absent .goreleaser.yaml '      - windows'
assert_absent deploy/install.sh '        x86_64)'
assert_absent deploy/install.sh '        darwin)'

printf 'docker runtime resources test passed\n'
