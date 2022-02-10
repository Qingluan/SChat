package controller

import (
	"fmt"
	"log"
	"time"
)

type User struct {
	Name       string `json:"name"`
	State      bool   `json:"login"`
	LastActive string `json:"last"`
}

func (u *User) Last() (t time.Time) {
	t, err := time.Parse(TIME_TMP, u.LastActive)
	if err != nil {
		log.Println("parse last active time failed !", u.LastActive, err)
		t, _ = time.Parse(TIME_TMP, TIME_TMP)
	}
	return
}

func (u *User) Acivte() string {
	raw := time.Since(u.Last())
	msg := ""

	if raw.Hours() > 1 {
		if raw.Hours() >= 24 {
			day := int(raw.Hours() / 24)
			hour := int(raw.Hours()) % 24
			msg += fmt.Sprintf("%d天", int(day))
			msg += fmt.Sprintf(" %d小时", int(hour))
		} else {
			msg += fmt.Sprintf(" %d小时", int(raw.Hours()))

		}
	}
	if raw.Minutes() > 1 {
		msg += fmt.Sprintf(" %d分", int(raw.Minutes()))
	}

	if raw.Seconds() > 1 {
		msg += fmt.Sprintf(" %d秒前", int(raw.Seconds()))
	}
	if msg == "" {
		return "现在"
	}
	return msg
}

func (u *User) IfAlive() bool {
	return !time.Now().After(u.Last().Add(15 * time.Second))
}
