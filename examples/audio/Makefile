.PHONY: default build install-libsoundio build-libsoundio run-publisher run-subscriber
default: build 
build: 
	cp ../../clivekit.h ../../clivekit.a .
	gcc -o subscriber subscriber.c clivekit.a libsoundio/build/libsoundio.a -lasound -lm -lpulse -ljack
	gcc -o publisher publisher.c clivekit.a libsoundio/build/libsoundio.a -lasound -lm -lpulse -ljack
run-publisher: build
	./publisher x
run-subscriber: build
	./subscriber x
install-libsoundio: build-libsoundio
	git clone -b "2.0.1-7" https://github.com/andrewrk/libsoundio
	sed -i 's/cmake_minimum_required(VERSION 2.8.5)/cmake_minimum_required(VERSION 3.5)/' ./libsoundio/CMakeLists.txt
build-libsoundio:
	cd libsoundio && mkdir build && cd build && cmake .. && make
