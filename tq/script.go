package tq

type Script struct {
	Name      string
	Timestamp int
	Index     int
	Data      string
}

type ByAge []Script

func (a ByAge) Len() int           { return len(a) }
func (a ByAge) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByAge) Less(i, j int) bool { return a[i].Timestamp < a[j].Timestamp }

type ByManifest []Script

func (a ByManifest) Len() int           { return len(a) }
func (a ByManifest) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByManifest) Less(i, j int) bool { return a[i].Index < a[j].Index }
