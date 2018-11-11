package cache

import (
	"fmt"
	"io"
)

type UUIDAndHash struct {
	uuid [16]byte
	hash [16]byte
}

// Stringer returns HASH-UUID
func (u UUIDAndHash) String() string {
	return fmt.Sprintf("%x-%x", u.uuid, u.hash)
}

// Bytes returns all the bytes in UUID and the HASH. 32 bytes in all.
func (b UUIDAndHash) Bytes() []byte {
	out := make([]byte, 32)
	for i, v := range b.uuid {
		out[i] = v
	}
	for i, v := range b.hash {
		out[i+16] = v
	}
	return out
}

func (b UUIDAndHash) WriteTo(w io.Writer) (int64, error) {
	c, err := w.Write(b.Bytes())
	return int64(c), err
}

// ReadFrom implements io.ReaderFrom
func (u *UUIDAndHash) ReadFrom(r io.Reader) (int64, error) {
	bytes := make([]byte, 32)
	count, err := io.ReadFull(r, bytes)
	if err != nil {
		return int64(count), err
	}
	copy(u.uuid[:], bytes[:16])
	copy(u.hash[:], bytes[16:])
	return int64(count), nil
}
