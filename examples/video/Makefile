.PHONY: default build run-publisher run-subscriber run-ffmpeg-writer run-ffmpeg-reader
default: build 
build:
	cp ../../clivekit.h ../../clivekit.a .
	gcc -o publisher publisher.c clivekit.a
	gcc -o subscriber subscriber.c clivekit.a
run-publisher: build
	./publisher
run-subscriber: build
	./subscriber
run-ffmpeg-reader:
	rm -f reader_pipe.ts && mkfifo reader_pipe.ts
	ffplay -probesize 100000 -analyzeduration 100000 -fflags nobuffer -flags low_delay -framedrop -f mpegts reader_pipe.ts
# ## raw launch reader without subscriber & publisher (used for tests)
# 	ffplay -fflags nobuffer -flags low_delay -f mpegts writer_pipe.ts
run-ffmpeg-writer:
	rm -f writer_pipe.ts && mkfifo writer_pipe.ts
	ffmpeg -y -i /dev/video0 -c:v libx264 -preset ultrafast -crf 30 -f mpegts writer_pipe.ts
