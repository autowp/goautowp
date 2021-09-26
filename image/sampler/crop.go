package sampler

type Crop struct {
	Left   int
	Top    int
	Width  int
	Height int
}

func (c Crop) IsEmpty() bool {
	return c.Width <= 0 || c.Height <= 0
}
