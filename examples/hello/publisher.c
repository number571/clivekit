#include <stdio.h>
#include <unistd.h>
#include "clivekit.h" 

int main() {
    char room_id[LIVEKIT_DESC_SIZE];

    livekit_connect_info conn_info = {
        .host = "ws://localhost:7880",
        .api_key = "devkey",
        .api_secret = "secret",
        .room_name = "test",
        .ident = "c-go-sdk-1"
    };

    int status = livekit_connect_to_room(room_id, conn_info);
    if (status) {
        printf("connect failed\n");
        return 1;
    }
    
    printf("connect success\n");

    char msg[] = "hello_";
    while(1) {
        for(int i = 0; i < 10; i++) {
            msg[5] = '0'+(i%10);
            int status = livekit_write_data_to_room(room_id, "audio", msg, strlen(msg)+1);
            if (status) {
                printf("write failed\n");
                return 2;
            }
            printf("write %s\n", msg);
        }
        sleep(1);
    }

    livekit_disconnect_from_room(room_id);
    return 0;
}
