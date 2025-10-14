# clivekit

> The minimalist Livekit library in C language (build from Go)

### Dependencies

1. Go language `version >= 1.24.6`
2. Linker flags: `-lsoxr -lopus -lopusfile`

### Build library

```bash
$ make build
```

### Examples

Install and run livekit-server
```bash
$ make install-livekit-server
$ ./bin/livekit-server --dev
```

Compile examples
```bash
$ cd examples/hello
$ make build
```

Terminal 1
```bash
$ cd examples/hello
$ ./publisher
write hello0
write hello1
write hello2
write hello3
write hello4
...
```

Terminal 2
```bash
$ cd examples/hello
$ ./subscriber
c-go-sdk-1 - audio - 7 - hello0 (1)
c-go-sdk-1 - audio - 7 - hello1 (2)
c-go-sdk-1 - audio - 7 - hello2 (3)
c-go-sdk-1 - audio - 7 - hello3 (4)
c-go-sdk-1 - audio - 7 - hello4 (5)
...
```
