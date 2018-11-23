package cache

import (
	"io"
)

type NOPTransaction struct{}

func (nt NOPTransaction) Put(size int64, k Kind, r io.Reader) error {
	return nil
}

func (nt NOPTransaction) Commit() error { return nil }
func (nt NOPTransaction) Abort() error  { return nil }

// Cache that discards all given data and never returns any hits
type NOP struct{}

// Create a new NOP instance
func NewNOP() *NOP {
	return &NOP{}
}

// Return emoty readers
func (n *NOP) Get(ns string, k Kind, uuidAndHash []byte) (int64, io.ReadCloser, error) {
	return 0, nil, nil
	//return false, ioutil.NopCloser(strings.NewReader("")), nil)
}

func (n NOP) PutTransaction(ns string, uuidAndHash []byte) Transaction { return &NOPTransaction{} }
