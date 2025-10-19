# clivekit

> The minimalist Livekit library in C language with E2E encryption (build from Go)

## Dependencies

1. Go language `version >= 1.24.6`
2. Linker flags: `-lsoxr -lopus -lopusfile`

## Interface functions

```c
clivekit_error_type clivekit_connect_to_room(char* room_desc, clivekit_connect_info conn_info);
clivekit_error_type clivekit_disconnect_from_room(char* room_desc);

clivekit_error_type clivekit_read_data_from_room(char* room_desc, clivekit_data_packet* data_packet);
clivekit_error_type clivekit_write_data_to_room(char* room_desc, clivekit_data_type data_type, char* data, size_t data_size);

clivekit_error_type clivekit_add_rx_key_for_room(char* room_desc, char* ident, char* rx_key);
clivekit_error_type clivekit_del_rx_key_for_room(char* room_desc, char* ident);
clivekit_error_type clivekit_set_tx_key_for_room(char* room_desc, char* tx_key);
```

## Build library

```bash
$ make build
```

## Example use

Install and run livekit-server
```bash
$ make install-livekit-server
$ make run-livekit-server
```

Terminal 1
```bash
$ cd examples/text
$ make run-publisher
write hello0
write hello1
write hello2
write hello3
write hello4
...
```

Terminal 2
```bash
$ cd examples/text
$ make run-subscriber
publisher - 1 - 7 - hello0 (1)
publisher - 1 - 7 - hello1 (2)
publisher - 1 - 7 - hello2 (3)
publisher - 1 - 7 - hello3 (4)
publisher - 1 - 7 - hello4 (5)
...
```
