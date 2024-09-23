#!/bin/bash

echo "Building linux-amd64"

# Are we in macOS?
if [[ "$(uname -s)" == "Darwin" ]]; then
    GOOS=linux GOARCH=amd64 CGO_ENABLED=1 CC="x86_64-linux-musl-gcc" go build -C server/cmd/lsp -o ../../bin/c3lsp
else
    GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -C server/cmd/lsp -o ../../bin/c3lsp
fi

chmod +x server/bin/c3lsp
cd server/bin || exit
tar -czvf ./linux-amd64-c3lsp.tar.gz c3lsp

echo "linux-amd64 built"