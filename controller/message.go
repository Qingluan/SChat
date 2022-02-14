package controller

import (
	"fmt"
)

type Message struct {
	Date    string `json:"date"`
	Data    string `json:"data"`
	From    string `json:"from"`
	Group   string `json:"group"`
	Tp      int    `json:"tp"`
	To      string `json:"to"`
	Crypted bool   `json:"crypt"`
	Readed  string `json:"readed"`
}

var (
	MSG_HISTORY     = "message.history.txt"
	MSG_FILE        = "message.txt"
	MSG_HEART       = "heart"
	MSG_PRIVATE_KEY = "keys"
	MSG_FILE_ROOT   = "files"
	MSG_HELLO       = "info"
	MSG_TMP_FILE    = "tmp_file"
	ROOT            = "/tmp/SecureRoom"
	GROUP_TAIL      = "_GroUP"
	TMP             = "/tmp"

	MSG_CLIP_PREFIX = "[clipbroad]:"
	MsgIsNULL       = fmt.Errorf("msg is null!!")
)
