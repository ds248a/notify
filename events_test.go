package notify

import (
	// "fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"testing"
	"time"
)

// ------------------------
//   Init
// ------------------------

var eventTimeout = time.Millisecond * 150

// Creating a directory.
func mkDirAll(t *testing.T) {
	err := os.MkdirAll("a/b/c/d/e", os.ModeDir|os.ModePerm)
	if err != nil {
		t.Fatalf("unexpected error creating %v: %v", "a/b/c/d/e", err)
	}

	err = os.MkdirAll("f/g/h/i/j", os.ModeDir|os.ModePerm)
	if err != nil {
		t.Fatalf("unexpected error creating %v: %v", "f/g/h/i/j", err)
	}
}

// Removing a directory.
func rmDirAll(t *testing.T) {
	if err := os.RemoveAll("a"); err != nil {
		t.Fatalf("removing dir error: %v", err)
	}

	if err := os.RemoveAll("f"); err != nil {
		t.Fatalf("removing dir error: %v", err)
	}

	<-time.After(100 * time.Millisecond)
}

// Creates the named file.
func createFile(t *testing.T, filePath string) {
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("error file creating %v: %v", filePath, err)
	}

	err = file.Close()
	if err != nil {
		t.Fatalf("error file close: %v", err)
	}
}

//
func mkDir(t *testing.T, dirPath string) {
	if err := os.Mkdir(dirPath, os.ModeDir|os.ModePerm); err != nil {
		t.Fatalf("unexpected error creating %v: %v", dirPath, err)
	}
}

// ------------------------
//   File Test
// ------------------------

//
func TestWatcher_createEvent(t *testing.T) {
	t.Run("create_file", func(t *testing.T) {
		mkDirAll(t)
		defer rmDirAll(t)

		workingDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("unexpect err: %v", err)
		}

		w, err := NewDirNotify(workingDir, []*regexp.Regexp{})
		expectedErr := error(nil)
		if err != expectedErr {
			t.Fatalf("got %v, want %v", err, expectedErr)
		}
		defer w.Close()

		// new file
		filePath := path.Join("a/b/c/d/e", "a.txt")
		createFile(t, filePath)

		expectedEvent := CreateEvent{
			isDir: false,
			path:  path.Join(workingDir, filePath),
		}

		select {
		case e := <-w.Events():
			if e != expectedEvent {
				t.Fatalf("got %v, want %v", e, expectedEvent)
			}
		case err := <-w.Errs():
			t.Fatalf("unexpected err: %v", err)
		case <-w.done:
			t.Fatal("channel closed")
		case <-time.After(eventTimeout):
			t.Fatal("timeout reached waiting for event")
		}
	})

	//
	t.Run("create_directory", func(t *testing.T) {
		mkDirAll(t)
		defer rmDirAll(t)

		w, err := NewDirNotify(".", []*regexp.Regexp{})
		expectedErr := error(nil)
		if err != expectedErr {
			t.Fatalf("got %v, want %v", err, expectedErr)
		}
		defer w.Close()

		// new dir
		dirPath := path.Join("a/b/c/d/e", "z")
		mkDir(t, dirPath)

		expectedEvent := CreateEvent{
			isDir: true,
			path:  dirPath,
		}

		select {
		case e := <-w.Events():
			if e != expectedEvent {
				t.Fatalf("got %v, want %v", e, expectedEvent)
			}
		case err := <-w.Errs():
			t.Fatalf("unexpected err: %v", err)
		case <-w.done:
			t.Fatal("channel closed")
		case <-time.After(eventTimeout):
			t.Fatal("timeout reached waiting for event")
		}

		// new file
		filePath := path.Join(dirPath, "a.txt")
		createFile(t, filePath)

		expectedEvent = CreateEvent{
			isDir: false,
			path:  filePath,
		}

		select {
		case e := <-w.Events():
			if e != expectedEvent {
				t.Fatalf("got %v, want %v", e, expectedEvent)
			}
		case err := <-w.Errs():
			t.Fatalf("unexpected err: %v", err)
		case <-w.done:
			t.Fatal("channel closed")
		case <-time.After(eventTimeout):
			t.Fatal("timeout reached waiting for event")
		}
	})

	//
	t.Run("create_file_regexp", func(t *testing.T) {
		mkDirAll(t)
		defer rmDirAll(t)

		w, err := NewDirNotify(".", []*regexp.Regexp{regexp.MustCompile("^a/b/c/d/e/a.txt$")})
		expectedErr := error(nil)
		if err != expectedErr {
			t.Fatalf("got %v, want %v", err, expectedErr)
		}
		defer w.Close()

		// new file
		filePath := path.Join("a/b/c/d/e", "a.txt")
		createFile(t, filePath)

		select {
		case e := <-w.Events():
			t.Fatalf("unexpected event %v", e)
		case err := <-w.Errs():
			t.Fatalf("unexpected err: %v", err)
		case <-w.done:
			t.Fatal("channel closed")
		case <-time.After(eventTimeout):
		}
	})

	//
	t.Run("create_directory_regexp", func(t *testing.T) {
		mkDirAll(t)
		defer rmDirAll(t)

		w, err := NewDirNotify(".", []*regexp.Regexp{regexp.MustCompile("^a/b/c/d/e*")})
		expectedErr := error(nil)
		if err != expectedErr {
			t.Fatalf("got %v, want %v", err, expectedErr)
		}
		defer w.Close()

		// new dir
		dirPath := path.Join("a/b/c/d/e", "z")
		mkDir(t, dirPath)

		select {
		case e := <-w.Events():
			t.Fatalf("unexpected event %v", e)
		case err := <-w.Errs():
			t.Fatalf("unexpected err: %v", err)
		case <-w.done:
			t.Fatal("channel closed")
		case <-time.After(eventTimeout):
		}

		// new file
		filePath := path.Join(dirPath, "a.txt")
		createFile(t, filePath)

		select {
		case e := <-w.Events():
			t.Fatalf("unexpected event %v", e)
		case err := <-w.Errs():
			t.Fatalf("unexpected err: %v", err)
		case <-w.done:
			t.Fatal("channel closed")
		case <-time.After(eventTimeout):
		}
	})
}

//
func TestWatcher_deleteEvent(t *testing.T) {
	//
	t.Run("delete_file", func(t *testing.T) {
		mkDirAll(t)
		defer rmDirAll(t)

		// new file
		filePath := path.Join("a/b/c/d/e", "a.txt")
		createFile(t, filePath)

		w, err := NewDirNotify(".", []*regexp.Regexp{})
		expectedErr := error(nil)
		if err != expectedErr {
			t.Fatalf("got %v, want %v", err, expectedErr)
		}
		defer w.Close()

		err = os.Remove(filePath)
		if err != nil {
			t.Fatalf("error while removing %v: %v", filePath, err)
		}

		expectedEvent := DeleteEvent{
			isDir: false,
			path:  filePath,
		}

		select {
		case e := <-w.Events():
			if e != expectedEvent {
				t.Fatalf("got %v, want %v", e, expectedEvent)
			}
		case err := <-w.Errs():
			t.Fatalf("unexpected err: %v", err)
		case <-w.done:
			t.Fatal("channel closed")
		case <-time.After(eventTimeout):
			t.Fatal("timeout reached waiting for event")
		}
	})

	//
	t.Run("delete_directory", func(t *testing.T) {
		mkDirAll(t)
		defer rmDirAll(t)

		// new dir
		dirPath := path.Join("a/b/c/d/e", "z")
		mkDir(t, dirPath)

		workingDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("unexpect err: %v", err)
		}

		w, err := NewDirNotify(workingDir, []*regexp.Regexp{})
		expectedErr := error(nil)
		if err != expectedErr {
			t.Fatalf("got %v, want %v", err, expectedErr)
		}
		defer w.Close()

		err = os.RemoveAll(dirPath)
		if err != nil {
			t.Fatalf("error while removing %v: %v", dirPath, err)
		}

		expectedEvent := DeleteEvent{
			isDir: true,
			path:  path.Join(workingDir, dirPath),
		}

		select {
		case e := <-w.Events():
			if e != expectedEvent {
				t.Fatalf("got %v, want %v", e, expectedEvent)
			}
		case err := <-w.Errs():
			t.Fatalf("unexpected err: %v", err)
		case <-w.done:
			t.Fatal("channel closed")
		case <-time.After(eventTimeout):
			t.Fatal("timeout reached waiting for event")
		}
	})

	//
	t.Run("delete_file_regexp", func(t *testing.T) {
		mkDirAll(t)
		defer rmDirAll(t)

		// new file
		filePath := path.Join("f/g/h/i/j", "foo")
		createFile(t, filePath)

		w, err := NewDirNotify(".", []*regexp.Regexp{regexp.MustCompile("^" + filePath + "$")})
		expectedErr := error(nil)
		if err != expectedErr {
			t.Fatalf("got %v, want %v", err, expectedErr)
		}
		defer w.Close()

		err = os.Remove(filePath)
		if err != nil {
			t.Fatalf("error while removing %v: %v", filePath, err)
		}

		select {
		case e := <-w.Events():
			t.Fatalf("unexpected event %v", e)
		case err := <-w.Errs():
			t.Fatalf("unexpected err: %v", err)
		case <-w.done:
			t.Fatal("channel closed")
		case <-time.After(eventTimeout):
		}
	})

	//
	t.Run("delete_directory_regexp", func(t *testing.T) {
		mkDirAll(t)
		defer rmDirAll(t)

		// new dir
		dirPath := path.Join("f/g/h/i/j", "m")
		mkDir(t, dirPath)

		w, err := NewDirNotify(".", []*regexp.Regexp{regexp.MustCompile("^" + dirPath)})
		expectedErr := error(nil)
		if err != expectedErr {
			t.Fatalf("got %v, want %v", err, expectedErr)
		}
		defer w.Close()

		err = os.RemoveAll(dirPath)
		if err != nil {
			t.Fatalf("error while removing %v: %v", dirPath, err)
		}

		select {
		case e := <-w.Events():
			t.Fatalf("unexpected event %v", e)
		case err := <-w.Errs():
			t.Fatalf("unexpected err: %v", err)
		case <-w.done:
			t.Fatal("channel closed")
		case <-time.After(eventTimeout):
		}
	})
}

//
func TestWatcher_modifyEvent(t *testing.T) {
	t.Run("modify_file", func(t *testing.T) {
		mkDirAll(t)
		defer rmDirAll(t)

		// new file
		filePath := path.Join("a/b/c/d/e", "a.txt")
		createFile(t, filePath)

		workingDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("unexpect err: %v", err)
		}

		w, err := NewDirNotify(workingDir, []*regexp.Regexp{})
		expectedErr := error(nil)
		if err != expectedErr {
			t.Fatalf("got %v, want %v", err, expectedErr)
		}
		defer w.Close()

		// modify file
		err = ioutil.WriteFile(filePath, []byte("foo"), os.ModePerm)
		if err != nil {
			t.Fatalf("unexpected error writing to %v: %v", filePath, err)
		}

		expectedEvent := ModifyEvent{
			path: path.Join(workingDir, filePath),
		}

		select {
		case e := <-w.Events():
			if e != expectedEvent {
				t.Fatalf("got %v, want %v", e, expectedEvent)
			}
		case err := <-w.Errs():
			t.Fatalf("unexpected err: %v", err)
		case <-w.done:
			t.Fatal("channel closed")
		case <-time.After(eventTimeout):
			t.Fatal("timeout reached waiting for event")
		}
	})

	//
	t.Run("modify_file_regexp", func(t *testing.T) {
		mkDirAll(t)
		defer rmDirAll(t)

		// new file
		filePath := path.Join("a/b/c/d/e", "a.txt")
		createFile(t, filePath)

		w, err := NewDirNotify(".", []*regexp.Regexp{regexp.MustCompile("^" + filePath + "$")})
		expectedErr := error(nil)
		if err != expectedErr {
			t.Fatalf("got %v, want %v", err, expectedErr)
		}
		defer w.Close()

		// modify file
		err = ioutil.WriteFile(filePath, []byte("foo"), os.ModePerm)
		if err != nil {
			t.Fatalf("unexpected error writing to %v: %v", filePath, err)
		}

		select {
		case e := <-w.Events():
			t.Fatalf("unexpected event %v", e)
		case err := <-w.Errs():
			t.Fatalf("unexpected err: %v", err)
		case <-w.done:
			t.Fatal("channel closed")
		case <-time.After(eventTimeout):
		}
	})

	//
	t.Run("modify_file_regexp_2", func(t *testing.T) {
		defer rmDirAll(t)

		filePath := "foobar"
		createFile(t, filePath)
		defer os.Remove("foobar")

		err := os.MkdirAll("a/foobar", os.ModeDir|os.ModePerm)
		if err != nil {
			t.Fatalf("unexpected error creating %v: %v", "a/foobar", err)
		}
		defer os.RemoveAll("a")

		file2Path := path.Join("a", "foobar", "b.txt")
		createFile(t, file2Path)

		w, err := NewDirNotify(".", []*regexp.Regexp{regexp.MustCompile("foobar$")})
		expectedErr := error(nil)
		if err != expectedErr {
			t.Fatalf("got %v, want %v", err, expectedErr)
		}
		defer w.Close()

		// modify 'foobar' file
		err = ioutil.WriteFile(filePath, []byte("foo"), os.ModePerm)
		if err != nil {
			t.Fatalf("unexpected error writing to %v: %v", filePath, err)
		}

		// modify 'b.txt' file
		err = ioutil.WriteFile(file2Path, []byte("foo"), os.ModePerm)
		if err != nil {
			t.Fatalf("unexpected error writing to %v: %v", file2Path, err)
		}

		expectedEvent := ModifyEvent{
			path: file2Path,
		}

		select {
		case e := <-w.Events():
			if e != expectedEvent {
				t.Fatalf("got %v, want %v", e, expectedEvent)
			}
		case err := <-w.Errs():
			t.Fatalf("unexpected err: %v", err)
		case <-w.done:
			t.Fatal("channel closed")
		case <-time.After(eventTimeout):
			t.Fatal("timeout reached waiting for event")
		}
	})
}

//
func TestWatcher_renameEvent(t *testing.T) {
	//
	t.Run("rename file from a watched directory to a watched directory", func(t *testing.T) {
		mkDirAll(t)
		defer rmDirAll(t)

		oldFilePath := path.Join("a/b/c/d/e", "a.txt")
		newFilePath := path.Join("f/g/h/i/j", "b.txt")

		// old file
		createFile(t, oldFilePath)

		w, err := NewDirNotify(".", []*regexp.Regexp{})
		expectedErr := error(nil)
		if err != expectedErr {
			t.Fatalf("got %v, want %v", err, expectedErr)
		}
		defer w.Close()

		err = os.Rename(oldFilePath, newFilePath)
		if err != nil {
			t.Fatalf("unexpected error renaming %v to %v: %v", oldFilePath, newFilePath, err)
		}

		expectedEvent := RenameEvent{
			isDir:   false,
			path:    newFilePath,
			oldPath: oldFilePath,
		}

		select {
		case e := <-w.Events():
			if e != expectedEvent {
				t.Fatalf("got %v, want %v", e, expectedEvent)
			}
		case err := <-w.Errs():
			t.Fatalf("unexpected err: %v", err)
		case <-w.done:
			t.Fatal("channel closed")
		case <-time.After(eventTimeout):
			t.Fatal("timeout reached waiting for event")
		}
	})

	//
	t.Run("rename file from a watched directory to an unwatched directory", func(t *testing.T) {
		mkDirAll(t)
		defer rmDirAll(t)

		oldFilePath := path.Join("a/b/c/d/e", "a.txt")
		newFilePath := path.Join("f/g/h/i/j", "b.txt")

		// old file
		createFile(t, oldFilePath)

		w, err := NewDirNotify(".", []*regexp.Regexp{regexp.MustCompile("^f.*")})
		expectedErr := error(nil)
		if err != expectedErr {
			t.Fatalf("got %v, want %v", err, expectedErr)
		}
		defer w.Close()

		// new file
		err = os.Rename(oldFilePath, newFilePath)
		if err != nil {
			t.Fatalf("unexpected error renaming %v to %v: %v", oldFilePath, newFilePath, err)
		}

		expectedEvent := RenameEvent{
			isDir:   false,
			path:    "",
			oldPath: oldFilePath,
		}

		select {
		case e := <-w.Events():
			if e != expectedEvent {
				t.Fatalf("got %v, want %v", e, expectedEvent)
			}
		case err := <-w.Errs():
			t.Fatalf("unexpected err: %v", err)
		case <-w.done:
			t.Fatal("channel closed")
		case <-time.After(eventTimeout):
			t.Fatal("timeout reached waiting for event")
		}
	})

	//
	t.Run("rename file from an unwatched directory to a watched directory", func(t *testing.T) {
		mkDirAll(t)
		defer rmDirAll(t)

		oldFilePath := path.Join("f/g/h/i/j", "b.txt")
		newFilePath := path.Join("a/b/c/d/e", "a.txt")

		// old file
		createFile(t, oldFilePath)

		w, err := NewDirNotify(".", []*regexp.Regexp{regexp.MustCompile("^f.*")})
		expectedErr := error(nil)
		if err != expectedErr {
			t.Fatalf("got %v, want %v", err, expectedErr)
		}
		defer w.Close()

		// new file
		err = os.Rename(oldFilePath, newFilePath)
		if err != nil {
			t.Fatalf("unexpected error renaming %v to %v: %v", oldFilePath, newFilePath, err)
		}

		expectedEvent := RenameEvent{
			isDir:   false,
			path:    newFilePath,
			oldPath: "",
		}

		select {
		case e := <-w.Events():
			if e != expectedEvent {
				t.Fatalf("got %v, want %v", e, expectedEvent)
			}
		case err := <-w.Errs():
			t.Fatalf("unexpected err: %v", err)
		case <-w.done:
			t.Fatal("channel closed")
		case <-time.After(eventTimeout):
			t.Fatal("timeout reached waiting for event")
		}
	})

	//
	t.Run("rename file from an unwatched directory to an unwatched directory", func(t *testing.T) {
		mkDirAll(t)
		defer rmDirAll(t)

		oldFilePath := path.Join("f/g/h/i/j", "b.txt")
		newFilePath := path.Join("a/b/c/d/e", "a.txt")

		// old file
		createFile(t, oldFilePath)

		w, err := NewDirNotify(".", []*regexp.Regexp{
			regexp.MustCompile("^f.*"),
			regexp.MustCompile("^a.*"),
		})
		expectedErr := error(nil)
		if err != expectedErr {
			t.Fatalf("got %v, want %v", err, expectedErr)
		}
		defer w.Close()

		// new file
		err = os.Rename(oldFilePath, newFilePath)
		if err != nil {
			t.Fatalf("unexpected error renaming %v to %v: %v", oldFilePath, newFilePath, err)
		}

		select {
		case e := <-w.Events():
			t.Fatalf("unexpected event %v", e)
		case err := <-w.Errs():
			t.Fatalf("unexpected err: %v", err)
		case <-w.done:
			t.Fatal("channel closed")
		case <-time.After(eventTimeout):
		}
	})
}
