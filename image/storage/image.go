package storage

type Image struct {
	id       int
	width    int
	height   int
	filesize int
	src      string
}

func (s Image) GetId() int {
	return s.id
}

func (s Image) GetSrc() string {
	return s.src
}

func (s Image) GetWidth() int {
	return s.width
}

func (s Image) GetHeight() int {
	return s.height
}

func (s Image) GetFileSize() int {
	return s.filesize
}
