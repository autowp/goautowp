package sampler

import (
	"errors"
	"fmt"

	"github.com/autowp/goautowp/config"
)

const (
	PNGExtension  = "png"
	JPEGExtension = "jpeg"
	GIFExtension  = "gif"
	WebPExtension = "webp"
	BMPExtension  = "bmp"
	AVIFExtension = "avif"

	StorageFormatGIF  = "gif"
	StorageFormatPNG  = "png"
	StorageFormatJPG  = "jpg"
	StorageFormatJPEG = "jpeg"
	StorageFormatWebP = "webp"
	StorageFormatAVIF = "avif"
	StorageFormatBMP  = "bmp"

	GoFormatGIF  = "GIF"
	GoFormatPNG  = "PNG"
	GoFormatJPG  = "JPG"
	GoFormatJPEG = "JPEG"
	GoFormatWebP = "WEBP"
	GoFormatAVIF = "AVIF"

	ContentTypeImagePNG  = "image/png"
	ContentTypeImageXPNG = "image/x-png"
	ContentTypeImageJPEG = "image/jpeg"
	ContentTypeImageGIF  = "image/gif"
	ContentTypeImageAVIF = "image/avif"
	ContentTypeImageBMP  = "image/bmp"
	ContentTypeImageWebP = "image/webp"
)

var errUnsupportedFormat = errors.New("unsupported format")

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
	StorageFormatJPG:  JPEGExtension,
	StorageFormatJPEG: JPEGExtension,
	StorageFormatPNG:  PNGExtension,
	StorageFormatGIF:  GIFExtension,
	StorageFormatBMP:  BMPExtension,
	StorageFormatWebP: WebPExtension,
	StorageFormatAVIF: AVIFExtension,
}

func (f *Format) FormatExtension() (string, error) {
	if len(f.format) == 0 {
		return "", nil
	}

	value, ok := formatExt[f.format]

	if !ok {
		return "", fmt.Errorf("%w: `%s`", errUnsupportedFormat, f.format)
	}

	return value, nil
}

func (f *Format) IsIgnoreCrop() bool {
	return f.isIgnoreCrop
}

func (f *Format) Widest() float64 {
	return f.widest
}

func (f *Format) Highest() float64 {
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
