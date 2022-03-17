# notify

## Отслеживание изменений в указанной директории и её подкаталогах

### NewDirNotify(dirPath string, ignoreRegExps []*regexp.Regexp) (*Notify, error)

dirPath - указывает директирию изменения в которой следует отслеживать.
Все вложенные файлы и каталоги добавляются в список отслеживания.

ignoreRegExps - позволяет для переданной директории задать исключения для формирования списка отслеживаний.

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

// Cписок игнорирования.
var ignoreRegExps = []*regexp.Regexp{
	regexp.MustCompile("^vendor"),
}

func NewNotify() {
	// обработка прерываний
	deadlySignals := make(chan os.Signal, 1)
	signal.Notify(deadlySignals, os.Interrupt, syscall.SIGTERM)

	// обработчик изменений файловой системы
	n, err := notify.NewDirNotify(".", ignoreRegExps)
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case <-deadlySignals:
			return
		case err := <-n.Errs():
			fmt.Printf("watcher: %+v \n", err)
			return
		case e := <-n.Events():
			fmt.Printf("event: %#v \n", e)
		}
	}
}

/*
regexp.MustCompile("^vendor") - // приводит к игнорированию действий в каталоге './vendor'

event: notify.CreateEvent{path:"a", isDir:true}                    - new folder './a'
event: notify.CreateEvent{path:"a/b", isDir:true}                  - new folder './a/b'
event: notify.RenameEvent{OldPath:"a/b", path:"a/d", isDir:true}   - rename folder 'b' to 'd'
event: notify.DeleteEvent{path:"a/c", isDir:true}                  - delete folder './a/c'

event: notify.ModifyEvent{path:"a/a1.txt"}                                     - new or edit file
event: notify.RenameEvent{OldPath:"a/a1.txt", path:"a/a2.txt", isDir:false}    - rename file
event: notify.RenameEvent{OldPath:"a/a2.txt", path:"a/d/a2.txt", isDir:false}  - move file
event: notify.RenameEvent{OldPath:"a/d/a2.txt", path:"a/d.txt", isDir:false}   - rename && move file
event: notify.DeleteEvent{path:"a/d.txt", isDir:false}                         - delete file
*/
```
