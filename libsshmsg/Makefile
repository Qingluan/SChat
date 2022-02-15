CC=gcc
CFLAGS=-I. -lpthread
ODIR=.
LDIR =../lib
RELEASE = .

OBJ = $(RELEASE)/libschat.a

IOSLIB = $(RELEASE)/libSChat_arm64.a

IOSLIBTEST = $(RELEASE)/libSChat_amd64.a

	

.PHONY: clean
clean:
	rm test ;rm rm $(RELEASE)/*.a ; rm $(ODIR)/libschat.h; rm $(RELEASE)/libschat; go clean -cache
	rm -rf lib
	rm -rf libtest

libschat.a:
	go build -o $(RELEASE)/libschat.a -buildmode=c-archive $(ODIR)/main.go


test: test.c $(OBJ)
	$(CC) -v -o $@  $^ $(CFLAGS) -framework CoreFoundation -framework Security

build:$(OBJ)


libSChat_arm64.a:
	./build_ios.sh

libSChat_amd64.a:
	./build_iostest.sh

ios: $(IOSLIB)

iostest: $(IOSLIBTEST)

all: build
