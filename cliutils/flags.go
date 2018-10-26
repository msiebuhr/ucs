package cliutils

// Flag Namespace Map is a custom type that allows the flag-package to get
// multiple parameters of string:int (or just int), allowing it to
// understand namespace mappings

import (
	"fmt"
	"strconv"
	"strings"
)

type FlagNSMap map[string]uint

func (f *FlagNSMap) String() string {
	if len(*f) == 0 {
		return ""
	}
	str := ""
	for ns, port := range *f {
		str = fmt.Sprintf("%s %s:%d", str, ns, port)
	}
	return str[1:]
}
func (f FlagNSMap) Set(s string) error {
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
