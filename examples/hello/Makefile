.PHONY: default build build-clivekit
default: build 
build: build-clivekit
	gcc -o publisher publisher.c clivekit.a
	gcc -o subscriber subscriber.c clivekit.a
build-clivekit:
	make -C ../..
	cp ../../clivekit.h ../../clivekit.a .
