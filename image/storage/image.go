package storage

type Image struct {
	id       int
	width    int
	height   int
	filesize int
	src      string
}

func (s Image) Id() int {
	return s.id
}

func (s Image) Src() string {
	return s.src
}

func (s Image) Width() int {
	return s.width
}

func (s Image) Height() int {
	return s.height
}

func (s Image) FileSize() int {
	return s.filesize
}
