package main

import (
	"flag"
	// "fmt"
	"stom/generate"
	"stom/cmd"

	// "log"
	"os"
)

func main() {
	flag.Parse()
	// println(wd)

	if  cmd.ConnStr == "" {
		cmd.Usage()
		os.Exit(0)
	}
	generate.Generate();
}
