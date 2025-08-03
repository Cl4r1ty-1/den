package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/den/cmd/master"
	"github.com/den/cmd/slave"
)

func main() {
	var mode = flag.String("mode", "", "mode to run: master or slave")
	flag.Parse()

	switch *mode {
	case "master":
		if err := master.Run(); err != nil {
			log.Fatalf("master failed: %v", err)
		}
	case "slave":
		if err := slave.Run(); err != nil {
			log.Fatalf("slave failed: %v", err)
		}
	default:
		fmt.Println("den")
		fmt.Println("usage:")
		fmt.Println("  den -mode=master")
		fmt.Println("  den -mode=slave")
		os.Exit(1)
	}
}