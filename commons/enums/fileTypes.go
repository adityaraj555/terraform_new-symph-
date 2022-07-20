package enums

import "strings"

type FileType string

const (
	NorthImage = "22"
	SouthImage = "23"
	EastImage  = "24"
	WestImage  = "25"
	TopImage   = "6"
)

func FileTypeList() []string {
	return []string{NorthImage, SouthImage, EastImage, WestImage, TopImage}
}

func (a FileType) String() string {
	l := FileTypeList()
	x := strings.ToLower(string(a))
	for _, v := range l {
		if v == x {
			return x
		}
	}
	return ""
}
