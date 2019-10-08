package main

import (
	"flag"
	// "fmt"
	"stom/generate"

	// "log"
	// "os"
)

func main() {
	flag.Parse()
	// println(wd)

	// if len(flag.Args()) < 1 {
	// 	// cmd.Usage()
	// 	os.Exit(0)
	// }

	generate.GenerateCode();
}
