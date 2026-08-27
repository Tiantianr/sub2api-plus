#!/bin/sh
set -eu

image_binary=/app/sub2api
image_build_id_file=/app/.sub2api-image-build-id
state_dir=/app/.sub2api-update-state
trusted_build_id_file=$state_dir/image-build-id
runtime_dir=/app/data/.sub2api-runtime
runtime_binary=$runtime_dir/sub2api

runtime_binary_is_valid() {
    [ -f "$runtime_binary" ] && [ ! -L "$runtime_binary" ] && [ -x "$runtime_binary" ]
}

backup_binary_is_valid() {
    [ -f "$runtime_binary.backup" ] && [ ! -L "$runtime_binary.backup" ] && [ -x "$runtime_binary.backup" ]
}

seed_runtime_binary() {
    seed_tmp=$(mktemp -d "$runtime_dir/.image-seed.XXXXXX")

    trap 'rm -rf "$seed_tmp"' EXIT HUP INT TERM
    cp "$image_binary" "$seed_tmp/sub2api"
    chmod 0755 "$seed_tmp/sub2api"
    if [ -d "$runtime_binary" ] || [ -L "$runtime_binary" ]; then
        rm -rf "$runtime_binary"
    fi
    mv -f "$seed_tmp/sub2api" "$runtime_binary"
    rm -rf "$runtime_binary.backup"
    rmdir "$seed_tmp"
    trap - EXIT HUP INT TERM
}

prepare_runtime() {
    mode=$1
    mkdir -p "$runtime_dir"

    if [ "$mode" = preserve ] && ! runtime_binary_is_valid && backup_binary_is_valid; then
        rm -rf "$runtime_binary"
        mv -f "$runtime_binary.backup" "$runtime_binary"
    fi

    if { [ -e "$runtime_binary.backup" ] || [ -L "$runtime_binary.backup" ]; } && ! backup_binary_is_valid; then
        rm -rf "$runtime_binary.backup"
    fi

    if [ "$mode" = seed ] || ! runtime_binary_is_valid; then
        seed_runtime_binary
    fi

    runtime_binary_is_valid
}

commit_trusted_build_id() {
    state_tmp=$(mktemp "$state_dir/.image-build-id.XXXXXX")
    trap 'rm -f "$state_tmp"' EXIT HUP INT TERM
    printf '%s\n' "$image_build_id" > "$state_tmp"
    chmod 0600 "$state_tmp"
    if [ -d "$trusted_build_id_file" ] && [ ! -L "$trusted_build_id_file" ]; then
        rm -rf "$trusted_build_id_file"
    fi
    mv -f "$state_tmp" "$trusted_build_id_file"
    sync -f "$state_dir"
    trap - EXIT HUP INT TERM
}

if [ "${1:-}" = __prepare_runtime__ ]; then
    test "$(id -u)" -eq 1000
    test "$#" -eq 2
    prepare_runtime "$2"
    exit 0
fi

# The default entrypoint starts as container root only to prepare mounts and
# commit the trusted image identity. Runtime executable operations are delegated
# to uid 1000 before the application starts.
if [ "$(id -u)" != "0" ]; then
    printf '%s\n' 'sub2api Docker entrypoint must start as container root' >&2
    exit 1
fi

test -f "$image_build_id_file"
test ! -L "$image_build_id_file"
test -s "$image_build_id_file"
image_build_id=$(cat "$image_build_id_file")
case "$image_build_id" in
    ''|*[!0-9a-f]*) exit 1 ;;
esac
test "${#image_build_id}" -eq 64

mkdir -p /app/data "$state_dir"
chown root:root "$state_dir"
chmod 0700 "$state_dir"
# Preserve support for read-only child mounts such as config.yaml:ro.
chown -R sub2api:sub2api /app/data 2>/dev/null || true
if ! su-exec sub2api /bin/sh -c 'test -w /app/data'; then
    printf '%s\n' 'sub2api data directory is not writable by uid 1000' >&2
    exit 1
fi

trusted_build_id=
if [ -f "$trusted_build_id_file" ] && [ ! -L "$trusted_build_id_file" ]; then
    trusted_build_id=$(cat "$trusted_build_id_file")
fi

prepare_mode=preserve
if [ "$trusted_build_id" != "$image_build_id" ]; then
    prepare_mode=seed
fi

su-exec sub2api "$0" __prepare_runtime__ "$prepare_mode"
if [ "$prepare_mode" = seed ]; then
    # Flush the selected runtime filesystem before committing its trusted
    # identity on the separate state filesystem.
    sync -f "$runtime_binary"
    commit_trusted_build_id
fi

# Preserve the image's existing command contract while executing the writable,
# persisted binary as the non-root application user.
if [ "$#" -eq 0 ]; then
    set -- "$runtime_binary"
elif [ "$1" = "$image_binary" ]; then
    shift
    set -- "$runtime_binary" "$@"
elif [ "${1#-}" != "$1" ]; then
    set -- "$runtime_binary" "$@"
fi

exec su-exec sub2api "$@"
