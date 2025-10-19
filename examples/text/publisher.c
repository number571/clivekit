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
        .ident = "publisher"
    };

    int status = clivekit_connect_to_room(room_desc, conn_info);
    if (status) {
        printf("connect failed\n");
        return 1;
    }
    
    printf("connect success\n");

    char tx_key[CLIVEKIT_SIZE_ENCKEY] = {0};
    status = clivekit_set_tx_key_for_room(room_desc, tx_key);
    if (status) {
        printf("set tx_key\n");
        return 2;
    }

    char msg[] = "hello_";
    while(1) {
        for(int i = 0; i < 10; i++) {
            msg[5] = '0'+(i%10);
            int status = clivekit_write_data_to_room(room_desc, CLIVEKIT_DTYPE_TEXT, msg, strlen(msg)+1);
            if (status) {
                printf("write failed\n");
                return 3;
            }
            printf("write %s\n", msg);
        }
    }

    clivekit_disconnect_from_room(room_desc);
    return 0;
}
