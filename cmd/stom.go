package cmd

import (
	"fmt"
)



var usageTemplate = `Stom is a Fast and Flexible tool for generate models form MYSQL.
Usage:
	stom -conn[string].	common format 'user:passwd@tcp(ipAddress:port)/databasename'
	e,g:	stom -conn 'root:123123@tcp(127.0.0.1:3306)/test'
`


func Usage() {
	fmt.Println(usageTemplate)
}

func Help(args []string) {
	
}

