package generate

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	p "path"
	"stom/cmd"
	"strings"

	"stom/utils"

	"github.com/beego/bee/logger/colors"
	_ "github.com/go-sql-driver/mysql"
)

// Table represent a table in a database
type Table struct {
	Name          string
	Pk            string
	Uk            []string
	Fk            map[string]*ForeignKey
	Columns       []*Column
	ImportTimePkg bool
}

// Column reprsents a column for a table
type Column struct {
	Name string
	Type string
	Tag  *OrmTag
}

// ForeignKey represents a foreign key column for a table
type ForeignKey struct {
	Name      string
	RefSchema string
	RefTable  string
	RefColumn string
}

// OrmTag contains Beego ORM tag information for a column
type OrmTag struct {
	Auto        bool
	Pk          bool
	Null        bool
	Index       bool
	Unique      bool
	Column      string
	Size        string
	Decimals    string
	Digits      string
	AutoNow     bool
	AutoNowAdd  bool
	Type        string
	Default     string
	RelOne      bool
	ReverseOne  bool
	RelFk       bool
	ReverseMany bool
	RelM2M      bool
	Comment     string //column comment
}

// String returns the source code string for the Table struct
func (tb *Table) String() string {
	rv := fmt.Sprintf("type %s struct {\n", utils.CamelCase(tb.Name))
	for _, v := range tb.Columns {
		rv += v.String() + "\n"
	}
	rv += "}\n"
	return rv
}

// String returns the source code string of a field in Table struct
// It maps to a column in database table. e.g. Id int `orm:"column(id);auto"`
func (col *Column) String() string {
	return fmt.Sprintf("%s %s %s", col.Name, col.Type, col.Tag.String())
}

// String returns the ORM tag string for a column
func (tag *OrmTag) String() string {
	var ormOptions []string
	if tag.Column != "" {
		ormOptions = append(ormOptions, fmt.Sprintf("column(%s)", tag.Column))
	}
	if tag.Auto {
		ormOptions = append(ormOptions, "auto")
	}
	if tag.Size != "" {
		ormOptions = append(ormOptions, fmt.Sprintf("size(%s)", tag.Size))
	}
	if tag.Type != "" {
		ormOptions = append(ormOptions, fmt.Sprintf("type(%s)", tag.Type))
	}
	if tag.Null {
		ormOptions = append(ormOptions, "null")
	}
	if tag.AutoNow {
		ormOptions = append(ormOptions, "auto_now")
	}
	if tag.AutoNowAdd {
		ormOptions = append(ormOptions, "auto_now_add")
	}
	if tag.Decimals != "" {
		ormOptions = append(ormOptions, fmt.Sprintf("digits(%s);decimals(%s)", tag.Digits, tag.Decimals))
	}
	if tag.RelFk {
		ormOptions = append(ormOptions, "rel(fk)")
	}
	if tag.RelOne {
		ormOptions = append(ormOptions, "rel(one)")
	}
	if tag.ReverseOne {
		ormOptions = append(ormOptions, "reverse(one)")
	}
	if tag.ReverseMany {
		ormOptions = append(ormOptions, "reverse(many)")
	}
	if tag.RelM2M {
		ormOptions = append(ormOptions, "rel(m2m)")
	}
	if tag.Pk {
		ormOptions = append(ormOptions, "pk")
	}
	if tag.Unique {
		ormOptions = append(ormOptions, "unique")
	}
	if tag.Default != "" {
		ormOptions = append(ormOptions, fmt.Sprintf("default(%s)", tag.Default))
	}

	if len(ormOptions) == 0 {
		return ""
	}
	if tag.Comment != "" {
		return fmt.Sprintf("`orm:\"%s\" description:\"%s\"`", strings.Join(ormOptions, ";"), tag.Comment)
	}
	return fmt.Sprintf("`orm:\"%s\"`", strings.Join(ormOptions, ";"))
}

func GenerateCode() error {
	var selectedTables map[string]bool
	tables := cmd.Tables
	if tables != "" {
		selectedTables = make(map[string]bool)
		for _, v := range strings.Split(tables, ",") {
			selectedTables[v] = true
		}
	}
	if err := genModel(cmd.SQLConn, selectedTables); err != nil {
		return err
	}
	return nil
}

func genModel(connStr string, selectedTableNames map[string]bool) error {
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		log.Fatalf("Could not connect to database using '%s': %s", connStr, err)
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
	tables := getTableObjects(tablesNames, db)
	dbName := GetDatabaseName(connStr)
	writeModelFiles(tables, dbName)
	return nil
}

func GetDatabaseName(connStr string) string {
	return p.Base(connStr)
}

// GetTableNames returns a slice of table names in the current database
func GetTableNames(db *sql.DB) (tables []string) {
	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		log.Fatalf("Could not show tables: %s", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			log.Fatalf("Could not show tables: %s", err)
		}
		tables = append(tables, name)
	}
	return
}

// writeModelFiles generates model files
func writeModelFiles(tables []*Table, dbname string) {
	mpath, _ := os.Getwd()
	mpath = p.Join(dbname)

	var err error
	if err = os.Mkdir(mpath, os.ModePerm); err != nil {
		log.Printf("%s", err)
	}

	w := colors.NewColorWriter(os.Stdout)
	for _, tb := range tables {
		filename := strings.Replace(tb.Name, dbname+"_", "", 1)
		filename = utils.CamelCase(filename + "_model")
		fpath := p.Join(mpath, filename+".go")
		var f *os.File
		if utils.IsExist(fpath) {
			// log.Printf("'%s' already exists. Do you want to overwrite it? [Yes|No] ", fpath)
			// if utils.AskForConfirmation() {
			// 	f, err = os.OpenFile(fpath, os.O_RDWR|os.O_TRUNC, 0666)
			// 	if err != nil {
			// 		log.Printf("%s", err)
			// 		continue
			// 	}
			// } else {
			// 	log.Printf("Skipped create file '%s'", fpath)
			// 	continue
			// }
			f, err = os.OpenFile(fpath, os.O_RDWR|os.O_TRUNC, 0666)
			if err != nil {
				log.Printf("%s", err)
				continue
			}
		} else {
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
		utils.CloseFile(f)
		fmt.Fprintf(w, "\t%s%screate%s\t %s%s\n", "\x1b[32m", "\x1b[1m", "\x1b[21m", fpath, "\x1b[0m")
		utils.FormatSourceCode(fpath)
	}
}

// getTableObjects process each table name
func getTableObjects(tableNames []string, db *sql.DB) (tables []*Table) {
	// if a table has a composite pk or doesn't have pk, we can't use it yet
	// these tables will be put into blacklist so that other struct will not
	// reference it.
	blackList := make(map[string]bool)
	// process constraints information for each table, also gather blacklisted table names
	for _, tableName := range tableNames {
		// create a table struct
		tb := new(Table)
		tb.Name = tableName
		tb.Fk = make(map[string]*ForeignKey)
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
// information_schema and fill in the Table struct
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
		constraintType, columnName, refTableSchema, refTableName, refColumnName, refOrdinalPos :=
			string(constraintTypeBytes), string(columnNameBytes), string(refTableSchemaBytes),
			string(refTableNameBytes), string(refColumnNameBytes), string(refOrdinalPosBytes)
		if constraintType == "PRIMARY KEY" {
			if refOrdinalPos == "1" {
				table.Pk = columnName
			} else {
				table.Pk = ""
				// Add table to blacklist so that other struct will not reference it, because we are not
				// registering blacklisted tables
				blackList[table.Name] = true
			}
		} else if constraintType == "UNIQUE" {
			table.Uk = append(table.Uk, columnName)
		} else if constraintType == "FOREIGN KEY" {
			fk := new(ForeignKey)
			fk.Name = columnName
			fk.RefSchema = refTableSchema
			fk.RefTable = refTableName
			fk.RefColumn = refColumnName
			table.Fk[columnName] = fk
		}
	}
}

// GetColumns retrieves columns details from
// information_schema and fill in the Column struct
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

const (
	ModelTPL = `package models
import (
	"github.com/astaxie/beego/orm"
)

{{modelStruct}}

func init() {
	orm.RegisterModel(new({{tableName}}))
}

`

	ModelPrefixTPL = `package models

import (
	"github.com/astaxie/beego/orm"
)

{{modelStruct}}

func init() {
	orm.RegisterModelWithPrefix("{{dbName}}", new({{tableName}}))
}

`
)
