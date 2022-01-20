package controller

type ChatRoom struct {
	IP       string
	nowMsgTo string
	vps      *Vps
	stream   *Stream
}

func NewChatRoom(sshstr string) (chat *ChatRoom, err error) {
	chat = new(ChatRoom)
	chat.vps = Parse(sshstr)
	chat.IP = chat.vps.IP
	chat.stream, err = NewStreamWithAuthor(chat.vps.name)
	return chat, err
}

func (chat *ChatRoom) TalkTo(name string) {
	chat.vps.ContactTo(name)
	chat.vps.SendKey(chat.stream.Key)
}

// func (chat *ChatRoom) MsgTo(name string) *Message {
// 	msg := new(Message)
// 	chat.nowMsgTo = name

// 	return msg
// }
