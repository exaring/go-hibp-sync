package main

import (
	"fmt"

	hibpsync "github.com/exaring/go-hibp-sync"
)

func main() {
	if err := hibpsync.Sync(); err != nil {
		fmt.Printf("sync error: %q", err)
	}
}
