#!/bin/bash

### for x86 test 
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 CC=$(go env GOROOT)/misc/ios/clangwrap.sh CGO_CFLAGS="-fembed-bitcode" \
    go build -buildmode=c-archive -tags ios -o libSChat_amd64.a main.go


mkdir -p lib

mv libSChat_amd64* ./lib/