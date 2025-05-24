#!/bin/bash
# build.sh

# Linux Intel 64-bit
GOOS=linux GOARCH=amd64 go build -o dist/music-artwork-embedder-linux-amd64 main.go

# Windows Intel 64-bit
GOOS=windows GOARCH=amd64 go build -o dist/music-artwork-embedder-windows-amd64.exe main.go

# macOS Intel 64-bit
GOOS=darwin GOARCH=amd64 go build -o dist/music-artwork-embedder-macos-intel main.go

# macOS Apple Silicon (参考)
GOOS=darwin GOARCH=arm64 go build -o dist/music-artwork-embedder-macos-arm64 main.go

echo "ビルド完了！"