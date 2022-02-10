package main

import (
	"fmt"

	"github.com/Qingluan/SChat/controller"
)

func main() {
	test, _ := controller.NewStreamWithBase64Key("MTE1LjIzNi44LjE0ODo1MDAyMg==")
	fmt.Println(test.FlowDe("AAAAAAA5MTZFfg__"))
	fmt.Println(test.FlowDe("AAAAAAAAAAA5OTVuKgMlHw__"))
	fmt.Println(test.FlowDe("AAAAADkxNkU_"))

}
