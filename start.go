package main

import (
	"fmt"
	"path"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

func (n *Notify) startReading() {
	readingErr := make(chan error)
	readingRes := make(chan struct {
		inotifyE unix.InotifyEvent
		name     string
	})

	// reading from inotify instance's fd
	go func() {
		buff := [eventsBufferSize]byte{}

		for {
			select {
			case <-n.done:
				return
			default:
			}

			k, err := unix.Read(n.fd, buff[:])
			if err != nil {
				readingErr <- err
				return
			}

			previousNameLen := 0
			for i := 0; i < k; i += int(unix.SizeofInotifyEvent + previousNameLen) {
				select {
				case <-n.done:
					return
				default:
				}

				var name string

				inotifyE := (*unix.InotifyEvent)(unsafe.Pointer(&buff[i]))

				if inotifyE.Len > 0 {
					name = string(buff[i+unix.SizeofInotifyEvent : i+int(unix.SizeofInotifyEvent+inotifyE.Len)])
					// remove trailing null chars
					name = strings.TrimRight(name, "\x00")
				}

				readingRes <- struct {
					inotifyE unix.InotifyEvent
					name     string
				}{
					*inotifyE,
					name,
				}

				previousNameLen = int(inotifyE.Len)
			}
		}
	}()

	go func() {
		defer n.mvEvents.close()
		defer n.Close()

		for {
			select {
			case <-n.done:
				return
			case err := <-readingErr:
				n.errs <- fmt.Errorf("reading from inotify instance's fd: %v", err)

				return
			case res := <-readingRes:
				var e Event

				parentDir := n.tree.get(int(res.inotifyE.Wd))
				// this happens when an IN_IGNORED event about an already
				// removed directory is received.
				if parentDir == nil {
					continue
				}

				isDir := res.inotifyE.Mask&unix.IN_ISDIR == unix.IN_ISDIR

				fileOrDirPath := path.Join(n.tree.path(parentDir.wd), res.name)
				// if it matches, it means it should be ignored
				if n.matchPath(fileOrDirPath, isDir) {
					continue
				}

				switch {
				// this event is only handled if it is from the root,
				// since, if it is from any other directory, it means
				// that this directory's parent has already received
				// an IN_DELETE event and the directory's been already
				// removed from the inotify instance and the tree.
				case res.inotifyE.Mask&unix.IN_IGNORED == unix.IN_IGNORED && n.tree.get(int(res.inotifyE.Wd)) == n.tree.root:
					return
				case res.inotifyE.Mask&unix.IN_CREATE == unix.IN_CREATE:
					if isDir {
						_, match, err := n.addDir(res.name, parentDir.wd)
						if !match {
							if err != nil {
								n.errs <- err

								return
							}

							err = n.addDirsStartingAt(fileOrDirPath)
							if err != nil {
								n.errs <- err

								return
							}
						}
					}

					e = CreateEvent{
						path:  fileOrDirPath,
						isDir: isDir,
					}
				case res.inotifyE.Mask&unix.IN_DELETE == unix.IN_DELETE:
					if isDir {
						dir := n.tree.find(fileOrDirPath)
						// this should never happen
						if dir == nil {
							continue
						}

						// the directory isn't removed from the inotify instance
						// because it was removed automatically when it was removed
						n.tree.rm(dir.wd)
					}

					e = DeleteEvent{
						path:  fileOrDirPath,
						isDir: isDir,
					}
				case res.inotifyE.Mask&unix.IN_CLOSE_WRITE == unix.IN_CLOSE_WRITE:
					e = ModifyEvent{
						path: fileOrDirPath,
					}
				case res.inotifyE.Mask&unix.IN_MOVED_FROM == unix.IN_MOVED_FROM:
					n.mvEvents.addMvFrom(int(res.inotifyE.Cookie), res.name, int(res.inotifyE.Wd), isDir)
				case res.inotifyE.Mask&unix.IN_MOVED_TO == unix.IN_MOVED_TO:
					n.mvEvents.addMvTo(int(res.inotifyE.Cookie), res.name, int(res.inotifyE.Wd), isDir)
				}

				if e != nil {
					n.events <- e
				}
			case mvEvent := <-n.mvEvents.queue:
				var oldPath, newPath string

				hasMvFrom := mvEvent.oldName != ""
				hasMvTo := mvEvent.newName != ""

				switch {
				case hasMvFrom && hasMvTo:
					oldPath = path.Join(
						n.tree.path(mvEvent.oldParentWd),
						mvEvent.oldName,
					)
					newPath = path.Join(
						n.tree.path(mvEvent.newParentWd),
						mvEvent.newName,
					)

					if mvEvent.isDir {
						n.tree.mv(n.tree.find(oldPath).wd, mvEvent.newParentWd, mvEvent.newName)
					}
				case hasMvFrom:
					oldPath = path.Join(
						n.tree.path(mvEvent.oldParentWd),
						mvEvent.oldName,
					)

					if mvEvent.isDir {
						n.tree.rm(n.tree.find(oldPath).wd)
					}
				case hasMvTo:
					newPath = path.Join(
						n.tree.path(mvEvent.newParentWd),
						mvEvent.newName,
					)

					if mvEvent.isDir {
						_, match, err := n.addDir(mvEvent.newName, mvEvent.newParentWd)
						if !match {
							if err != nil {
								n.errs <- err

								return
							}

							err = n.addDirsStartingAt(newPath)
							if err != nil {
								n.errs <- err

								return
							}
						}
					}
				}

				n.events <- RenameEvent{
					isDir:   mvEvent.isDir,
					OldPath: oldPath,
					path:    newPath,
				}
			}
		}
	}()
}
