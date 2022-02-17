package controller

import (
	"fmt"

	"github.com/fatih/color"
)

func L(tmp string, args ...interface{}) {
	fmt.Println(color.New(color.FgGreen, color.Bold).Sprint("[+] "), color.New(color.FgCyan, color.Bold, color.Underline).Sprintf(tmp, args...))
}

func Ok(tmp string, args ...interface{}) {
	fmt.Println(color.New(color.FgGreen, color.Bold).Sprint("[⦿] "), color.New(color.FgCyan, color.Bold, color.Underline).Sprintf(tmp, args...))
}
