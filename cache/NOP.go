package cache

import "io"

// Cache that discards all given data and never returns any hits
type NOP struct{}

// Create a new NOP instance
func NewNOP() *NOP {
	return &NOP{}
}

// Discards all given data
func (n *NOP) Put(ns string, uuidAndHash []byte, data Line) error {
	return nil
}

// Return emoty readers
func (n *NOP) Get(ns string, k Kind, uuidAndHash []byte) (int64, io.ReadCloser, error) {
	return 0, nil, nil
	//return false, ioutil.NopCloser(strings.NewReader("")), nil)
}
