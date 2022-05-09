package enums

import "strings"

type RequestMethod string

const (
	GET    = "GET"
	PUT    = "PUT"
	DELETE = "DELETE"
	POST   = "POST"
)

func RequestMethodList() []string {
	return []string{GET, PUT, POST, DELETE}
}

func (r RequestMethod) String() string {
	l := RequestMethodList()
	x := strings.ToUpper(string(r))
	for _, m := range l {
		if x == m {
			return x
		}
	}
	return ""
}
