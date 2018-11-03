package customflags

// Flag Namespace Map is a custom type that allows the flag-package to get
// multiple parameters of string:int (or just int), allowing it to
// understand namespace mappings

import (
	"fmt"
	"strconv"
	"strings"
)

type Namespaces map[string]uint

// Pretty-prints namespace/port sets.
func (f *Namespaces) String() string {
	if len(*f) == 0 {
		return ""
	}
	str := ""
	for ns, port := range *f {
		str = fmt.Sprintf("%s %s:%d", str, ns, port)
	}
	return str[1:]
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

	f[name] = uint(port)
	return nil
}

