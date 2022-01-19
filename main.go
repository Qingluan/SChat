package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"SshChat/controller"

	"github.com/fatih/color"
)

func main() {
	page := "https://localhost:37777/ccc"
	ssh := ""
	sendToName := ""
	// R := false
	flag.StringVar(&page, "p", "resources/pages/index.html", "set page")
	flag.StringVar(&sendToName, "s", "", "set name")

	flag.StringVar(&ssh, "u", "my-name://115.236.8.148:50022/docker-hub", "set page")
	// flag.BoolVar(&R, "r", false, "true to send ")
	flag.Parse()

	// // astilog.FlagInit()
	chat := controller.NewChatRoom(ssh)
	chat.Server.Init()
	chat.Server.OnMessage(func(from, msg string, date time.Time) {

		line := color.New(color.FgGreen).Sprintf("[%s]", from)
		line += color.New(color.FgYellow).Sprintf(" -- %s\n\t", date.Format("2006年1月2号 15:04:05"))
		line += color.New(color.FgHiWhite, color.Bold).Sprintln(msg)
		fmt.Println(msg)
	})
	// line := ""
	read := bufio.NewReader(os.Stdin)
	fmt.Println(color.New(color.FgHiCyan).Sprint("send to? >"))
	line, _, _ := read.ReadLine()
	ip, err := chat.Server.ContactTo(string(line))
	if err != nil {
		log.Fatal("can not send to", err)
	} else {
		fmt.Println(line, "'s ip is ", ip)
	}
	user := string(line)
	for {
		fmt.Print(color.New(color.FgHiCyan).Sprintf("(%s)>", user))

		// fmt.Scanln(&line)
		line, _, _ := read.ReadLine()
		chat.Server.SendMsg(string(line))
	}
	// app := controller.NewApp(page)
	// // app.StartDevel()
	// app.Start(true)

	// Create windows
	// if err = w.Create(); err != nil {
	// 	l.Fatal(fmt.Errorf("main: creating window failed: %w", err))
	// }

	// // Blocking pattern
	// a.Wait()
}
