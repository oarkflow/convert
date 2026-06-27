package main

import (
	"database/sql"
	"fmt"

	convert "github.com/oarkflow/convert"
)

func main() {
	n, _ := convert.ScanTo[int64]([]byte("42"))
	ns := convert.ToSQLNullString("ok")
	ni := convert.ToSQLNullInt64("123")
	fmt.Println(n, ns.Valid, ns.String, ni.Valid, ni.Int64, sql.NullBool{Bool: true, Valid: true}.Bool)
}
