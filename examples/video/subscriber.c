#include <stdio.h>
#include <unistd.h>
#include "clivekit.h" 

int main() {
    char room_desc[CLIVEKIT_SIZE_DESC];

    clivekit_connect_info conn_info = {
        .host = "ws://localhost:7880",
        .api_key = "devkey",
        .api_secret = "secret",
        .room_name = "test",
        .ident = "subscriber"
    };

    int status = clivekit_connect_to_room(room_desc, conn_info);
    if (status) {
        printf("connect failed\n");
        return 1;
    }

    printf("connect success\n");

    char rx_key[CLIVEKIT_SIZE_ENCKEY] = {0};
    status = clivekit_add_rx_key_for_room(room_desc, "publisher", rx_key);
    if (status) {
        printf("set rx_key\n");
        return 2;
    }

    FILE *reader_pipe = fopen("reader_pipe.ts", "wb");
    if (reader_pipe == NULL) {
        printf("fopen\n");
        return 3;
    }

    clivekit_data_packet data_packet;
    while(1){
        int status = clivekit_read_data_from_room(room_desc, &data_packet);
        if (status) {
            printf("read failed\n");
            return 3;
        }
        // printf("READ %zu\n", data_packet.payload_size);
        fwrite(data_packet.payload, sizeof(char),data_packet.payload_size, reader_pipe);
        fflush(reader_pipe);
    }

    fclose(reader_pipe);
    clivekit_disconnect_from_room(room_desc);
    return 0;
}
