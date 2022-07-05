package enums

import "strings"

type CallType string

const (
	HipsterCT = "hipster"
	LegacyCT  = "eagleflow"
	LambdaCT  = "lambda"
)

func CallTypeList() []string {
	return []string{HipsterCT, LegacyCT, LambdaCT}
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
