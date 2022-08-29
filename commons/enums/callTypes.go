package enums

import "strings"

type CallType string

const (
	HipsterCT = "hipster"
	LegacyCT  = "eagleflow"
	LambdaCT  = "lambda"
	SQSCT     = "sqs"
)

func CallTypeList() []string {
	return []string{HipsterCT, LegacyCT, LambdaCT, SQSCT}
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
