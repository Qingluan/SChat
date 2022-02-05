CC=gcc
CFLAGS=-I. -lpthread
ODIR=.
LDIR =../lib
RELEASE = .

OBJ = $(RELEASE)/libschat.a

%.a:
	go build -o $(RELEASE)/libschat.a -buildmode=c-archive $(ODIR)/main.go

test: $(OBJ)
	$(CC) -o $@ main.c  libschat.a $(CFLAGS)
	

.PHONY: clean

clean:
	rm test ;rm rm $(RELEASE)/*.a ; rm $(ODIR)/libschat.h; rm $(RELEASE)/libschat; go clean -cache