package sampler

import "fmt"

type FitType int

const (
	FitTypeInner   FitType = 0
	FitTypeOuter   FitType = 1
	FitTypeMaximum FitType = 2
)

type FormatConfig struct {
	FitType          FitType `mapstructure:"fit-type"`
	Width            int     `mapstructure:"width"`
	Height           int     `mapstructure:"height"`
	Background       string  `mapstructure:"background"`
	Strip            bool    `mapstructure:"strip"`
	ReduceOnly       bool    `mapstructure:"reduce-only"`
	ProportionalCrop bool    `mapstructure:"proportional-crop"`
	Format           string  `mapstructure:"format"`
	IgnoreCrop       bool    `mapstructure:"ignore-crop"`
	Widest           float64 `mapstructure:"widest"`
	Highest          float64 `mapstructure:"highest"`
}

type Format struct {
	format             string
	isIgnoreCrop       bool
	width              int
	height             int
	widest             float64
	highest            float64
	isProportionalCrop bool
	background         string
	fitType            FitType
	isStrip            bool
	isReduceOnly       bool
}

func NewFormat(config FormatConfig) *Format {
	return &Format{
		format:             config.Format,
		isIgnoreCrop:       config.IgnoreCrop,
		width:              config.Width,
		height:             config.Height,
		isProportionalCrop: config.ProportionalCrop,
		background:         config.Background,
		fitType:            config.FitType,
		isStrip:            config.Strip,
		isReduceOnly:       config.ReduceOnly,
		widest:             config.Widest,
		highest:            config.Highest,
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

func (f Format) GetWidest() float64 {
	return f.widest
}

func (f Format) GetHighest() float64 {
	return f.highest
}

func (f *Format) IsProportionalCrop() bool {
	return f.isProportionalCrop
}

func (f *Format) GetBackground() string {
	return f.background
}

func (f *Format) FitType() FitType {
	return f.fitType
}

func (f *Format) IsReduceOnly() bool {
	return f.isReduceOnly
}
