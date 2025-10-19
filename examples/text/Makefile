.PHONY: default build build-clivekit run-publisher run-subscriber
default: build 
build:
	cp ../../clivekit.h ../../clivekit.a .
	gcc -o publisher publisher.c clivekit.a
	gcc -o subscriber subscriber.c clivekit.a
run-publisher: build
	./publisher
run-subscriber: build
	./subscriber
