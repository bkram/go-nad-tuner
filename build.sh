#!/bin/bash
#Linux builds
ARCHS="amd64 386 arm64 arm mips64 mips"
MODULE=nad-tuner.go

mkdir builds

for arch in ${ARCHS}; do
    echo Build ${MODULE} for "${arch}", creating: "builds/${arch}/${MODULE}"
    GOARCH="${arch}" go build -o "builds/${arch}/" ${MODULE}
done

