package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

const (
	IN_ACCESS = 0x1
	IN_MODIFY = 0x2

	IN_OPEN          = 0x20
	IN_CLOSE         = 0x18
	IN_CLOSE_NOWRITE = 0x10
	IN_CLOSE_WRITE   = 0x8
	IN_CREATE        = 0x100
	IN_DELETE        = 0x200
	IN_DELETE_SELF   = 0x400
	IN_MOVE          = 0xc0
	IN_MOVED_FROM    = 0x40
	IN_MOVED_TO      = 0x80
	IN_MOVE_SELF     = 0x800

	IN_ISDIR = 0x40000000
)

func main() {
	// обработка прерываний
	deadlySignals := make(chan os.Signal, 1)
	signal.Notify(deadlySignals, os.Interrupt, syscall.SIGTERM)

	n, err := NewDirNotify(".", alwaysIgnoreRegExps) //c.IgnoreRegExps)
	if err != nil {
		// logs.Err.Printf("watcher: %v\n", err)
		fmt.Printf("watcher: %v\n", err)
		return
	}

	for {
		// ctx, cancel := context.WithCancel(context.Background())
		_, cancel := context.WithCancel(context.Background())

		select {
		case <-deadlySignals:
			return
		case err := <-n.Errs():
			fmt.Printf("watcher: %+v \n", err)
			cancel()
			return
		case e := <-n.Events():
			fmt.Printf("event: %#v \n", e)
			cancel()
		}
	}
}

/*
event: main.CreateEvent{path:"test.txt", isDir:false}              <- new file
event: main.ModifyEvent{path:"test.txt"}                           <- edit file
event: main.RenameEvent{OldPath:"test.txt", path:"", isDir:false}  <- delete file

event: main.CreateEvent{path:"folder", isDir:true}                            <- new folder
event: main.RenameEvent{OldPath:"folder", path:"folder2", isDir:true}         <- rename folder
event: main.RenameEvent{OldPath:"cmd.go", path:"folder/cmd.go", isDir:false}  <- move file
event: main.DeleteEvent{path:"folder/cmd.go", isDir:false}                    <- file delete
event: main.DeleteEvent{path:"folder", isDir:true}                            <- folder delete
*/
