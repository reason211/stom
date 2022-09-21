package generate

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	p "path"
	"stom/cmd"
	"stom/utils"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func Generate() error {
	var selectedTables map[string]bool
	tables := cmd.Tables
	if tables != "" {
		selectedTables = make(map[string]bool)
		for _, v := range strings.Split(tables, ",") {
			selectedTables[v] = true
		}
	}
	if err := genModel(cmd.ConnStr, selectedTables); err != nil {
		return err
	}
	return nil
}

func genModel(connStr string, selectedTableNames map[string]bool) error {
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		log.Fatalf("Can not connect to database using '%s': %s", connStr, err)
	}
	defer db.Close()
	var tablesNames []string
	if len(selectedTableNames) != 0 {
		for table := range selectedTableNames {
			tablesNames = append(tablesNames, table)
		}
	} else {
		tablesNames = GetTableNames(db)
	}
	tables := GetTableObjects(tablesNames, db)
	dbName := GetDatabaseName(connStr)
	writeModelFiles(tables, dbName)
	return nil
}

// GetDatabaseName returns database name
func GetDatabaseName(connStr string) string {
	return p.Base(connStr)
}

// GetTableNames returns a slice of table names in the current database
func GetTableNames(db *sql.DB) (tables []string) {
	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		log.Fatalf("Could query tables: %s", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			log.Fatalf("Could query tables: %s", err)
		}
		tables = append(tables, name)
	}
	return
}

// writeModelFiles generate files
func writeModelFiles(tables []*Table, dbname string) {
	mpath, _ := os.Getwd()
	mpath = p.Join(dbname)

	var err error
	if err = os.Mkdir(mpath, os.ModePerm); err != nil {
		log.Printf("%s", err)
	}

	w := io.Writer(os.Stdout)
	for _, tb := range tables {
		filename := strings.Replace(tb.Name, dbname+"_", "", 1)
		filename = utils.CamelCase(filename + "_model")
		fpath := p.Join(mpath, filename+".go")
		var f *os.File
		if utils.IsExist(fpath) {
			//revert wd
			f, err = os.OpenFile(fpath, os.O_RDWR|os.O_TRUNC, 0666)
			if err != nil {
				log.Printf("%s", err)
				continue
			}
		} else {
			//creat & wd
			f, err = os.OpenFile(fpath, os.O_CREATE|os.O_RDWR, 0666)
			if err != nil {
				log.Printf("%s", err)
				continue
			}
		}
		var template string
		template = ModelTPL
		if strings.Contains(tb.Name, dbname) {
			template = ModelPrefixTPL
			tb.Name = strings.Replace(tb.Name, dbname+"_", "", 1)
		}
		fileStr := strings.Replace(template, "{{modelStruct}}", tb.String(), 1)
		fileStr = strings.Replace(fileStr, "{{modelName}}", utils.CamelCase(tb.Name), -1)
		fileStr = strings.Replace(fileStr, "{{tableName}}", utils.CamelCase(tb.Name), -1)
		fileStr = strings.Replace(fileStr, "{{dbName}}", dbname+"_", -1)

		if _, err := f.WriteString(fileStr); err != nil {
			log.Fatalf("Could not write model file to '%s': %s", fpath, err)
		}
		defer f.Close()
		fmt.Fprintf(w, "\t%s%screate%s\t %s%s\n", "\x1b[32m", "\x1b[1m", "\x1b[21m", fpath, "\x1b[0m")
		utils.FormatSourceCode(fpath)
	}
}

// GetTableObjects process each table name
func GetTableObjects(tableNames []string, db *sql.DB) (tables []*Table) {
	blackList := make(map[string]bool)
	// process constraints information for each table, also gather blacklisted table names
	for _, tableName := range tableNames {
		// create a table struct
		tb := new(Table)
		tb.Name = tableName
		GetConstraints(db, tb, blackList)
		tables = append(tables, tb)
	}
	// process columns, ignoring blacklisted tables
	for _, tb := range tables {
		GetColumns(db, tb, blackList)
	}
	return
}

// GetConstraints gets primary key, unique key and foreign keys of a table from
func GetConstraints(db *sql.DB, table *Table, blackList map[string]bool) {
	rows, err := db.Query(
		`SELECT
			c.constraint_type, u.column_name, u.referenced_table_schema, u.referenced_table_name, referenced_column_name, u.ordinal_position
		FROM
			information_schema.table_constraints c
		INNER JOIN
			information_schema.key_column_usage u ON c.constraint_name = u.constraint_name
		WHERE
			c.table_schema = database() AND c.table_name = ? AND u.table_schema = database() AND u.table_name = ?`,
		table.Name, table.Name) //  u.position_in_unique_constraint,
	if err != nil {
		log.Fatal("Could not query INFORMATION_SCHEMA for PK/UK/FK information")
	}
	for rows.Next() {
		var constraintTypeBytes, columnNameBytes, refTableSchemaBytes, refTableNameBytes, refColumnNameBytes, refOrdinalPosBytes []byte
		if err := rows.Scan(&constraintTypeBytes, &columnNameBytes, &refTableSchemaBytes, &refTableNameBytes, &refColumnNameBytes, &refOrdinalPosBytes); err != nil {
			log.Fatal("Could not read INFORMATION_SCHEMA for PK/UK/FK information")
		}
		constraintType, columnName, refOrdinalPos :=
			string(constraintTypeBytes), string(columnNameBytes), string(refOrdinalPosBytes)
		if constraintType == "PRIMARY KEY" {
			if refOrdinalPos == "1" {
				table.Pk = columnName
			} else {
				table.Pk = ""
				// Add table to blacklist so that other struct will not reference it, because we are not
				// registering blacklisted tables
				blackList[table.Name] = true
			}
		}
	}
}

// GetColumns retrieves columns details from  information_schema and fill in the Column struct
func GetColumns(db *sql.DB, table *Table, blackList map[string]bool) {
	// retrieve columns
	colDefRows, err := db.Query(
		`SELECT
			column_name, data_type, column_type, is_nullable, column_default, extra, column_comment 
		FROM
			information_schema.columns
		WHERE
			table_schema = database() AND table_name = ?`,
		table.Name)
	if err != nil {
		log.Fatalf("Could not query the database: %s", err)
	}
	defer colDefRows.Close()

	for colDefRows.Next() {
		// datatype as bytes so that SQL <null> values can be retrieved
		var colNameBytes, dataTypeBytes, columnTypeBytes, isNullableBytes, columnDefaultBytes, extraBytes, columnCommentBytes []byte
		if err := colDefRows.Scan(&colNameBytes, &dataTypeBytes, &columnTypeBytes, &isNullableBytes, &columnDefaultBytes, &extraBytes, &columnCommentBytes); err != nil {
			log.Fatalf("Could not query INFORMATION_SCHEMA for column information error: %s", err)
		}
		colName, columnComment :=
			string(colNameBytes), string(columnCommentBytes)

		// create a column
		col := new(Column)
		col.Name = utils.CamelCase(colName)
		col.Type = "string"
		if strings.Contains(string(dataTypeBytes), "int") {
			col.Type = "int"
		}

		// Tag info
		tag := new(OrmTag)
		if table.Pk == colName || colName == "id" {
			col.Type = "string"
			tag.Pk = true
		}
		tag.Column = colName
		tag.Comment = columnComment
		col.Tag = tag

		table.Columns = append(table.Columns, col)
	}
}
