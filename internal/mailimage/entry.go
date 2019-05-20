package mailimage

type entry struct {
	ID        int
	From      string
	Subject   string
	Text      string
	Extension string
	Created   string
}

type byCreated []entry

func (a byCreated) Len() int           { return len(a) }
func (a byCreated) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byCreated) Less(i, j int) bool { return a[i].Created < a[j].Created }
