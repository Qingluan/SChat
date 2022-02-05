CC=gcc
CFLAGS=-I. -lpthread
ODIR=.
LDIR =../lib
RELEASE = .

libschat.a:
	go build -o $(RELEASE)/libschat.a -buildmode=c-archive $(ODIR)/main.go

test: 
	$(CC) -o $@ main.c  libschat.a $(CFLAGS)
	

.PHONY: clean

clean:
	rm $(ODIR)/*.a ; rm $(ODIR)/libschat.h; rm $(RELEASE)/libschat;