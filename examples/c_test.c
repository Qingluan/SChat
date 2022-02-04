#include <stdio.h>
#include <string.h>
#include "atest.h"

#define BOOL int
#define TRUE 1
#define FALSE 0

GoString
GoStr(char *cstr, int cstrlen) {
    GoString a;
    a.p = cstr;
    a.n = cstrlen;
    return a;
}

BOOL test(char * str1){
    if (strlen(str1) < 10){
        printf("false %s \n",str1);
        return TRUE;
    }else{
        printf("true %s \n",str1);
        return FALSE;
    }
}

int 
main(){
    char * a = "test hello world!";
    int(*ptr)(char* a) = test; 
    GoString str = GoStr(a, strlen(a));
    test2("hello world!");
    // test1(str, ptr);
    // test("test1");
    // ptr("test 2 hell o workld ");
    test1("test adfsa sdf", ptr);
    return 0;
}