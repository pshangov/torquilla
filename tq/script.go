package tq

type Script struct {
	Name      string
	Timestamp int
	Data      string
}

type ByAge []Script

func (a ByAge) Len() int           { return len(a) }
func (a ByAge) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByAge) Less(i, j int) bool { return a[i].Timestamp < a[j].Timestamp }
