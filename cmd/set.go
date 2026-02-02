package main

import (
	"context"

	"github.com/Velocidex/disklru"
	"github.com/alecthomas/kingpin"
)

var (
	set_command          = app.Command("set", "Set string into cache.")
	set_command_filename = set_command.Arg(
		"filename", "Cache filename").Required().String()

	set_command_key = set_command.Arg("key", "The key to store").String()

	set_command_value = set_command.Arg("value", "The value to store").String()
)

func doStore() {
	opts := disklru.Options{
		Filename: *set_command_filename,
	}

	cache, err := disklru.NewDiskLRU(context.Background(), opts)
	kingpin.FatalIfError(err, "Creating cache")
	defer cache.Close()

	err = cache.Set(*set_command_key, *set_command_value)
	kingpin.FatalIfError(err, "Setting cache")
}

func init() {
	command_handlers = append(command_handlers, func(command string) bool {
		switch command {
		case set_command.FullCommand():
			doStore()
		default:
			return false
		}
		return true
	})
}
