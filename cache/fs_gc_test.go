package cache

import (
	"bytes"
	//"reflect"
	//"time"
	"fmt"
	"math/rand"
	"os"
	"testing"
)

func TestFSFindApproximateOldFiles(t *testing.T) {
	f, err := NewFS(func(f *FS) { f.Quota = 100; f.Basepath = "./testdata/gc-test/" })
	if err != nil {
		t.Fatalf("Error creating FS: %s", err)
	}
	os.RemoveAll(f.Basepath) // Make sure we start empty
	defer func() { os.RemoveAll(f.Basepath) }()

	// Insert three keys, uuid.asset, uuid.info and uuid.resource
	key := make([]byte, 32)
	rand.Read(key)

	// Insert one of each key
	tx := f.PutTransaction("list-old-files", key)
	for i, kind := range []Kind{KIND_INFO, KIND_ASSET, KIND_RESOURCE} {
		err := tx.Put(1, kind, bytes.NewReader([]byte{byte(i)}))
		if err != nil {
			t.Fatalf("Unexpected error calling Put(): %s", err)
		}
	}
	tx.Commit()

	// Do a single scan and confirm the numbers are right
	size, entries, err := findApproximateOldFiles(f.Basepath)

	if err != nil {
		t.Errorf("Unexpected error calling findApproximateOldFiles(): %s", err)
	}

	if size != 3 {
		t.Errorf("Expected cache size to be 3, got %d", size)
	}

	if len(entries) != 1 {
		t.Errorf("Expected one entry, got %d", len(entries))
	}

	expected := fsCacheEntry{
		ns:          "list-old-files",
		uuidAndHash: fmt.Sprintf("%x-%x.info", key[0:16], key[16:32]),
		size:        3,
		//time: time.Now(), // TODO: Grab from FS?
	}

	if entries[0].ns != expected.ns {
		t.Errorf("Expected entry ns to be %s, got %s", expected.ns, entries[0].ns)
	}

	if entries[0].uuidAndHash != expected.uuidAndHash {
		t.Errorf("Expected entry UUID+Hash to be\n\t%s\ngot\n\t%s", expected.uuidAndHash, entries[0].uuidAndHash)
	}
}
