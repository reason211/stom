// Package cmd ...
package cmd

import (
	"fmt"
)



var usageTemplate = "Stom is a Fast and Flexible tool for generate models form MYSQL."


func Usage() {
	fmt.Println(usageTemplate)
}

func Help(args []string) {
	// if len(args) == 0 {
	// 	Usage()
	// }
	// if len(args) != 1 {
	// 	utils.PrintErrorAndExit("Too many arguments", ErrorTemplate)
	// }

	// arg := args[0]

	// for _, cmd := range commands.AvailableCommands {
	// 	if cmd.Name() == arg {
	// 		utils.Tmpl(helpTemplate, cmd)
	// 		return
	// 	}
	// }
	// utils.PrintErrorAndExit("Unknown help topic", ErrorTemplate)
}

