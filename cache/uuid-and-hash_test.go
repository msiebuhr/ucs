package cache

import (
	"testing"
	//"strings"
	"bytes"
)

//func test_uuidandhash_stringer() {}

func TestUUIDAndHashReadfrom(t *testing.T) {
	raw := []byte{
		1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}
	out := "010000000000000000000000000000ff-00000000000000000000000000000000"
	if len(raw) != 32 {
		t.Fatal("Length not 32!")
	}
	if len(out) != 32*2+1 {
		t.Fatal("Length not", 32*2+1)
	}

	r := bytes.NewReader(raw)

	uh := UUIDAndHash{}

	// Check we can read it in
	uh.ReadFrom(r)

	// Does Bytes() read the same thing out
	if !bytes.Equal(uh.Bytes(), raw) {
		t.Errorf("Expected ReadFrom to return\n\t%x\n,got\n\t%x", raw, uh.Bytes())
	}

	// Does it come out as a string?
	if uh.String() != out {
		t.Errorf("Expected ReadFrom to return\n\t%s\n,got\n\t%s", out, uh.String())
	}
}
