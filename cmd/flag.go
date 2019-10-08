package cmd

import (
	"flag"
)

func init() {
	flag.StringVar(&ConnStr, "conn", "", "Connection string used to connect to a database.")
	flag.StringVar(&Tables, "tables", "", "Table names separated by a comma.")
	flag.StringVar(&Fields, "fileds", "", "Table Fields.")
}

var ConnStr string
var Tables string
var Fields string