package storage

type Image struct {
	id       int
	width    int
	height   int
	filepath string
	filesize int
	src      string
	dir      string
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

func (s Image) Dir() string {
	return s.dir
}

func (s Image) Filepath() string {
	return s.filepath
}
