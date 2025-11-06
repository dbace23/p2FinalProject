package rental

import "strconv"

func strconvParseInt(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}
