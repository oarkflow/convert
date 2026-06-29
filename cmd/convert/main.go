package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/oarkflow/convert"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: convert <schema|validate|inspect> [flags]")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "validate":
		fs := flag.NewFlagSet("validate", flag.ExitOnError)
		file := fs.String("file", "", "JSON file to validate as generic payload")
		_ = fs.Parse(os.Args[2:])
		var data any
		b, err := os.ReadFile(*file)
		if err != nil {
			fatal(err)
		}
		if err := json.Unmarshal(b, &data); err != nil {
			fatal(err)
		}
		fmt.Println("valid JSON payload; use package APIs for typed DTO validation")
	case "schema", "inspect":
		fmt.Println("The CLI is installed. Typed schemas are generated in Go via convert.StableJSONSchema[T] and convert.Describe[T].")
		fmt.Println(convert.SafeString(map[string]any{"status": "ok"}))
	default:
		fmt.Fprintln(os.Stderr, "unknown command")
		os.Exit(2)
	}
}

func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
