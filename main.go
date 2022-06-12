package main

import (
	"fmt"
	"os"

	"github.com/3096/furnace/commands"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: " + os.Args[0] + " <in wismt> <texture dir> <out wismt>")
		os.Exit(1)
	}
	commands.ReplaceTexturesInWismt(os.Args[1], os.Args[2], os.Args[3])
}
