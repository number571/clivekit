#include <stdio.h>
#include <unistd.h>
#include "clivekit.h" 

int main() {
    char room_desc[LIVEKIT_DESC_SIZE];

    livekit_connect_info conn_info = {
        .host = "ws://localhost:7880",
        .api_key = "devkey",
        .api_secret = "secret",
        .room_name = "test",
        .ident = "c-go-sdk-2"
    };

    int status = livekit_connect_to_room(room_desc, conn_info);
    if (status) {
        printf("connect failed\n");
        return 1;
    }

    printf("connect success\n");

    int count = 0;

    livekit_data_packet data_packet;
    while(1){
        int status = livekit_read_data_from_room(room_desc, &data_packet);
        printf("%s - %s - %zu - %s (%d)\n", data_packet.ident, data_packet.topic, data_packet.payload_size, data_packet.payload, ++count);
    }

    livekit_disconnect_from_room(room_desc);
    return 0;
}
