package controller

type ChatRoom struct {
	IP       string
	nowMsgTo string
	Server   *Vps
}

func NewChatRoom(sshstr string) (chat *ChatRoom) {
	chat = new(ChatRoom)
	chat.Server = Parse(sshstr)
	chat.IP = chat.Server.IP
	return chat
}

// func (chat *ChatRoom) MsgTo(name string) *Message {
// 	msg := new(Message)
// 	chat.nowMsgTo = name

// 	return msg
// }
