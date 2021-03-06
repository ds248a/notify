package notify

import (
	"path"
	"testing"
)

// ------------------------
//   Dir Test
// ------------------------

// Создание корневой директории.
func TestWatchDirsTreeSetRoot(t *testing.T) {
	wdt := newWatchDirsTree()

	rootWd := 0
	wdt.setRoot(".", rootWd)

	if wdt.root == nil {
		t.Errorf("got %v, want %v", nil, "non-nil value")
	}

	if wdt.root.name != "" {
		t.Errorf("got %v, want %v", wdt.root.name, "empty string")
	}

	if wdt.root.wd != rootWd {
		t.Errorf("got %v, want %v", wdt.root.wd, rootWd)
	}
}

// Добавление каталога.
func TestWatchDirsTreeAddHasGet(t *testing.T) {
	wdt := newWatchDirsTree()
	wdt.setRoot(".", 0)

	dirWd := 1
	dirName := "some"

	wdt.add(dirWd, dirName, wdt.root.wd)

	has := wdt.has(dirWd)
	if !has {
		t.Errorf("got %v, want %v", false, true)
	}

	d := wdt.get(dirWd)
	if d == nil {
		t.Fatalf("got %v, want %v", nil, "non-nil value")
	}

	if d.wd != dirWd {
		t.Errorf("got %v, want %v", d.wd, dirWd)
	}

	if d.name != dirName {
		t.Errorf("got %v, want %v", d.name, dirName)
	}

	if d.parent != wdt.root {
		t.Errorf("got %v, want %v", d.parent, wdt.root)
	}
}

// Удаление каталога.
func TestWatchDirsTreeRmHasGet(t *testing.T) {
	wdt := newWatchDirsTree()
	wdt.setRoot(".", 0)

	dirWd := 1
	dirName := "some"

	wdt.add(dirWd, dirName, wdt.root.wd)
	wdt.rm(dirWd)

	has := wdt.has(dirWd)
	if has {
		t.Errorf("got %v, want %v", true, false)
	}

	d := wdt.get(dirWd)

	if d != nil {
		t.Fatalf("got %v, want %v", d, nil)
	}
}

// Добавление подкаталога и удаление родительского.
func TestWatchDirsTreeRmHasGet_child(t *testing.T) {
	wdt := newWatchDirsTree()
	wdt.setRoot(".", 0)

	parentWd := 1
	parentName := "parent"
	dirWd := 2
	dirName := "child"

	wdt.add(parentWd, parentName, wdt.root.wd)
	wdt.add(dirWd, dirName, parentWd)
	wdt.rm(parentWd)

	has := wdt.has(dirWd)
	if has {
		t.Errorf("got %v, want %v", true, false)
	}

	d := wdt.get(dirWd)
	if d != nil {
		t.Fatalf("got %v, want %v", d, nil)
	}
}

// Манипуляции с каталогами.
func TestWatchDirsTreeMvInvalidate(t *testing.T) {
	// перемещение
	t.Run("only parent", func(t *testing.T) {
		wdt := newWatchDirsTree()
		wdt.setRoot(".", 0)

		dir1Wd := 1
		dir1Name := "some"
		dir1ParentWd := wdt.root.wd
		dir2Wd := 2
		dir2Name := "foo"
		dir2ParentWd := dir1Wd
		dir3Wd := 3
		dir3Name := "bar"
		dir3ParentWd := dir1Wd
		dir4Wd := 4
		dir4Name := "fourth"
		dir4ParentWd := dir3Wd

		wdt.add(dir1Wd, dir1Name, dir1ParentWd)
		wdt.add(dir2Wd, dir2Name, dir2ParentWd)
		wdt.add(dir3Wd, dir3Name, dir3ParentWd)
		wdt.add(dir4Wd, dir4Name, dir4ParentWd)

		// кеширование
		wdt.path(dir4Wd)

		// реализация перемещения
		wdt.mv(dir4Wd, dir2Wd, "")
		/*
			before moving:
			- root
				- dir1
					- dir2
					- dir3
						- dir4
			after moving:
			- root
				- dir1
					- dir2
						- dir4
					- dir3
		*/

		expectedDir4Path := path.Join(
			dir1Name,
			dir2Name,
			dir4Name,
		)
		dir4Path := wdt.path(dir4Wd)

		_, isChildOfDir3 := wdt.get(dir3Wd).children[dir3Name]
		expectedIsChildOfDir3 := false

		if dir4Path != expectedDir4Path {
			t.Errorf("got %v, want %v", dir4Path, expectedDir4Path)
		}

		if isChildOfDir3 != expectedIsChildOfDir3 {
			t.Errorf("got %v, want %v", isChildOfDir3, expectedIsChildOfDir3)
		}
	})

	// переименование
	t.Run("only name", func(t *testing.T) {
		wdt := newWatchDirsTree()
		wdt.setRoot(".", 0)

		dir1Wd := 1
		dir1Name := "some"
		dir1ParentWd := wdt.root.wd
		dir2Wd := 2
		dir2Name := "foo"
		dir2ParentWd := dir1Wd
		dir3Wd := 3
		dir3Name := "bar"
		dir3ParentWd := dir1Wd
		dir4Wd := 4
		dir4Name := "fourth"
		dir4NewName := "the_fourth"
		dir4ParentWd := dir3Wd

		wdt.add(dir1Wd, dir1Name, dir1ParentWd)
		wdt.add(dir2Wd, dir2Name, dir2ParentWd)
		wdt.add(dir3Wd, dir3Name, dir3ParentWd)
		wdt.add(dir4Wd, dir4Name, dir4ParentWd)

		// кеширование
		wdt.path(dir4Wd)

		// реализация переименования
		wdt.mv(dir4Wd, -1, dir4NewName)

		expectedDir4Path := path.Join(
			dir1Name,
			dir3Name,
			dir4NewName,
		)
		dir4Path := wdt.path(dir4Wd)

		if dir4Path != expectedDir4Path {
			t.Errorf("got %v, want %v", dir4Path, expectedDir4Path)
		}
	})

	// перемещение с переименованием
	t.Run("name and parent", func(t *testing.T) {
		wdt := newWatchDirsTree()
		wdt.setRoot(".", 0)

		dir1Wd := 1
		dir1Name := "some"
		dir1ParentWd := wdt.root.wd
		dir2Wd := 2
		dir2Name := "foo"
		dir2ParentWd := dir1Wd
		dir3Wd := 3
		dir3Name := "bar"
		dir3ParentWd := dir1Wd
		dir4Wd := 4
		dir4Name := "fourth"
		dir4NewName := "the_fourth"
		dir4ParentWd := dir3Wd

		wdt.add(dir1Wd, dir1Name, dir1ParentWd)
		wdt.add(dir2Wd, dir2Name, dir2ParentWd)
		wdt.add(dir3Wd, dir3Name, dir3ParentWd)
		wdt.add(dir4Wd, dir4Name, dir4ParentWd)

		// кеширование
		wdt.path(dir4Wd)

		wdt.mv(dir4Wd, dir2Wd, dir4NewName)
		/*
			before moving:
			- root
				- dir1
					- dir2
					- dir3
						- dir4
			after moving:
			- root
				- dir1
					- dir2
						- dir4
					- dir3
		*/

		expectedDir4Path := path.Join(
			dir1Name,
			dir2Name,
			dir4NewName,
		)
		dir4Path := wdt.path(dir4Wd)

		_, isChildOfDir3 := wdt.get(dir3Wd).children[dir3Name]
		expectedIsChildOfDir3 := false

		if dir4Path != expectedDir4Path {
			t.Errorf("got %v, want %v", dir4Path, expectedDir4Path)
		}

		if isChildOfDir3 != expectedIsChildOfDir3 {
			t.Errorf("got %v, want %v", isChildOfDir3, expectedIsChildOfDir3)
		}
	})
}

// Компановка относительного и абсолютного путей с использованием кеширования.
func TestWatchDirsTreePath(t *testing.T) {
	t.Run("root relative path", func(t *testing.T) {
		wdt := newWatchDirsTree()
		wdt.setRoot(".", 0)

		dir1Wd := 1
		dir1Name := "some"
		dir1ParentWd := wdt.root.wd
		dir2Wd := 2
		dir2Name := "foo"
		dir2ParentWd := dir1Wd
		dir3Wd := 3
		dir3Name := "bar"
		dir3ParentWd := dir2Wd

		wdt.add(dir1Wd, dir1Name, dir1ParentWd)
		wdt.add(dir2Wd, dir2Name, dir2ParentWd)
		wdt.add(dir3Wd, dir3Name, dir3ParentWd)

		expectedDir3Path := path.Join(
			dir1Name,
			dir2Name,
			dir3Name,
		)
		// использование кеширования
		dir3Path := wdt.path(dir3Wd) // path, _ := wdt.cache.path(wd)

		if dir3Path != expectedDir3Path {
			t.Errorf("got %v, want %v", dir3Path, expectedDir3Path)
		}
	})

	t.Run("root absolute path", func(t *testing.T) {
		wdt := newWatchDirsTree()
		wdt.setRoot("/home/usr", 0)

		dir1Wd := 1
		dir1Name := "some"
		dir1ParentWd := wdt.root.wd
		dir2Wd := 2
		dir2Name := "foo"
		dir2ParentWd := dir1Wd
		dir3Wd := 3
		dir3Name := "bar"
		dir3ParentWd := dir2Wd

		wdt.add(dir1Wd, dir1Name, dir1ParentWd)
		wdt.add(dir2Wd, dir2Name, dir2ParentWd)
		wdt.add(dir3Wd, dir3Name, dir3ParentWd)

		expectedDir3Path := path.Join(
			"/home/usr",
			dir1Name,
			dir2Name,
			dir3Name,
		)
		// использование кеширования
		dir3Path := wdt.path(dir3Wd) // path, _ := wdt.cache.path(wd)

		if dir3Path != expectedDir3Path {
			t.Errorf("got %v, want %v", dir3Path, expectedDir3Path)
		}
	})
}

// Поиск скомпанованного пути по его дескрипрору, с использванием кеширования.
func TestWatchDirsTreeFind(t *testing.T) {
	t.Run("root relative path", func(t *testing.T) {
		wdt := newWatchDirsTree()
		wdt.setRoot(".", 0)

		dir1Wd := 1
		dir1Name := "some"
		dir1ParentWd := wdt.root.wd
		dir2Wd := 2
		dir2Name := "foo"
		dir2ParentWd := dir1Wd
		dir3Wd := 3
		dir3Name := "bar"
		dir3ParentWd := dir2Wd

		// "./some/foo/bar"
		wdt.add(dir1Wd, dir1Name, dir1ParentWd)
		wdt.add(dir2Wd, dir2Name, dir2ParentWd)
		wdt.add(dir3Wd, dir3Name, dir3ParentWd)

		dir3Path := path.Join(
			dir1Name,
			dir2Name,
			dir3Name,
		)
		// использование кеширования
		dir3 := wdt.get(dir3Wd)
		findRes := wdt.find(dir3Path) // wd, ok := wdt.cache.wd(path)

		if findRes != dir3 {
			t.Errorf("got %v, want %v", findRes, dir3)
		}
	})

	t.Run("root absolute path", func(t *testing.T) {
		wdt := newWatchDirsTree()
		wdt.setRoot("/home/user", 0)

		dir1Wd := 1
		dir1Name := "some"
		dir1ParentWd := wdt.root.wd
		dir2Wd := 2
		dir2Name := "foo"
		dir2ParentWd := dir1Wd
		dir3Wd := 3
		dir3Name := "bar"
		dir3ParentWd := dir2Wd

		// "/home/user/some/foo/bar"
		wdt.add(dir1Wd, dir1Name, dir1ParentWd)
		wdt.add(dir2Wd, dir2Name, dir2ParentWd)
		wdt.add(dir3Wd, dir3Name, dir3ParentWd)

		dir3Path := path.Join(
			"/home/user",
			dir1Name,
			dir2Name,
			dir3Name,
		)
		// использование кеширования
		dir3 := wdt.get(dir3Wd)
		findRes := wdt.find(dir3Path) // wd, ok := wdt.cache.wd(path)

		if findRes != dir3 {
			t.Errorf("got %v, want %v", findRes, dir3)
		}
	})
}

// Поиск не существующей структуры каталогов (пути).
func TestWatchDirsTreeFind_notFound(t *testing.T) {
	wdt := newWatchDirsTree()
	wdt.setRoot(".", 0)

	dir1Wd := 1
	dir1Name := "some"
	dir1ParentWd := wdt.root.wd
	dir2Wd := 2
	dir2Name := "foo"
	dir2ParentWd := dir1Wd
	dir3Wd := 3
	dir3Name := "bar"
	dir3ParentWd := dir2Wd

	// "./some/foo/bar"
	wdt.add(dir1Wd, dir1Name, dir1ParentWd)
	wdt.add(dir2Wd, dir2Name, dir2ParentWd)
	wdt.add(dir3Wd, dir3Name, dir3ParentWd)

	dir3Path := path.Join(
		dir1Name,
		dir2Name,
		dir3Name,
	)
	findRes := wdt.find(path.Join(dir3Path, "aa"))

	if findRes != nil {
		t.Errorf("got %v, want %v", findRes, nil)
	}
}
