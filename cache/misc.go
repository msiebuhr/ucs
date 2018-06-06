package cache

import (
	"fmt"
)

func PrettyUuidAndHash(d []byte) string {
	return fmt.Sprintf("%x/%x", d[:16], d[17:])
}
