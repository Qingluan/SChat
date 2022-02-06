package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"Chat/controller"

	"github.com/c-bata/go-prompt"
	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
)

var (
	MainMenue = []string{
		"Choose User to Talk",
		"Download Files from clound",
		"Send File to other",
		"Set time to clear my data",
		"Quit SSH Msg",
	}

	I_CONTACT = 0
	I_DOWN    = 1
	I_SEND    = 2
	I_CLEAR   = 3
	I_EXIT    = 4
)

func Input(msg string) string {
	read := bufio.NewReader(os.Stdin)
	fmt.Print(color.New(color.FgHiCyan).Sprint(msg))
	line, _, _ := read.ReadLine()
	return string(line)
}

var (
	promptlabel = "[no user talk] >"
)

func main() {
	ssh := ""
	sendToName := ""
	name := ""
	Pass := ""
	Home := ""
	// R := false
	flag.StringVar(&sendToName, "s", "", "set name")
	flag.StringVar(&name, "u", "a", "set user name")
	flag.StringVar(&ssh, "H", "://115.236.8.148:50022/docker-hub", "set page")
	flag.StringVar(&Pass, "P", "", "set password ")
	flag.StringVar(&Home, "home", "", "set password ")

	flag.Parse()
	if Pass != "" {
		ssh += ":" + Pass
	}
	if Home != "" {
		log.Println("Use New Home:", Home)
		controller.SetHome(Home)
	}
	// // astilog.FlagInit()
	chat, err := controller.NewChatRoom(name + ssh)
	if err != nil {
		log.Fatal(err)
	}
	err = chat.Login()
	if err != nil {
		log.Fatal(err)
	}
	chat.SetWacher(func(msg *controller.Message) {
		line := color.New(color.FgGreen).Sprintf("[%s] >", msg.From)
		line += color.New(color.FgYellow).Sprintf(" -- %s\n ", msg.Date)
		line += color.New(color.FgHiWhite, color.Bold).Sprintln(msg.Data)
		// fmt.Print(line, color.New(color.FgHiCyan).Sprint("\nsend msg or cmd $ls/$ >"))
		fmt.Print(line)
	})
	// cmd := ""
	// msg := ""
MAINLOOP:
	for {
		// 	fmt.Print(color.New(color.FgHiCyan).Sprintf("(%s)>", user))
		out := Repl(promptlabel, Datas{
			"/":         "to menu",
			"/user":     "show users to talk",
			"/hist":     "show history",
			"/file":     "show cloud files",
			"/down":     "downlaod file in clound",
			"/upload":   "upload file to other user",
			"/clear":    "set delay time to clear my data in remote",
			"/quit":     "quit ssh msger",
			"/newgroup": "create new group",
			"/allow":    "allow who join which group",
			"/join":     "join which group",
			"/ls":       "show contact and groups",
			"/del":      "remove group",
		})
		switch out {
		case "/":
			SelectMenu(chat)
		case "/user":
			SelectContact(chat)
		case "/ls":
			for _, u := range chat.Contact() {
				if strings.HasSuffix(u.Name, controller.GROUP_TAIL) {
					fmt.Println("[Group] ", u.Name)
				} else {
					fmt.Println(u.Name)
				}

			}

		case "/hist":
			ShowHist(chat)
		case "/file":
			ShowFiles(chat)
		case "/down":
			DownFile(chat)
		case "/upload":
			UploadFile(chat)
		case "/clear":
			SetDelayClear(chat)
			break MAINLOOP

		case "/quit":
			os.Exit(0)

		default:
			if strings.HasPrefix(out, "/newgroup") {
				gname := strings.TrimSpace(strings.SplitN(out, "/newgroup", 2)[1])
				if gname != "" {
					controller.SecurityCheckName(gname)
					chat.CreateGroup(gname)

				}
			} else if strings.HasPrefix(out, "/join") {
				gname := strings.TrimSpace(strings.SplitN(out, "/join", 2)[1])
				if gname != "" {
					controller.SecurityCheckName(gname)
					fmt.Println("try join ", gname)
					chat.JoinGroup(gname)
				}
			} else if strings.HasPrefix(out, "/allow") {
				gnameAndReq := strings.TrimSpace(strings.SplitN(out, "/allow", 2)[1])
				if gnameAndReq != "" {
					fs := strings.Fields(gnameAndReq)
					if len(fs) == 2 {
						controller.SecurityCheckName(fs[1])
						controller.SecurityCheckName(fs[0])
						log.Println("permit ", fs[1], "join", fs[0])
						chat.Permitt(fs[1], fs[0])
					}
				}
			} else if strings.HasPrefix(out, "/del") {
				gname := strings.TrimSpace(strings.SplitN(out, "/del", 2)[1])
				if gname != "" {
					controller.SecurityCheckName(gname)
					fmt.Println("try delete group ", gname)
					chat.RemoveGroup(gname)
				}
			} else {
				if strings.HasPrefix(out, "[") && strings.Contains(out, "]") {
					fs := strings.SplitN(out[1:], "]", 2)
					gname := strings.TrimSpace(fs[0])
					msg := strings.TrimSpace(fs[1])
					chat.WriteGroup(gname, msg)
				} else {
					chat.Write(out)

				}

			}
		}
		// fmt.Scanln(&line)
		// line := Input("send msg or cmd $ls/$ >")
		// if strings.HasPrefix(line, "$") {
		// 	if line == "$ls" {
		// 		for _, user := range chat.Contact() {
		// 			fmt.Println(user.Name, time.Since(user.Last()))
		// 		}
		// 	} else if line == "$files" {
		// 		for _, f := range chat.CloudFiles() {
		// 			fmt.Println("[file]", f)
		// 		}
		// 		fmt.Println()
		// 	} else if line == "$hist" {
		// 		fmt.Println("[history ] pulling")
		// 		chat.History()

		// 	} else if strings.HasPrefix(line, "$get") {
		// 		name := strings.TrimSpace(strings.SplitN(line, "$get", 2)[1])
		// 		chat.GetFile(name)
		// 	} else if strings.HasPrefix(line, "$put") {
		// 		name := strings.TrimSpace(strings.SplitN(line, "$put", 2)[1])
		// 		if err := chat.SendFile(name); err != nil {
		// 			log.Println("send file er:", err)
		// 		}
		// 	} else if strings.HasPrefix(line, "$clear") {
		// 		fs := strings.SplitN(line, "$clear", 2)
		// 		t, err := strconv.Atoi(fs[1])
		// 		if err != nil {
		// 			t = 5
		// 		}
		// 		chat.CloseWithClear(t)
		// 		fmt.Println("bye~~~", t, "second will clear all data")
		// 	} else {
		// 		chat.TalkTo(line[1:])
		// 		line = ""
		// 	}
		// } else {
		// 	chat.Write(line)
		// 	line = ""
		// }

	}

}

func SelectMenu(chat *controller.ChatRoom) string {

	prompt := promptui.Select{
		Label: "/ menu",
		Items: MainMenue,
	}

	a, _, err := prompt.Run()
	if err != nil {
		return ""
	}
	switch a {
	case I_CONTACT:
		SelectContact(chat)
	case I_DOWN:
		DownFile(chat)
	case I_SEND:
		UploadFile(chat)
	case I_EXIT:
		os.Exit(0)

	}
	return ""

}

func SelectList(label string, items []string) string {
	prompt := promptui.Select{
		Label: label,
		Items: items,
		Size:  10,
		Searcher: func(input string, index int) bool {
			return strings.Contains(items[index], input)
		},
	}

	a, _, err := prompt.Run()
	if err != nil {
		return ""
	}
	return items[a]
}

func SelectContact(chat *controller.ChatRoom) *controller.User {
	users := chat.Contact()
	userstr := []string{}
	for _, u := range users {
		userstr = append(userstr, u.Name+"|"+time.Since(u.Last()).String())
	}
	prompt := promptui.Select{
		Label: "select user to talk",
		Items: userstr,
		Size:  10,
		Searcher: func(input string, index int) bool {
			return strings.Contains(userstr[index], input)
		},
	}

	a, _, err := prompt.Run()
	if err != nil {
		return nil
	}
	u := users[a]
	chat.TalkTo(u.Name)
	promptlabel = fmt.Sprintf("%s|(%s) >", chat.MyName, u.Name)
	return u
}

type Datas map[string]string

func Repl(label string, suggest Datas) string {
	return prompt.Input(label, func(d prompt.Document) (s []prompt.Suggest) {
		for k, v := range suggest {
			s = append(s, prompt.Suggest{
				Text:        k,
				Description: v,
			})
		}
		return prompt.FilterFuzzy(s, d.GetWordBeforeCursor(), true)
	})
}

func UploadFile(chat *controller.ChatRoom) {
	validate := func(input string) error {
		f, err := os.Stat(input)
		if err != nil {
			return errors.New("not exists path !")
		}
		if f.IsDir() {
			return errors.New("must be a file not dir!")
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    "how many second to delete my data:",
		Validate: validate,
	}
	result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}
	chat.SendFile(result)

}

func SetDelayClear(chat *controller.ChatRoom) {
	// fs := strings.SplitN(line, "$clear", 2)
	// 		t, err := strconv.Atoi(fs[1])
	// 		if err != nil {
	// 			t = 5
	// 		}
	validate := func(input string) error {
		_, err := strconv.ParseInt(input, 10, 64)
		if err != nil {
			return errors.New("Invalid number")
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    "how many second to delete my data:",
		Validate: validate,
	}
	result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}
	t, err := strconv.ParseInt(result, 10, 64)
	chat.CloseWithClear(int(t))

}

func ShowHist(chat *controller.ChatRoom) {
	chat.History()
}

func ShowFiles(chat *controller.ChatRoom) {
	for _, f := range chat.CloudFiles() {
		fmt.Println("[clound]", f)
	}
}

func DownFile(chat *controller.ChatRoom) {
	fs := chat.CloudFiles()
	f := SelectList("Download file ctrl -c to cancel", fs)
	if f != "" {
		chat.GetFile(f)
	}
}
