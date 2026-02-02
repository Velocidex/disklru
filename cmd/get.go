package main

import (
	"context"
	"fmt"

	"github.com/Velocidex/disklru"
	"github.com/alecthomas/kingpin"
)

var (
	get_command          = app.Command("get", "Get string from cache.")
	get_command_filename = get_command.Arg(
		"filename", "Cache filename").Required().String()

	get_command_key = get_command.Arg("key", "The key to store").String()
)

func doGet() {
	opts := disklru.Options{
		Filename: *get_command_filename,
	}
	cache, err := disklru.NewDiskLRU(context.Background(), opts)
	kingpin.FatalIfError(err, "Creating cache")
	defer cache.Close()

	value, err := cache.Get(*get_command_key)
	kingpin.FatalIfError(err, "Getting cache")

	fmt.Printf("Value %v\n", value)
}

func init() {
	command_handlers = append(command_handlers, func(command string) bool {
		switch command {
		case get_command.FullCommand():
			doGet()
		default:
			return false
		}
		return true
	})
}
