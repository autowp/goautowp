package schema

import (
	geo "github.com/paulmach/go.geo"
)

type NullPoint struct {
	Point geo.Point
	Valid bool // Valid is true if Point is not NULL
}

func (g *NullPoint) Scan(value interface{}) error {
	if value == nil {
		g.Point, g.Valid = geo.Point{}, false

		return nil
	}

	err := g.Point.Scan(value)
	if err != nil {
		return err
	}

	g.Valid = true

	return nil
}
