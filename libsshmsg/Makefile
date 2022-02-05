CC=gcc
CFLAGS=-I. -lpthread
ODIR=.
LDIR =../lib
RELEASE = .

OBJ = $(RELEASE)/libschat.a


	

.PHONY: clean
clean:
	rm test ;rm rm $(RELEASE)/*.a ; rm $(ODIR)/libschat.h; rm $(RELEASE)/libschat; go clean -cache


%.a:
	go build -o $(RELEASE)/libschat.a -buildmode=c-archive $(ODIR)/main.go


test: test.c $(OBJ)
	$(CC) -o $@  $^ $(CFLAGS)


build:$(OBJ)

all: build
