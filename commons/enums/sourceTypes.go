package enums

import "strings"

type Sources string

const (
	PropertyDataOrchestrator = "PDO"
	AutoImageSelection       = "AIS"
)

func SourcesList() []string {
	return []string{PropertyDataOrchestrator, AutoImageSelection}
}

func (s Sources) String() string {
	l := SourcesList()
	x := strings.ToLower(string(s))
	for _, v := range l {
		if v == x {
			return x
		}
	}
	return ""
}
