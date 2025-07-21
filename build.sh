#!/bin/bash

set -e  # Exit on any error

echo "Installing and building frontend..."

cd webui/react-ui
bun install
bun run build

echo "Building Go backend..."
cd ../../
go build -tags netgo -ldflags '-s -w' -o app
