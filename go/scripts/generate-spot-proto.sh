#!/bin/sh
set -eu

module_root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
proto_dir="$module_root/websocket/spot/protocol/proto"
options=""

for proto_file in "$proto_dir"/*.proto; do
    proto_name=${proto_file##*/}
    options="$options --go_opt=M$proto_name=github.com/k4k3ru-hub/mexc/go/websocket/spot/protocol/pb"
done

cd "$module_root"
# The arguments are assembled from the checked-in proto filenames above.
# shellcheck disable=SC2086
protoc -I "$proto_dir" --go_out=module=github.com/k4k3ru-hub/mexc/go:. $options "$proto_dir"/*.proto
