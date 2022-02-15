#!/bin/bash

# export CFLAGS="-arch arm64 -miphoneos-version-min=13.0 -isysroot "$(xcrun -sdk iphoneos --show-sdk-path) 

# CGO_ENABLED=1 GOARCH=arm64 CC="clang $CFLAGS" go build -x -buildmode=c-archive -o ./libSChat_arm64.a ./main.go


CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 CC=$(go env GOROOT)/misc/ios/clangwrap.sh CGO_CFLAGS="-fembed-bitcode" \
    go build -buildmode=c-archive -tags ios -o libSChat_arm64.a main.go

mkdir -p lib 
mv libSChat_arm64* ./lib/
cp bridge.h ./lib/