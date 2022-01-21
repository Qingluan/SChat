package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"Chat/controller"

	"github.com/fatih/color"
)

func Input(msg string) string {
	read := bufio.NewReader(os.Stdin)
	fmt.Print(color.New(color.FgHiCyan).Sprint(msg))
	line, _, _ := read.ReadLine()
	return string(line)
}

func main() {
	page := "https://localhost:37777/ccc"
	ssh := ""
	sendToName := ""
	name := ""
	// R := false
	flag.StringVar(&page, "p", "resources/pages/index.html", "set page")
	flag.StringVar(&sendToName, "s", "", "set name")
	flag.StringVar(&name, "u", "a", "set user name")
	flag.StringVar(&ssh, "H", "://115.236.8.148:50022/docker-hub", "set page")
	// flag.BoolVar(&R, "r", false, "true to send ")
	flag.Parse()

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
		line := color.New(color.FgGreen).Sprintf("[%s]", msg.From)
		line += color.New(color.FgYellow).Sprintf(" -- %s\n\t", msg.Date)
		line += color.New(color.FgHiWhite, color.Bold).Sprintln(msg.Data)
		fmt.Print(line, color.New(color.FgHiCyan).Sprint("\nsend msg or cmd $ls/$ >"))

	})

	for {
		// 	fmt.Print(color.New(color.FgHiCyan).Sprintf("(%s)>", user))

		// fmt.Scanln(&line)
		line := Input("send msg or cmd $ls/$ >")
		if strings.HasPrefix(line, "$") {
			if line == "$ls" {
				for _, user := range chat.Contact() {
					fmt.Println(user.Name, user.Last())
				}
			} else if line == "$files" {
				for _, f := range chat.CloudFiles() {
					fmt.Println("[file]", f)
				}
				fmt.Println()
			} else if strings.HasPrefix(line, "$get") {
				name := strings.TrimSpace(strings.SplitN(line, "$get", 2)[1])
				chat.GetFile(name)
			} else if strings.HasPrefix(line, "$put") {
				name := strings.TrimSpace(strings.SplitN(line, "$put", 2)[1])
				if err := chat.SendFile(name); err != nil {
					log.Println("send file er:", err)
				}

			} else {
				chat.TalkTo(line[1:])
				line = ""
			}
		} else {
			chat.Write(line)
			line = ""
		}

	}
	// chat.Server.Init()
	// chat.Server.OnMessage(func(from, msg string, date time.Time) {

	// 	line := color.New(color.FgGreen).Sprintf("[%s]", from)
	// 	line += color.New(color.FgYellow).Sprintf(" -- %s\n\t", date.Format("2006年1月2号 15:04:05"))
	// 	line += color.New(color.FgHiWhite, color.Bold).Sprintln(msg)
	// 	fmt.Println(msg)
	// })
	// // line := ""
	// users, err := chat.Server.Contact()

	// if err != nil {
	// 	log.Fatal("contanct err:", err)
	// }
	// for _, u := range users {
	// 	fmt.Println(u.Name, u.Acivte(), "login:", u.State)
	// }

	// ip, err := chat.Server.ContactTo(string(line))
	// if err != nil {
	// 	log.Fatal("can not send to", err)
	// } else {
	// 	fmt.Println(line, "'s ip is ", ip)
	// }
	// user := string(line)
	// for {
	// 	fmt.Print(color.New(color.FgHiCyan).Sprintf("(%s)>", user))

	// 	// fmt.Scanln(&line)
	// 	line, _, _ := read.ReadLine()
	// 	chat.Server.SendMsg(string(line))
	// }
	// key := base64.StdEncoding.EncodeToString([]byte("hello write this !!!"))
	// stream, err := controller.NewStreamWithAuthor("check-onea-fasfsdfsadfads")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// stream.EncryptFile("SshChat", "test1")
	// stream.DecryptFile("test1", "test2")
}
