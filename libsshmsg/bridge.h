#include <stdio.h>
#include <stdlib.h>

#define BOOL int
#define TRUE 1
#define FALSE 0

typedef struct Cmsg{
	BOOL Crypted;
	char* Date;
	char* From;
	char* Data;
}Cmsg;

typedef struct User{
	BOOL State;
	char* Name;
	char* LastActive;
}User;

typedef struct Users{
	int num;
	int used;
	User* users;
} Users;


typedef void (*MsgCallback)(Cmsg* cmsg);
typedef int(*Call)(char* event);

// int bridge_call(Call cb,char* str);
// Cmsg* create_cmsg(char * msg, char * from , char * date, int crypted);
// User* create_user(char *name,char* last,BOOL state );
// Users* create_users(int num);
// int add_user(Users * users,  User * user);
// int set_on_message(MsgCallback call ,Cmsg* msg );



static
int 
bridge_call(Call cb,char* str)
{
	return cb(str);
}

static
Cmsg*
create_cmsg(char * msg, char * from , char * date, int crypted){
	Cmsg* cmsg = (Cmsg *) calloc(1,sizeof(Cmsg));
	cmsg->Date = date;
	cmsg->Data = msg;
	cmsg->From = from;
	cmsg->Crypted = crypted;
	return cmsg;
}

static
User*
create_user(char *name,char* last,BOOL state ){
	User * user = (User *) calloc(1, sizeof(User));
	user->Name = name;
	user->LastActive = last;
	user->State = state;
	return user;
}


static 
Users*
create_users(int num){
	Users * users_ptr = (Users *) calloc(num, sizeof(Users));
	users_ptr->users = (User*) calloc(num, sizeof(User));
	users_ptr->num = num;
	return users_ptr;
}

static
int 
add_user(Users * users,  User * user){
	int used = users->used;
	users->users[used].Name = user->Name;
	users->users[used].LastActive = user->LastActive;
	users->users[used].State = user->State;
	users->used ++;
	return users->used;
}


static
int
set_on_message(
    MsgCallback call ,Cmsg* msg ){
	call(msg);
	return 0;
}