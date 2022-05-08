package enums

type TaskStatus string

const (
	StatusSuccess = "success"
	StatusFailure = "failure"
	StatusRework  = "rework"
)

func TaskStatusList() []string {
	return []string{StatusSuccess, StatusFailure, StatusRework}
}

func (ts TaskStatus) String() string {
	l := TaskStatusList()
	x := string(ts)
	for _, v := range l {
		if v == x {
			return x
		}
	}
	return ""
}
