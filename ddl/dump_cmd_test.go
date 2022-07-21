package ddl

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bokwoon95/sqddl/internal/testutil"
)

func TestDumpCmd(t *testing.T) {
	t.Run("tgz", func(t *testing.T) {
		t.Parallel()
		dsn := "sqlite:file:/" + t.Name() + "?vfs=memdb&_foreign_keys=false"
		// Load the data.
		loadCmd, err := LoadCommand("-db", dsn, "testdata/sqlite/dump.tgz")
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		loadCmd.Stderr = io.Discard
		loadCmd.DirFS = testFS
		loadCmd.db = "" // Keep database open after running command.
		defer loadCmd.DB.Close()
		err = filepath.WalkDir("sqlite_migrations/repeatable", func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".sql") {
				loadCmd.Filenames = append(loadCmd.Filenames, filepath.ToSlash(path))
			}
			return nil
		})
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		err = loadCmd.Run()
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		// Dump the data.
		tempDir := t.TempDir()
		dumpCmd, err := DumpCommand(
			"-db", dsn,
			"-output-dir", tempDir,
			"-tgz", "dump.tgz",
		)
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		dumpCmd.Stderr = io.Discard
		err = dumpCmd.Run()
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		// Compare the dumps.
		assertFilesAreIdentical(t, filepath.Join(tempDir, "dump.tgz"), "testdata/sqlite/dump.tgz")
	})

	t.Run("zip", func(t *testing.T) {
		t.Parallel()
		dsn := "sqlite:file:/" + t.Name() + "?vfs=memdb&_foreign_keys=false"
		// Load the data.
		loadCmd, err := LoadCommand("-db", dsn, "testdata/sqlite/dump.zip")
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		loadCmd.Stderr = io.Discard
		loadCmd.db = "" // Keep database open after running command.
		defer loadCmd.DB.Close()
		err = filepath.WalkDir("sqlite_migrations/repeatable", func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".sql") {
				loadCmd.Filenames = append(loadCmd.Filenames, filepath.ToSlash(path))
			}
			return nil
		})
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		err = loadCmd.Run()
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		// Dump the data.
		tempDir := t.TempDir()
		dumpCmd, err := DumpCommand(
			"-db", dsn,
			"-output-dir", tempDir,
			"-zip", "dump.zip",
		)
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		dumpCmd.Stderr = io.Discard
		err = dumpCmd.Run()
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		// Compare the dumps.
		assertFilesAreIdentical(t, filepath.Join(tempDir, "dump.zip"), "testdata/sqlite/dump.zip")
	})

	t.Run("subset", func(t *testing.T) {
		t.Parallel()
		dsn := "sqlite:file:/" + t.Name() + "?vfs=memdb&_foreign_keys=true"
		// Load the data.
		loadCmd, err := LoadCommand(
			"-db", dsn,
			"testdata/sqlite/schema.sql",
			"csv_testdata",
			"testdata/sqlite/indexes.sql",
			"testdata/sqlite/constraints.sql",
		)
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		loadCmd.Stderr = io.Discard
		loadCmd.db = "" // Keep database open after running command.
		defer loadCmd.DB.Close()
		err = loadCmd.Run()
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		// Dump the data.
		tempDir := t.TempDir()
		dumpCmd, err := DumpCommand(
			"-db", dsn,
			"-output-dir", tempDir,
			"-data-only",
			"-subset", "SELECT {*} FROM {film} ORDER BY film_id LIMIT 10",
			"-subset", "SELECT {*} FROM {actor}",
		)
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		dumpCmd.Stderr = io.Discard
		err = dumpCmd.Run()
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		assertCSVsAreIdentical(t, tempDir, "testdata/subset", nil, nil)
	})

	t.Run("extended_subset", func(t *testing.T) {
		t.Parallel()
		dsn := "sqlite:file:/" + t.Name() + "?vfs=memdb&_foreign_keys=true"
		// Load the data.
		loadCmd, err := LoadCommand(
			"-db", dsn,
			"testdata/sqlite/schema.sql",
			"csv_testdata",
			"testdata/sqlite/indexes.sql",
			"testdata/sqlite/constraints.sql",
		)
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		loadCmd.Stderr = io.Discard
		loadCmd.db = "" // Keep database open after running command.
		defer loadCmd.DB.Close()
		err = loadCmd.Run()
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		// Dump the data.
		tempDir := t.TempDir()
		dumpCmd, err := DumpCommand(
			"-db", dsn,
			"-output-dir", tempDir,
			"-data-only",
			"-extended-subset", "SELECT {*} FROM {film} ORDER BY film_id LIMIT 10",
		)
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		dumpCmd.Stderr = io.Discard
		err = dumpCmd.Run()
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		assertCSVsAreIdentical(t, tempDir, "testdata/extended_subset", nil, nil)
	})
}

func assertFilesAreIdentical(t *testing.T, gotFilename, wantFilename string) {
	// filename1
	gotFile, err := os.Open(gotFilename)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	defer gotFile.Close()
	gotHash := sha256.New()
	_, err = io.Copy(gotHash, gotFile)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	gotChecksum := hex.EncodeToString(gotHash.Sum(nil))
	// filename2
	wantFile, err := os.Open(wantFilename)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	defer wantFile.Close()
	wantHash := sha256.New()
	_, err = io.Copy(wantHash, wantFile)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	wantChecksum := hex.EncodeToString(wantHash.Sum(nil))
	// Are they the same?
	if gotChecksum == wantChecksum {
		return
	}
	t.Errorf("%s and %s are different", gotFilename, wantFilename)
	gotFile, err = os.Open(gotFilename)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	defer gotFile.Close()
	outFilename := time.Now().Format("20060102150405") + filepath.Ext(gotFilename)
	outFile, err := os.OpenFile(outFilename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	defer outFile.Close()
	_, err = io.Copy(outFile, gotFile)
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	err = outFile.Close()
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	fmt.Fprintf(os.Stderr, "%[1]s has been copied to %[2]s, please compare the differences between %[2]s and %[3]s\n", gotFilename, outFilename, wantFilename)
}

func assertCSVsAreIdentical(t *testing.T, gotDir, wantDir string, filepairs [][2]string, transforms map[string]func([]string) []string) {
	err := fs.WalkDir(os.DirFS(wantDir), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != "." {
			return fs.SkipDir
		}
		if !strings.HasSuffix(path, ".csv") {
			return nil
		}
		wantFilename := filepath.Join(wantDir, path)
		gotFilename := filepath.Join(gotDir, path)
		_, err = os.Stat(gotFilename)
		if errors.Is(err, os.ErrNotExist) {
			t.Errorf(testutil.Callers()+" %q is missing from dump", gotFilename)
			return nil
		}
		if err != nil {
			return err
		}
		filepairs = append(filepairs, [2]string{gotFilename, wantFilename})
		return nil
	})
	if err != nil {
		t.Fatal(testutil.Callers(), err)
	}
	for _, filepair := range filepairs {
		filename := filepath.Base(filepair[0])
		if transform, ok := transforms[filename]; ok {
			err = rewriteCSV(filepair[0], transform)
			if err != nil {
				t.Fatal(testutil.Callers(), err)
			}
		}
		gotContents, err := os.ReadFile(filepair[0])
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		wantContents, err := os.ReadFile(filepair[1])
		if err != nil {
			t.Fatal(testutil.Callers(), err)
		}
		if diff := testutil.Diff(string(gotContents), string(wantContents)); diff != "" {
			t.Error(testutil.Callers(), "\n"+filename, diff)
		}
	}
}
