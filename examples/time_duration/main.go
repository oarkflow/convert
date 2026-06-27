package main

import (
	"fmt"
	"time"

	"example.com/goconvert/convert"
)

func main() {
	delay, _ := convert.To[time.Duration]("1h30m")
	seconds, _ := convert.ToDurationSeconds(2.5)
	createdAt, _ := convert.To[time.Time]("2026-06-27T12:30:00Z")
	fromUnix, _ := convert.ToTime(1_700_000_000)

	fmt.Println("delay:", delay)
	fmt.Println("seconds:", seconds)
	fmt.Println("created_at:", createdAt.UTC().Format(time.RFC3339))
	fmt.Println("from_unix:", fromUnix.UTC().Format(time.RFC3339))
}
