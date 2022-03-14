package notify

import (
	"fmt"
	"sync"
	"time"
)

// Event is an event emitted by a watcher.
type Event interface {
	fmt.Stringer
	IsDir() bool
	Path() string
	WatcherEvent() string
}

// ------------------------
//   CreateEvent
// ------------------------

// CreateEvent represents the creation of a file or directory.
type CreateEvent struct {
	path  string
	isDir bool
}

func (ce CreateEvent) String() string {
	return ce.WatcherEvent()
}

// IsDir returns whether the event item is a directory.
func (ce CreateEvent) IsDir() bool {
	return ce.isDir
}

// Path returns the event item's path.
func (ce CreateEvent) Path() string {
	return ce.path
}

// WatcherEvent returns a string representation of the event.
func (ce CreateEvent) WatcherEvent() string {
	str := fmt.Sprintf("CREATE %v", ce.path)

	return str
}

// ------------------------
//   DeleteEvent
// ------------------------

// DeleteEvent represents the removal of a file or directory.
type DeleteEvent struct {
	path  string
	isDir bool
}

func (de DeleteEvent) String() string {
	return de.WatcherEvent()
}

// IsDir returns whether the event item is a directory.
func (de DeleteEvent) IsDir() bool {
	return de.isDir
}

// Path returns the event item's path.
func (de DeleteEvent) Path() string {
	return de.path
}

// WatcherEvent returns a string representation of the event.
func (de DeleteEvent) WatcherEvent() string {
	str := fmt.Sprintf("DELETE %v", de.path)

	return str
}

// ------------------------
//   ModifyEvent
// ------------------------

// ModifyEvent represents the modification of a file or directory.
type ModifyEvent struct {
	path string
}

func (me ModifyEvent) String() string {
	return me.WatcherEvent()
}

// IsDir returns whether the event item is a directory.
func (me ModifyEvent) IsDir() bool {
	return false
}

// Path returns the event item's path.
func (me ModifyEvent) Path() string {
	return me.path
}

// WatcherEvent returns a string representation of the event.
func (me ModifyEvent) WatcherEvent() string {
	str := fmt.Sprintf("MODIFY %v", me.path)

	return str
}

// ------------------------
//   RenameEvent
// ------------------------

// RenameEvent represents the moving of a file or directory.
// OldPath can be equal to "" if the old path is from an unwatched directory.
type RenameEvent struct {
	OldPath string
	path    string
	isDir   bool
}

func (re RenameEvent) String() string {
	return re.WatcherEvent()
}

// IsDir returns whether the event item is a directory.
func (re RenameEvent) IsDir() bool {
	return re.isDir
}

// Path returns the event item's path.
// Path can be equal to "" if the new path is from an unwatched directory.
func (re RenameEvent) Path() string {
	return re.path
}

// WatcherEvent returns a string representation of the event.
func (re RenameEvent) WatcherEvent() string {
	var str string

	switch {
	case re.OldPath != "" && re.path != "":
		str = fmt.Sprintf("RENAME %v to %v", re.OldPath, re.path)
	case re.OldPath != "":
		str = fmt.Sprintf("RENAME %v", re.OldPath)
	case re.path != "":
		str = fmt.Sprintf("RENAME to %v", re.path)
	}

	return str
}

// ------------------------
//   Move Event
// ------------------------

type mvEvent struct {
	oldParentWd int
	newParentWd int
	oldName     string
	newName     string
	isDir       bool
}

type mvEvents struct {
	mx     sync.Mutex
	mvFrom map[int]*mvFromEvent
	queue  chan *mvEvent
	done   chan struct{}
}

type mvFromEvent struct {
	cookie   int
	parentWd int
	name     string
	isDir    bool
	done     chan struct{}
}

type mvToEvent struct {
	cookie   int
	parentWd int
	name     string
}

//
func newMvEvents() *mvEvents {
	return &mvEvents{
		queue:  make(chan *mvEvent, 1),
		mvFrom: map[int]*mvFromEvent{},
		done:   make(chan struct{}),
	}
}

//
func (me *mvEvents) addMvFrom(cookie int, name string, parentWd int, isDir bool) {
	done := make(chan struct{})

	me.mx.Lock()
	me.mvFrom[cookie] = &mvFromEvent{
		cookie:   cookie,
		parentWd: parentWd,
		name:     name,
		isDir:    isDir,
		done:     done,
	}
	me.mx.Unlock()

	go func() {
		select {
		case <-done:
		case <-me.done:
		case <-time.After(time.Millisecond * 100):
			me.queue <- &mvEvent{
				oldParentWd: parentWd,
				oldName:     name,
				newParentWd: -1,
				isDir:       isDir,
			}
		}

		me.mx.Lock()
		delete(me.mvFrom, cookie)
		me.mx.Unlock()
	}()
}

//
func (me *mvEvents) addMvTo(cookie int, name string, parentWd int, isDir bool) {
	me.mx.Lock()
	mvFrom := me.mvFrom[cookie]
	me.mx.Unlock()

	if mvFrom != nil {
		me.mx.Lock()
		delete(me.mvFrom, cookie)
		me.mx.Unlock()

		close(mvFrom.done)

		me.queue <- &mvEvent{
			oldParentWd: mvFrom.parentWd,
			oldName:     mvFrom.name,
			newParentWd: parentWd,
			newName:     name,
			isDir:       isDir,
		}

		return
	}

	me.queue <- &mvEvent{
		oldParentWd: -1,
		newParentWd: parentWd,
		newName:     name,
	}
}

//
func (me *mvEvents) close() {
	me.mx.Lock()
	defer me.mx.Unlock()

	close(me.done)
}
