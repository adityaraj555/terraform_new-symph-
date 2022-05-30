package enums

import "strings"

type TaskStatus string

const (
	StatusSuccess = "success"
	StatusFailure = "failure"
	StatusRework  = "rework"
	StatusFailed  = "failed"
)

func TaskStatusList() []string {
	return []string{StatusSuccess, StatusFailure, StatusRework, StatusFailed}
}

func (ts TaskStatus) String() string {
	l := TaskStatusList()
	x := strings.ToLower(string(ts))
	for _, v := range l {
		if v == x {
			return x
		}
	}
	return ""
}
