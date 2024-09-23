#!/bin/bash

go build -C server/cmd/lsp -o ../../bin/c3lsp

OS="$(uname -s)"

if [[ "$OS" == "Linux" || "$OS" == "Darwin" ]]; then
    chmod +x server/bin/c3lsp
fi