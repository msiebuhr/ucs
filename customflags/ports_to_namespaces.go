package customflags

// Flag Namespace Map is a custom type that allows the flag-package to get
// multiple parameters of string:int (or just int), allowing it to
// understand namespace mappings

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type Namespaces map[uint]string

// Pretty-prints namespace/port sets.
func (f *Namespaces) String() string {
	if len(*f) == 0 {
		return ""
	}
	pairs := make([]string, 0, len(*f))
	for port, ns := range *f {
		pairs = append(pairs, fmt.Sprintf("%s:%d", ns, port))
	}
	sort.Strings(pairs)
	return strings.Join(pairs, " ")
}

// Sets a namespace/port combination from either just a number, e.g. "5000",
// a namespace:port set, e.g. "alpha:5000", and finally, a set of these,
// e.g. "alpha:5000,beta:5001,5002"
func (f Namespaces) Set(s string) error {
	parts := strings.Split(s, ",")
	for _, part := range parts {
		err := f.setSingle(part)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f Namespaces) setSingle(s string) error {
	portStr := s
	name := s
	if strings.Contains(s, ":") {
		nsAndPort := strings.SplitN(s, ":", 2)
		portStr = nsAndPort[1]
		name = nsAndPort[0]
	}

	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return err
	}

	// Fail if port is already set
	if ns, ok := f[uint(port)]; ok {
		return fmt.Errorf("Port %d is already used for namespace '%s'", port, ns)
	}

	f[uint(port)] = name
	return nil
}
