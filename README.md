# notify

## Tracking changes in the specified directory and its subdirectories

### NewDirNotify(dirPath string, ignoreRegExps []*regexp.Regexp) (*Notify, error)

dirPath - specifies the directory for changes to be tracked. All attached files and directories are added to the watchlist.

ignoreRegExps - defines an ignore list. They can be both files and directories. 


```go
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"syscall"

	"github.com/ds248a/notify"
)

// ignore list
var ignoreRegExps = []*regexp.Regexp{
	regexp.MustCompile("^vendor"),
}

func NewNotify() {
	
	// interrupt handling 
	deadlySignals := make(chan os.Signal, 1)
	signal.Notify(deadlySignals, os.Interrupt, syscall.SIGTERM)

	// filesystem change handler
	n, err := notify.NewDirNotify(".", ignoreRegExps)
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case <-deadlySignals:
			return
		case err := <-n.Errs():
			fmt.Printf("watcher: %+v\n", err)
			return
		case e := <-n.Events():
			fmt.Printf("event: %#v\n", e)
		}
	}
}

/*
event: notify.CreateEvent{path:"a", isDir:true}                    - new folder './a'
event: notify.CreateEvent{path:"a/b", isDir:true}                  - new folder './a/b'
event: notify.RenameEvent{OldPath:"a/b", path:"a/d", isDir:true}   - rename folder 'b' to 'd'
event: notify.DeleteEvent{path:"a/b", isDir:true}                  - delete folder './a/b'

event: notify.ModifyEvent{path:"a/a1.txt"}                                     - new or edit file
event: notify.RenameEvent{OldPath:"a/a1.txt", path:"a/a2.txt", isDir:false}    - rename file
event: notify.RenameEvent{OldPath:"a/a2.txt", path:"a/d/a2.txt", isDir:false}  - move file
event: notify.RenameEvent{OldPath:"a/d/a2.txt", path:"a/d.txt", isDir:false}   - rename && move file
event: notify.DeleteEvent{path:"a/d.txt", isDir:false}                         - delete file
*/
```
