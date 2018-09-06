package cache

import (
	"errors"
	"io"
)

type Line struct {
	// TODO: Do we really need a self-reference?
	uuidAndHash []byte

	// Various kinds of data...
	Asset    *[]byte
	Info     *[]byte
	Resource *[]byte
}

func (c Line) Get(kind Kind) ([]byte, bool) {
	var ptr *[]byte
	switch kind {
	case KIND_ASSET:
		ptr = c.Asset
	case KIND_INFO:
		ptr = c.Info
	case KIND_RESOURCE:
		ptr = c.Resource
	default:
		return []byte{}, false
	}

	if ptr == nil {
		return nil, false
	}

	return *ptr, true
}

func (c *Line) Put(kind Kind, data []byte) error {
	switch kind {
	case KIND_ASSET:
		c.Asset = &data
	case KIND_INFO:
		c.Info = &data
	case KIND_RESOURCE:
		c.Resource = &data
	default:
		return errors.New("Trying to put unknown resource")
	}
	return nil
}

func (l Line) Size() int64 {
	size := 0

	if l.Asset != nil {
		size += len(*l.Asset)
	}
	if l.Info != nil {
		size += len(*l.Info)
	}
	if l.Resource != nil {
		size += len(*l.Resource)
	}

	return int64(size)
}

// Put data from a reader into the cacheline. The kind and number of bytes
// to be read as well
func (c *Line) PutReader(kind Kind, size uint64, r io.Reader) error {
	tmp := make([]byte, size)
	_, err := io.ReadFull(r, tmp)
	if err != nil {
		return err
	}
	return c.Put(kind, tmp)
}
