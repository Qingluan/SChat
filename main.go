package main

import (
	"flag"
	"log"

	"SshChat/controller"
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
	// chat := controller.NewChatRoom(ssh)
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

	// read := bufio.NewReader(os.Stdin)
	// fmt.Print(color.New(color.FgHiCyan).Sprint("send to? >"))
	// line, _, _ := read.ReadLine()
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
	stream, err := controller.NewStreamWithAuthor("check-onea-fasfsdfsadfads")
	if err != nil {
		log.Fatal(err)
	}
	stream.EncryptFile("SshChat", "test1")
	stream.DecryptFile("test1", "test2")
}
