package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"SshChat/controller"
)

func main() {
	page := "https://localhost:37777/ccc"
	ssh := ""
	myname := ""
	R := false
	flag.StringVar(&page, "p", "resources/pages/index.html", "set page")
	flag.StringVar(&myname, "n", "my-name", "set name")

	flag.StringVar(&ssh, "s", "my-name://115.236.8.148:50022/docker-hub", "set page")
	flag.BoolVar(&R, "r", false, "true to send ")
	flag.Parse()

	// // astilog.FlagInit()
	chat := controller.NewChatRoom(ssh)

	chat.Server.Init()
	if R {
		chat.Server.OnMessage(func(msg string, date time.Time) {
			fmt.Println("----- recv :\n\t", msg, "\n", date, "\n-------------- end ------\n")
		})
	} else {
		go func() {
			chat.Server.OnMessage(func(msg string, date time.Time) {
				fmt.Println("----- recv :\n\t", msg, "\n", date, "\n-------------- end ------\n")
			})
		}()
		remote, err := chat.Server.ContactTo(myname)
		if err != nil {
			log.Println(err)
		}
		fmt.Println("remote logined : ", remote)
		for {
			time.Sleep(2 * time.Second)
			err := chat.Server.SendMsg("my say ...... : \n\t" + time.Now().String())
			if err != nil {
				break
			}
		}
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
