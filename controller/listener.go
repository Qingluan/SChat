package controller

// package controller

// import (
// 	"encoding/json"
// 	"fmt"
// 	"log"
// 	"sync"

// 	"github.com/asticode/go-astikit"
// 	"github.com/asticode/go-astilectron"
// )

// var (
// 	VersionAstilectron = "0.51.0"
// 	VersionElectron    = "11.4.3"
// )

// type Message struct {
// 	Oper string `json:"oper"`
// 	Data string `json:"data"`
// }

// type Handler func(message *Message) *Message
// type App struct {
// 	index       string
// 	l           *log.Logger
// 	application *astilectron.Astilectron
// 	window      *astilectron.Window
// 	handlers    map[string]Handler
// 	lock        sync.RWMutex
// }

// func NewApp(indexPage string) (app *App) {

// 	app = new(App)
// 	app.l = log.New(log.Writer(), log.Prefix(), log.Flags())

// 	// var err error
// 	var a, err = astilectron.New(app.l, astilectron.Options{
// 		AppName: "SecurMessage",
// 		// Asset:              Asset,
// 		// AssetDir:           AssetDir,
// 		VersionAstilectron: VersionAstilectron,
// 		VersionElectron:    VersionElectron,
// 	})
// 	app.application = a
// 	if err != nil {
// 		log.Fatal("err:", err)
// 	}
// 	app.index = indexPage
// 	return
// }

// func (app *App) Init() {
// 	// Add a listener on Astilectron
// 	app.application.On(astilectron.EventNameAppCrash, func(e astilectron.Event) (deleteListener bool) {
// 		log.Println("App has crashed")
// 		return
// 	})
// 	app.handlers = make(map[string]Handler)
// 	app.lock = sync.RWMutex{}
// }

// func (app *App) OnResize() {
// 	// Add a listener on the window
// 	app.window.On(astilectron.EventNameWindowEventResize, func(e astilectron.Event) (deleteListener bool) {
// 		log.Println("Window resized")
// 		return
// 	})
// }

// func (app *App) OnListener(oper string, call Handler) {
// 	app.lock.Lock()
// 	defer app.lock.Unlock()
// 	app.handlers[oper] = call
// }

// func (app *App) OpenDevTools() {
// 	app.window.OpenDevTools()
// }

// func (app *App) Start(debug bool) {

// 	app.application.HandleSignals()

// 	// Start
// 	if err := app.application.Start(); err != nil {
// 		app.l.Fatal(fmt.Errorf("main: starting astilectron failed: %w", err))
// 	}

// 	defer app.application.Close()

// 	var err error
// 	app.window, err = app.application.NewWindow(app.index, &astilectron.WindowOptions{
// 		Center: astikit.BoolPtr(true),
// 		Height: astikit.IntPtr(600),
// 		Width:  astikit.IntPtr(600),
// 	})

// 	if err != nil {
// 		log.Fatal("err:", err)
// 	}

// 	if app.window != nil {
// 		app.window.Create()
// 	}

// 	app.Init()
// 	if debug {
// 		app.OpenDevTools()
// 	}
// 	// Start astilectron
// 	// Blocking pattern
// 	// This will listen to messages sent by Javascript
// 	app.window.OnMessage(func(m *astilectron.EventMessage) interface{} {
// 		// Unmarshal
// 		s := new(Message)
// 		var msgstr string
// 		m.Unmarshal(&msgstr)
// 		fmt.Println(msgstr)
// 		err := json.Unmarshal([]byte(msgstr), s)
// 		if err != nil {
// 			app.l.Println("[font msg err:]", err)
// 			return nil
// 		}
// 		// oper =
// 		// Process message
// 		if call, ok := app.handlers[s.Oper]; ok {
// 			if msg := call(s); msg != nil {
// 				data, _ := json.Marshal(msg)
// 				app.window.SendMessage(string(data))
// 			}
// 		}
// 		return nil
// 	})

// 	app.OnListener("hello", func(message *Message) *Message {
// 		fmt.Println("============= start ==================")
// 		return message
// 	})

// 	app.application.Wait()

// }
