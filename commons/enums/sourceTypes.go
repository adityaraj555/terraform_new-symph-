package enums

type Sources string

const (
	MeasurementAutomation = "MA"
	AutoImageSelection    = "AIS"
	SIM                   = "SIM"
)

func SourcesList() []string {
	return []string{MeasurementAutomation, AutoImageSelection, SIM}
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
