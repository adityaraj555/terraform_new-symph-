package enums

type Sources string

const (
	MeasurementAutomation = "MA"
	AutoImageSelection    = "AIS"
)

func SourcesList() []string {
	return []string{MeasurementAutomation, AutoImageSelection}
}

func (s Sources) String() string {
	l := SourcesList()
	x := string(s)
	for _, v := range l {
		if v == x {
			return x
		}
	}
	return ""
}
