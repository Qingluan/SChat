package main

/*
typedef int(*Call)(char* event);
static int bridge_call(Call cb,char* str)
{
	return cb(str);
}
*/
import "C"
import (
	"fmt"
)

// type Cdeal func(tes *C.char) C.int

//export test1
func test1(str *C.char, call C.Call) {
	// b := (*Cdeal)(unsafe.Pointer(&a))
	if C.bridge_call(call, str) > 0 {
		fmt.Println("Test ok!")
	} else {
		fmt.Println("test failed!!!")
	}
}

//export test2
func test2(str *C.char) bool {
	fmt.Println("test 1 func call")
	fmt.Println(C.GoString(str))
	return true
}

func main() {

}
