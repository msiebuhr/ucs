package cache

import (
)

// Cache that discards all given data and never returns any hits
type NOP struct { }

// Create a new NOP instance
func NewNOP() *NOP {
	return &NOP{}
}

// Discards all given data
func (n *NOP) Put(uuidAndHash []byte, data Line) error {
	return nil
}

// Misses getting anything
func (n *NOP) Get(kind Kind, uuidAndHash []byte) ([]byte, error) {
	return []byte{}, nil
}
