package cmd

import (
	"flag"
)

func init() {
	flag.StringVar(&SQLConn, "conn", "root:@tcp(127.0.0.1:3306)/test", "Connection string used by the SQLDriver to connect to a database instance.")
	flag.StringVar(&Tables, "tables", "", "List of table names separated by a comma.")
	flag.StringVar(&Fields, "fileds", "", "List of table Fields.")
}

var SQLConn string
var Tables string
var Fields string