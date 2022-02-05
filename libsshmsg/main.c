#include "libschat.h"
#include <string.h>
#include <unistd.h>

void
callback(Cmsg*msg){
    printf("%s say [%s]:\n %s\n",msg->From,msg->Date,msg->Data);
    free(msg);
    return;
}

int
main(const int argc, char*argv[]){
    printf("arg: %d \n",argc);
    char * defaultMsg = "linux://115.236.8.148:50022/docker-hub:Dark@2hub";
    if (argc > 1){
        printf("argv : %s\n", argv[1]);
        // exit(0);
        // strcpy(defaultMsg, argv[1]);
        
        InitChatRoom(argv[1]);
    }else{
        InitChatRoom(defaultMsg);
    }
    OnMessage(callback);
    Users * users = ListUsers();
    int i =0;
    // users->num
    for(i=0;i < users->num;i++){
        printf("[%d/%d] name: %s\n",i,users->num,(User *)(users->users +i)->Name);
    }
    printf("users num: %d\nUser no.3's info : %s\n", users->num,UserActive(&users->users[2]));
    if (UserTalkTo("lin3")){
        printf("ok\n");
        if (WriteMessage("hallo this message write from C lang !")){
            printf("Write Successful!");
        };
    }
    do{
        /* code */
        sleep(2);
    } while (1==1);
    
}