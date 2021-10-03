package sampler

import (
	"fmt"
	"github.com/autowp/goautowp/config"
)

type Format struct {
	format             string
	isIgnoreCrop       bool
	width              int
	height             int
	widest             float64
	highest            float64
	isProportionalCrop bool
	background         string
	fitType            config.FitType
	isStrip            bool
	isReduceOnly       bool
}

func NewFormat(cfg config.ImageStorageSamplerFormatConfig) *Format {
	return &Format{
		format:             cfg.Format,
		isIgnoreCrop:       cfg.IgnoreCrop,
		width:              cfg.Width,
		height:             cfg.Height,
		isProportionalCrop: cfg.ProportionalCrop,
		background:         cfg.Background,
		fitType:            cfg.FitType,
		isStrip:            cfg.Strip,
		isReduceOnly:       cfg.ReduceOnly,
		widest:             cfg.Widest,
		highest:            cfg.Highest,
	}
}

func (f *Format) Format() string {
	return f.format
}

func (f *Format) IsStrip() bool {
	return f.isStrip
}

func (f *Format) Height() int {
	return f.height
}

func (f *Format) Width() int {
	return f.width
}

var formatExt = map[string]string{
	"jpg":  "jpeg",
	"jpeg": "jpeg",
	"png":  "png",
	"gif":  "gif",
	"bmp":  "bmp",
}

func (f Format) FormatExtension() (string, error) {
	if len(f.format) <= 0 {
		return "", nil
	}

	value, ok := formatExt[f.format]

	if !ok {
		return "", fmt.Errorf("unsupported format `%s`", f.format)
	}

	return value, nil
}

func (f Format) IsIgnoreCrop() bool {
	return f.isIgnoreCrop
}

func (f Format) Widest() float64 {
	return f.widest
}

func (f Format) Highest() float64 {
	return f.highest
}

func (f *Format) IsProportionalCrop() bool {
	return f.isProportionalCrop
}

func (f *Format) Background() string {
	return f.background
}

func (f *Format) FitType() config.FitType {
	return f.fitType
}

func (f *Format) IsReduceOnly() bool {
	return f.isReduceOnly
}
