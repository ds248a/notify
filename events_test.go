package notify

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"testing"
	"time"
)

// ------------------------
//   File Test
// ------------------------

var eventTimeout = time.Millisecond * 150

func mkDirAll(t *testing.T) {
	os.RemoveAll("a")
	os.RemoveAll("f")

	err := os.MkdirAll("a/b/c/d/e", os.ModeDir|os.ModePerm)
	if err != nil {
		t.Fatalf("unexpected error creating %v: %v", "a/b/c/d/e", err)
	}

	err = os.MkdirAll("f/g/h/i/j", os.ModeDir|os.ModePerm)
	if err != nil {
		t.Fatalf("unexpected error creating %v: %v", "f/g/h/i/j", err)
	}
}

func rmDirAll() {
	os.RemoveAll("a")
	os.RemoveAll("f")
}

//
func TestWatcher_createEvent(t *testing.T) {
	//
	t.Run("create_file", func(t *testing.T) {
		mkDirAll(t)
		defer rmDirAll()

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

		filePath := path.Join("a/b/c/d/e", "a.txt")
		_, err = os.Create(filePath)
		if err != nil {
			t.Fatalf("unexpected error creating %v: %v", filePath, err)
		}
		defer os.Remove(filePath)

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
		defer rmDirAll()

		w, err := NewDirNotify(".", []*regexp.Regexp{})
		expectedErr := error(nil)
		if err != expectedErr {
			t.Fatalf("got %v, want %v", err, expectedErr)
		}
		defer w.Close()

		dirPath := path.Join("a/b/c/d/e", "z")
		err = os.Mkdir(dirPath, os.ModeDir|os.ModePerm)
		if err != nil {
			t.Fatalf("unexpected error creating %v: %v", dirPath, err)
		}

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

		filePath := path.Join(dirPath, "a.txt")
		_, err = os.Create(filePath)
		if err != nil {
			t.Fatalf("unexpected error creating %v: %v", filePath, err)
		}

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
	t.Run("create_file_(regexp)", func(t *testing.T) {
		mkDirAll(t)
		defer rmDirAll()

		w, err := NewDirNotify(".", []*regexp.Regexp{regexp.MustCompile("^a/b/c/d/e/a.txt$")})
		expectedErr := error(nil)
		if err != expectedErr {
			t.Fatalf("got %v, want %v", err, expectedErr)
		}
		defer w.Close()

		filePath := path.Join("a/b/c/d/e", "a.txt")
		_, err = os.Create(filePath)
		if err != nil {
			t.Fatalf("unexpected error creating %v: %v", filePath, err)
		}
		defer os.Remove(filePath)

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
	t.Run("create_directory_(regexp)", func(t *testing.T) {
		mkDirAll(t)
		defer rmDirAll()

		w, err := NewDirNotify(".", []*regexp.Regexp{regexp.MustCompile("^a/b/c/d/e*")})
		expectedErr := error(nil)
		if err != expectedErr {
			t.Fatalf("got %v, want %v", err, expectedErr)
		}
		defer w.Close()

		dirPath := path.Join("a/b/c/d/e", "z")
		os.Mkdir(dirPath, os.ModeDir|os.ModePerm)

		select {
		case e := <-w.Events():
			t.Fatalf("unexpected event %v", e)
		case err := <-w.Errs():
			t.Fatalf("unexpected err: %v", err)
		case <-w.done:
			t.Fatal("channel closed")
		case <-time.After(eventTimeout):
		}

		filePath := path.Join(dirPath, "a.txt")
		_, err = os.Create(filePath)
		if err != nil {
			t.Fatalf("unexpected error creating %v: %v", filePath, err)
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
func TestWatcher_deleteEvent(t *testing.T) {
	//
	t.Run("delete_file", func(t *testing.T) {
		mkDirAll(t)
		defer rmDirAll()

		filePath := path.Join("a/b/c/d/e", "a.txt")
		if _, err := os.Create(filePath); err != nil {
			t.Fatalf("unexpected error creating %v: %v", filePath, err)
		}

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
		defer rmDirAll()

		dirPath := path.Join("a/b/c/d/e", "z")
		os.Mkdir(dirPath, os.ModeDir|os.ModePerm)

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

		fmt.Println("complie")
	})

	//
	t.Run("delete_file_(regexp)", func(t *testing.T) {
		mkDirAll(t)
		defer rmDirAll()

		filePath := path.Join("f/g/h/i/j", "foo")
		if _, err := os.Create(filePath); err != nil {
			t.Fatalf("unexpected error creating %v: %v", filePath, err)
		}

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
	t.Run("delete_directory_(regexp)", func(t *testing.T) {
		mkDirAll(t)
		defer rmDirAll()

		dirPath := path.Join("f/g/h/i/j", "foo")
		if _, err := os.Create(dirPath); err != nil {
			t.Fatalf("unexpected error creating %v: %v", dirPath, err)
		}

		w, err := NewDirNotify(".", []*regexp.Regexp{regexp.MustCompile("^" + dirPath + "$")})
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

		fmt.Println("complite regexp")
	})
}

//
func TestWatcher_modifyEvent(t *testing.T) {
	t.Run("modify_file", func(t *testing.T) {
		mkDirAll(t)
		defer rmDirAll()

		filePath := path.Join("a/b/c/d/e", "a.txt")
		if _, err := os.Create(filePath); err != nil {
			t.Fatalf("unexpected error creating %v: %v", filePath, err)
		}

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
	t.Run("modify_file_(regexp)", func(t *testing.T) {
		mkDirAll(t)
		defer rmDirAll()

		filePath := path.Join("a/b/c/d/e", "a.txt")
		if _, err := os.Create(filePath); err != nil {
			t.Fatalf("unexpected error creating %v: %v", filePath, err)
		}

		w, err := NewDirNotify(".", []*regexp.Regexp{regexp.MustCompile("^" + filePath + "$")})
		expectedErr := error(nil)
		if err != expectedErr {
			t.Fatalf("got %v, want %v", err, expectedErr)
		}
		defer w.Close()

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
	t.Run("modify_file_(regexp)_(2)", func(t *testing.T) {
		filePath := "foobar"
		_, err := os.Create(filePath)
		if err != nil {
			t.Fatalf("unexpected error creating %v: %v", filePath, err)
		}
		defer os.Remove("foobar")

		err = os.MkdirAll("a/foobar", os.ModeDir|os.ModePerm)
		if err != nil {
			t.Fatalf("unexpected error creating %v: %v", "a/b/c/d/e", err)
		}
		defer os.RemoveAll("a")

		file2Path := path.Join("a", "foobar", "b.txt")
		_, err = os.Create(file2Path)
		if err != nil {
			t.Fatalf("unexpected error creating %v: %v", file2Path, err)
		}

		w, err := NewDirNotify(".", []*regexp.Regexp{regexp.MustCompile("foobar$")})
		expectedErr := error(nil)
		if err != expectedErr {
			t.Fatalf("got %v, want %v", err, expectedErr)
		}
		defer w.Close()

		err = ioutil.WriteFile(filePath, []byte("foo"), os.ModePerm)
		if err != nil {
			t.Fatalf("unexpected error writing to %v: %v", filePath, err)
		}

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
		defer rmDirAll()

		oldFilePath := path.Join("a/b/c/d/e", "a.txt")
		newFilePath := path.Join("f/g/h/i/j", "b.txt")

		if _, err := os.Create(oldFilePath); err != nil {
			t.Fatalf("unexpected error creating %v: %v", oldFilePath, err)
		}

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
			OldPath: oldFilePath,
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
		defer rmDirAll()

		oldFilePath := path.Join("a/b/c/d/e", "a.txt")
		newFilePath := path.Join("f/g/h/i/j", "b.txt")

		if _, err := os.Create(oldFilePath); err != nil {
			t.Fatalf("unexpected error creating %v: %v", oldFilePath, err)
		}

		w, err := NewDirNotify(".", []*regexp.Regexp{regexp.MustCompile("^f.*")})
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
			path:    "",
			OldPath: oldFilePath,
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
		defer rmDirAll()

		oldFilePath := path.Join("f/g/h/i/j", "b.txt")
		newFilePath := path.Join("a/b/c/d/e", "a.txt")

		if _, err := os.Create(oldFilePath); err != nil {
			t.Fatalf("unexpected error creating %v: %v", oldFilePath, err)
		}

		workingDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("unexpect err: %v", err)
		}

		w, err := NewDirNotify(workingDir, []*regexp.Regexp{regexp.MustCompile("^" + workingDir + "/f.*")})
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
			path:    path.Join(workingDir, newFilePath),
			OldPath: "",
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
		defer rmDirAll()

		oldFilePath := path.Join("f/g/h/i/j", "b.txt")
		newFilePath := path.Join("a/b/c/d/e", "a.txt")

		if _, err := os.Create(oldFilePath); err != nil {
			t.Fatalf("unexpected error creating %v: %v", oldFilePath, err)
		}

		w, err := NewDirNotify(".", []*regexp.Regexp{
			regexp.MustCompile("^f.*"),
			regexp.MustCompile("^a.*"),
		})
		expectedErr := error(nil)
		if err != expectedErr {
			t.Fatalf("got %v, want %v", err, expectedErr)
		}
		defer w.Close()

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
