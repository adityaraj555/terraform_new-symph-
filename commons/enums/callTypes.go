package enums

import "strings"

type CallType string

const (
	HipsterCT = "hipster"
	LegacyCT  = "eagleflow"
)

func CallTypeList() []string {
	return []string{HipsterCT, LegacyCT}
}

func (ct CallType) String() string {
	l := CallTypeList()
	x := strings.ToLower(string(ct))
	for _, m := range l {
		if x == m {
			return x
		}
	}
	return ""
}
