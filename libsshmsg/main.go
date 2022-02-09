package main

/*
#cgo CFLAGS: -g -Wall -Iinclude
#cgo LDFLAGS: -Wl,--allow-multiple-definition
#include "bridge.h"
*/
import "C"
import (
	"fmt"
	"log"
	"sync"

	"github.com/Qingluan/SChat/controller"
)

var (
	GlobalChat *controller.ChatRoom
	lock       = sync.RWMutex{}
)

//export InitChatRoom
func InitChatRoom(loginInof *C.char, home *C.char) {
	sshstr := C.GoString(loginInof)
	shome := C.GoString(home)

	chat, err := controller.NewChatRoom(sshstr, shome)
	if err != nil {
		log.Fatal(err)
	}
	err = chat.Login()
	if err != nil {
		log.Fatal(err)
	}
	GlobalChat = chat
	fmt.Println("Init Ok")
}

//export ListUsers
func ListUsers() *C.Users {
	// cuserarray := []*C.User{}
	users := GlobalChat.Contact()
	cusers := C.create_users(C.int(len(users)))

	// subusers := (*[1 << 30]C.User)(unsafe.Pointer(&cusers.users))[:len(users)]
	for _, u := range users {
		b := 0
		if u.State {
			b = 1
		}
		cu := C.create_user(C.CString(u.Name), C.CString(u.LastActive), C.BOOL(b))
		// cuserarray = append(cuserarray, cu)
		// subusers[n].Name = cu.Name
		// subusers[n].LastActive = cu.LastActive
		// subusers[n].State = cu.State

		// cusers.users = cuserarray[n]
		C.add_user(cusers, cu)
	}

	// cusers.users = cuserarray[0]
	return cusers
}

//export OnMessage
func OnMessage(call C.MsgCallback) {
	if GlobalChat != nil {
		GlobalChat.SetWacher(func(msg *controller.Message) {
			i := 0
			if msg.Crypted {
				i = 1
			}
			// cmsg := &C.Cmsg{
			// 	Data:    C.CString(msg.Data),
			// 	Date:    C.CString(msg.Date),
			// 	From:    C.CString(msg.From),
			// 	Crypted: i,
			// }
			cmsg := C.create_cmsg(C.CString(msg.Group), C.CString(msg.Data), C.CString(msg.From), C.CString(msg.Date), C.BOOL(i), C.int(msg.Tp))
			C.set_on_message(call, cmsg)
		})
	}
}

//export UserActive
func UserActive(cuser *C.User) *C.char {
	user := &controller.User{
		Name:       C.GoString(cuser.Name),
		LastActive: C.GoString(cuser.LastActive),
	}
	if cuser.State == 1 {
		user.State = true
	}
	return C.CString(user.Acivte())
}

// UserExists
func UserExists(name *C.char) int {
	str := C.GoString(name)
	for _, u := range GlobalChat.Contact() {
		if u.Name == str {
			return 1
		}
	}
	return 0
}

//export UserTalkTo
func UserTalkTo(name *C.char) int {
	if UserExists(name) == 1 {
		GlobalChat.TalkTo(C.GoString(name))
		return 1
	} else {
		return 0
	}
}

//export WriteMessage
func WriteMessage(msg *C.char) int {
	str := C.GoString(msg)
	if GlobalChat != nil {
		GlobalChat.Write(str)
		return len(str)
	}
	return 0
}

//export WriteGroupMessage
func WriteGroupMessage(group *C.char, msg *C.char) int {
	str := C.GoString(msg)
	if GlobalChat != nil {
		GlobalChat.WriteGroup(C.GoString(group), str)
		return len(str)
	}
	return 0
}

//export ChatJoinGroup
func ChatJoinGroup(group *C.char) {
	str := C.GoString(group)
	if GlobalChat != nil {
		GlobalChat.JoinGroup(str)
	}
}

//export ChatAllowJoin
func ChatAllowJoin(group *C.char, username *C.char) {
	str := C.GoString(group)
	ustr := C.GoString(username)
	if GlobalChat != nil {
		GlobalChat.Permitt(str, ustr)
	}
}

//export ChatCreateGroup
func ChatCreateGroup(group *C.char) {
	str := C.GoString(group)
	if GlobalChat != nil {
		GlobalChat.CreateGroup(str)
	}
}

//export ChatCloudFiles
func ChatCloudFiles() *C.TmpFiles {
	var fs []string
	if GlobalChat != nil {
		fs = GlobalChat.CloudFiles()

	}
	cfs := C.create_files(C.int(len(fs)))

	for _, f := range fs {
		C.tmp_add_file(cfs, C.CString(f))
	}

	return cfs
}

func main() {

}
