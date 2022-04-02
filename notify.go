package notify

import (
	"fmt"
	"sync"

	"io/ioutil"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/sys/unix"
)

// sys/unix
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

//                       (0x10 + 0xff + 1) * 64 = 17408
const eventsBufferSize = (unix.SizeofInotifyEvent + unix.NAME_MAX + 1) * 64
const inotifyMask = unix.IN_CREATE | unix.IN_DELETE | unix.IN_CLOSE_WRITE | unix.IN_MOVED_FROM | unix.IN_MOVED_TO

var alwaysIgnoreRegExps = []*regexp.Regexp{
	regexp.MustCompile("(?:^\\..*)|(?:/\\.)"),
}

// ------------------------
//   Notify
// ------------------------

type Notify struct {
	mx            sync.RWMutex
	fd            int
	closed        bool
	tree          *watchDirsTree
	ignoreRegExps []*regexp.Regexp
	done          chan struct{}
	events        chan Event
	errs          chan error
	mvEvents      *mvEvents
}

// NewDirNotify listens for changes in the specified directory.
// ignoreRegExps may contain a list of directories whose contents should be ignored.
// It can be either files or directories.
func NewDirNotify(dirPath string, ignoreRegExps []*regexp.Regexp) (*Notify, error) {
	if len(ignoreRegExps) == 0 {
		ignoreRegExps = alwaysIgnoreRegExps
	}

	fd, err := unix.InotifyInit1(0)
	if err != nil {
		return nil, fmt.Errorf("creating inotify instance: %v", err)
	}

	done := make(chan struct{})
	n := &Notify{
		fd:            fd,
		tree:          newWatchDirsTree(),
		done:          done,
		ignoreRegExps: ignoreRegExps,
	}

	rootWd, err := n.addToInotify(dirPath)
	if err != nil {
		return nil, err
	}
	n.tree.setRoot(dirPath, rootWd)

	err = n.addDirsStartingAt(dirPath)
	if err != nil {
		return nil, err
	}

	n.events = make(chan Event)
	n.errs = make(chan error)
	n.mvEvents = newMvEvents()

	n.run()

	return n, nil
}

// addToInotify adds the given path to the inotify instance and returns the added directory's wd.
// Note that it doesn't check whether the given path is match for any of w.ignoreRegExps.
func (n *Notify) addToInotify(path string) (int, error) {
	wd, err := unix.InotifyAddWatch(n.fd, path, inotifyMask)
	if err != nil {
		return -1, fmt.Errorf("adding directory to inotify instance: %v", err)
	}

	return wd, nil
}

// removeFromInotify removes the given path from the inotify instance.
func (n *Notify) removeFromInotify(wd int) error {
	wd, err := unix.InotifyRmWatch(n.fd, uint32(wd))
	if err != nil {
		return fmt.Errorf("removing directory from inotify instance: %v", err)
	}

	return nil
}

// addDir checks if a directory isn't a match for any of w.ignoreRegExps and, if it isn't,
// adds it to the tree and to the inotify instance and returns the added directory's wd.
func (n *Notify) addDir(name string, parentWd int) (wd int, match bool, err error) {
	dirPath := path.Join(n.tree.path(parentWd), name)

	if n.matchPath(dirPath, true) {
		return -1, true, nil
	}

	wd, err = n.addToInotify(dirPath)
	if err != nil {
		return -1, false, err
	}

	n.tree.add(wd, name, parentWd)

	return wd, false, nil
}

// addDirsStartingAt adds every directory descendant of rootPath recursively to the tree and to the inotify instance.
// This functions assumes that there's a node in the tree whose path is equal to cleanPath(rootPath).
func (n *Notify) addDirsStartingAt(rootPath string) error {
	entries, err := ioutil.ReadDir(rootPath)
	if err != nil {
		return fmt.Errorf("reading %v dir: %v", rootPath, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			_, match, err := n.addDir(entry.Name(), n.tree.find(cleanPath(rootPath)).wd)
			if match {
				continue
			}
			if err != nil {
				return err
			}

			dirPath := path.Join(rootPath, entry.Name())
			err = n.addDirsStartingAt(dirPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// matchPath returns whether the given path matchs any of w.ignoreRegExps.
func (n *Notify) matchPath(path string, isDir bool) bool {
	if isDir {
		path += "/"
	}

	for _, rx := range n.ignoreRegExps {
		if match := rx.MatchString(path); match {
			return true
		}
	}

	return false
}

// Events returns the events channel.
func (n *Notify) Events() chan Event {
	return n.events
}

// Errs returns the errors channel.
func (n *Notify) Errs() chan error {
	return n.errs
}

// Wait blocks until the watcher is closed.
func (n *Notify) Wait() {
	<-n.done
}

// Close closes the watcher.
// If the watcher is already closed, it's a no-op.
func (n *Notify) Close() error {
	if n.closed {
		return nil
	}

	n.closed = true
	err := unix.Close(n.fd)
	close(n.done)
	if err != nil {
		return fmt.Errorf("closing fd: %v", err)
	}

	return nil
}

// ------------------------
//   Dirs Tree
// ------------------------

// watchDir represents a directory being watched.
// If it's the root, parent=nil.
type watchDir struct {
	mx       sync.RWMutex
	wd       int
	name     string
	parent   *watchDir
	children map[string]*watchDir
}

type watchDirsTree struct {
	mx    sync.RWMutex
	root  *watchDir
	items map[int]*watchDir
	cache *watchDirsTreeCache
}

//
func newWatchDirsTree() *watchDirsTree {
	return &watchDirsTree{
		items: map[int]*watchDir{},
		cache: newWatchDirsTreeCache(),
	}
}

//
func (wd *watchDir) Name() string {
	wd.mx.RLock()
	defer wd.mx.RUnlock()

	return wd.name
}

//
func (wd *watchDir) setName(name string) {
	wd.mx.Lock()
	defer wd.mx.Unlock()

	wd.name = name
}

//
func (wd *watchDir) getChild(name string) *watchDir {
	wd.mx.RLock()
	defer wd.mx.RUnlock()

	return wd.children[name]
}

//
func (wd *watchDir) setChild(name string, dir *watchDir) {
	wd.mx.Lock()
	defer wd.mx.Unlock()

	wd.children[name] = dir
}

//
func (wd *watchDir) rmChild(name string) {
	wd.mx.Lock()
	defer wd.mx.Unlock()

	delete(wd.children, name)
}

//
func (wd *watchDir) setParent(d *watchDir) {
	wd.mx.Lock()
	defer wd.mx.Unlock()

	wd.parent = d
}

//
func (wdt *watchDirsTree) getRoot() *watchDir {
	wdt.mx.RLock()
	defer wdt.mx.RUnlock()

	return wdt.root
}

//
func (wdt *watchDirsTree) setRoot(path string, wd int) {
	if wdt.getRoot() != nil {
		panic("there's already a root")
	}

	d := &watchDir{
		wd:       wd,
		name:     cleanPath(path),
		children: map[string]*watchDir{},
	}

	wdt.mx.Lock()
	defer wdt.mx.Unlock()

	wdt.root = d
	wdt.items[d.wd] = d
}

//
func (wdt *watchDirsTree) add(wd int, name string, parentWd int) {
	parent := wdt.get(parentWd)
	if parent == nil {
		panic("parent not found")
	}

	d := &watchDir{
		wd:       wd,
		name:     name,
		parent:   parent,
		children: map[string]*watchDir{},
	}

	// d.parent.children[d.name] = d
	// wdt.items[d.wd] = d
	d.parent.setChild(name, d)
	wdt.set(d.wd, d)
}

//
func (wdt *watchDirsTree) get(wd int) *watchDir {
	wdt.mx.RLock()
	defer wdt.mx.RUnlock()

	return wdt.items[wd]
}

//
func (wdt *watchDirsTree) set(wd int, d *watchDir) {
	wdt.mx.Lock()
	defer wdt.mx.Unlock()

	wdt.items[wd] = d
}

//
func (wdt *watchDirsTree) rm(wd int) {
	item := wdt.get(wd)
	dirName := item.Name()

	if item == nil {
		return
	}

	if item.parent == nil {
		panic("cannot remove root")
	}

	item.parent.rmChild(dirName)

	for _, child := range item.children {
		wdt.rm(child.wd)
	}

	wdt.invalidate(wd)

	wdt.mx.Lock()
	delete(wdt.items, item.wd)
	wdt.mx.Unlock()
}

// if newParentWd < 0, the dir's parent isn't updated.
// if name == "", the dir's name isn't updated.
func (wdt *watchDirsTree) mv(wd, newParentWd int, name string) {
	item := wdt.get(wd)
	dirName := item.Name()

	if item == nil {
		panic("item not found")
	}

	if item.parent == nil {
		panic("cannot move root")
	}

	if newParentWd == -1 {
		newParentWd = item.parent.wd
	}

	newParent := wdt.get(newParentWd)
	if newParent == nil {
		panic("newParent not found")
	}

	if name != "" && name != dirName {
		// delete(item.parent.children, item.name)
		// item.name = name
		// item.parent.children[name] = item
		item.parent.rmChild(dirName)
		item.setName(name)
		item.parent.setChild(name, item)
	}

	if newParentWd != item.parent.wd {
		// delete(item.parent.children, item.name)
		// newParent.children[dirName] = item
		// item.parent = newParent
		item.parent.rmChild(dirName)
		newParent.setChild(dirName, item)
		item.setParent(newParent)
	}

	wdt.invalidate(wd)
}

//
func (wdt *watchDirsTree) path(wd int) string {
	if _, ok := wdt.cache.path(wd); !ok {
		item := wdt.get(wd)
		if item == nil {
			panic("item not found while generating path")
		}

		// if this is true, it's the root
		if item.parent == nil {
			return item.Name()
		}

		wdt.cache.add(wd, path.Join(wdt.path(item.parent.wd), item.Name()))
	}

	path, _ := wdt.cache.path(wd)
	return path
}

//
func (wdt *watchDirsTree) has(wd int) bool {
	wdt.mx.RLock()
	defer wdt.mx.RUnlock()

	_, ok := wdt.items[wd]
	return ok
}

//
func (wdt *watchDirsTree) find(path string) *watchDir {
	if wdt.root.Name() == path {
		return wdt.getRoot()
	}

	if path == "" {
		return nil
	}

	wd, ok := wdt.cache.wd(path)
	if !ok {
		pathWithoutRoot := strings.TrimPrefix(path, wdt.root.Name()+"/")
		pathSegments := strings.Split(pathWithoutRoot, string(filepath.Separator))

		parent := wdt.getRoot()

		for _, pathSegment := range pathSegments {
			d := parent.getChild(pathSegment)
			if d == nil {
				return nil
			}

			parent = d
		}

		return parent
	}

	return wdt.get(wd)
}

//
func (wdt *watchDirsTree) invalidate(wd int) {
	item := wdt.get(wd)
	if item == nil {
		panic("item not found")
	}

	for _, child := range item.children {
		wdt.invalidate(child.wd)
	}

	wdt.cache.rmByWd(wd)
}

// cleanPath cleans the path p.
// It has the same behaviour as path.Clean(), except when p == ".",
// which results in an empty string.
func cleanPath(p string) string {
	if p == "." {
		return ""
	}

	return path.Clean(p)
}

// ------------------------
//   Dirs Tree Cache
// ------------------------

type watchDirsTreeCache struct {
	mx       sync.RWMutex
	pathByWd map[int]string
	wdByPath map[string]int
}

//
func newWatchDirsTreeCache() *watchDirsTreeCache {
	return &watchDirsTreeCache{
		pathByWd: map[int]string{},
		wdByPath: map[string]int{},
	}
}

//
func (wdtc *watchDirsTreeCache) add(wd int, path string) {
	wdtc.mx.Lock()
	defer wdtc.mx.Unlock()

	wdtc.pathByWd[wd] = path
	wdtc.wdByPath[path] = wd
}

//
func (wdtc *watchDirsTreeCache) path(wd int) (string, bool) {
	wdtc.mx.RLock()
	defer wdtc.mx.RUnlock()

	path, ok := wdtc.pathByWd[wd]
	return path, ok
}

//
func (wdtc *watchDirsTreeCache) rmByPath(path string) {
	wd, ok := wdtc.wd(path)
	if !ok {
		return
	}

	wdtc.mx.Lock()
	defer wdtc.mx.Unlock()

	delete(wdtc.pathByWd, wd)
	delete(wdtc.wdByPath, path)
}

//
func (wdtc *watchDirsTreeCache) wd(path string) (int, bool) {
	wdtc.mx.RLock()
	defer wdtc.mx.RUnlock()

	wd, ok := wdtc.wdByPath[path]
	return wd, ok
}

//
func (wdtc *watchDirsTreeCache) rmByWd(wd int) {
	path, ok := wdtc.path(wd)
	if !ok {
		return
	}

	wdtc.mx.Lock()
	defer wdtc.mx.Unlock()

	delete(wdtc.pathByWd, wd)
	delete(wdtc.wdByPath, path)
}
