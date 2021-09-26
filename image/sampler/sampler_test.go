package sampler

import (
	"github.com/autowp/goautowp/config"
	"github.com/stretchr/testify/require"
	"gopkg.in/gographics/imagick.v2/imagick"
	"testing"
)

func TestShouldResizeOddWidthPictureStrictlyToTargetWidthByOuterFitType(t *testing.T) {
	sampler := NewSampler()
	file := "./_files/Towers_Schiphol_small.jpg"
	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	err := mw.ReadImage(file)
	require.NoError(t, err)
	format := NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeOuter,
		Width:      102,
		Height:     149,
		Background: "red",
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 102)
}

func TestShouldResizeOddHeightPictureStrictlyToTargetHeightByOuterFitType(t *testing.T) {
	sampler := NewSampler()
	file := "./_files/Towers_Schiphol_small.jpg"
	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	err := mw.ReadImage(file)
	require.NoError(t, err)
	format := NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeOuter,
		Width:      101,
		Height:     150,
		Background: "red",
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageHeight(), 150)
}

func TestReduceOnlyWithInnerFitWorks(t *testing.T) {
	sampler := NewSampler()
	file := "./_files/Towers_Schiphol_small.jpg"
	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	// both size less
	err := mw.ReadImage(file)
	require.NoError(t, err)
	format := NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeInner,
		Width:      150,
		Height:     200,
		ReduceOnly: true,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 101)
	require.EqualValues(t, mw.GetImageHeight(), 149)
	mw.Clear()

	// width less
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeInner,
		Width:      150,
		Height:     100,
		ReduceOnly: true,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 68)
	require.EqualValues(t, mw.GetImageHeight(), 100)
	mw.Clear()

	// height less
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeInner,
		Width:      50,
		Height:     200,
		ReduceOnly: true,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 50)
	require.EqualValues(t, mw.GetImageHeight(), 74)
	mw.Clear()

	// not less
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeInner,
		Width:      50,
		Height:     100,
		ReduceOnly: true,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 50)
	require.EqualValues(t, mw.GetImageHeight(), 100)
	mw.Clear()

	// both size less, reduceOnly off
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeInner,
		Width:      150,
		Height:     200,
		ReduceOnly: false,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 150)
	require.EqualValues(t, mw.GetImageHeight(), 200)
	mw.Clear()

	// width less, reduceOnly off
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeInner,
		Width:      150,
		Height:     100,
		ReduceOnly: false,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 150)
	require.EqualValues(t, mw.GetImageHeight(), 100)
	mw.Clear()

	// height less, reduceOnly off
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeInner,
		Width:      50,
		Height:     200,
		ReduceOnly: false,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 50)
	require.EqualValues(t, mw.GetImageHeight(), 200)
	mw.Clear()

	// not less, reduceOnly off
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeInner,
		Width:      50,
		Height:     100,
		ReduceOnly: false,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 50)
	require.EqualValues(t, mw.GetImageHeight(), 100)
	mw.Clear()
}

func TestReduceOnlyWithOuterFitWorks(t *testing.T) {
	sampler := NewSampler()
	file := "./_files/Towers_Schiphol_small.jpg"
	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	// both size less
	err := mw.ReadImage(file)
	require.NoError(t, err)
	format := NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeOuter,
		Width:      150,
		Height:     200,
		ReduceOnly: true,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 150)
	require.EqualValues(t, mw.GetImageHeight(), 200)
	mw.Clear()

	// width less
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeOuter,
		Width:      150,
		Height:     100,
		ReduceOnly: true,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 150)
	require.EqualValues(t, mw.GetImageHeight(), 100)
	mw.Clear()

	// height less
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeOuter,
		Width:      50,
		Height:     200,
		ReduceOnly: true,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 50)
	require.EqualValues(t, mw.GetImageHeight(), 200)
	mw.Clear()

	// not less
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeOuter,
		Width:      50,
		Height:     100,
		ReduceOnly: true,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 50)
	require.EqualValues(t, mw.GetImageHeight(), 100)
	mw.Clear()

	// both size less, reduceOnly off
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeOuter,
		Width:      150,
		Height:     200,
		ReduceOnly: false,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 150)
	require.EqualValues(t, mw.GetImageHeight(), 200)
	mw.Clear()

	// width less, reduceOnly off
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeOuter,
		Width:      150,
		Height:     100,
		ReduceOnly: false,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 150)
	require.EqualValues(t, mw.GetImageHeight(), 100)
	mw.Clear()

	// height less, reduceOnly off
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeOuter,
		Width:      50,
		Height:     200,
		ReduceOnly: false,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 50)
	require.EqualValues(t, mw.GetImageHeight(), 200)
	mw.Clear()

	// not less, reduceOnly off
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeOuter,
		Width:      50,
		Height:     100,
		ReduceOnly: false,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 50)
	require.EqualValues(t, mw.GetImageHeight(), 100)
	mw.Clear()
}

func TestReduceOnlyWithMaximumFitWorks(t *testing.T) {
	sampler := NewSampler()
	file := "./_files/Towers_Schiphol_small.jpg"
	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	// both size less
	err := mw.ReadImage(file)
	require.NoError(t, err)
	format := NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeMaximum,
		Width:      150,
		Height:     200,
		ReduceOnly: true,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 101)
	require.EqualValues(t, mw.GetImageHeight(), 149)
	mw.Clear()

	// width less
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeMaximum,
		Width:      150,
		Height:     100,
		ReduceOnly: true,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 68)
	require.EqualValues(t, mw.GetImageHeight(), 100)
	mw.Clear()

	// height less
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeMaximum,
		Width:      50,
		Height:     200,
		ReduceOnly: true,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 50)
	require.EqualValues(t, mw.GetImageHeight(), 74)
	mw.Clear()

	// not less
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeMaximum,
		Width:      50,
		Height:     100,
		ReduceOnly: true,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 50)
	require.EqualValues(t, mw.GetImageHeight(), 74)
	mw.Clear()

	// both size less, reduceOnly off
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeMaximum,
		Width:      150,
		Height:     200,
		ReduceOnly: false,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 136)
	require.EqualValues(t, mw.GetImageHeight(), 200)
	mw.Clear()

	// width less, reduceOnly off
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeMaximum,
		Width:      150,
		Height:     100,
		ReduceOnly: false,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 68)
	require.EqualValues(t, mw.GetImageHeight(), 100)
	mw.Clear()

	// height less, reduceOnly off
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeMaximum,
		Width:      50,
		Height:     200,
		ReduceOnly: false,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 50)
	require.EqualValues(t, mw.GetImageHeight(), 74)
	mw.Clear()

	// not less, reduceOnly off
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeMaximum,
		Width:      50,
		Height:     100,
		ReduceOnly: false,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 50)
	require.EqualValues(t, mw.GetImageHeight(), 74)
	mw.Clear()
}

func TestReduceOnlyByWidthWorks(t *testing.T) {
	sampler := NewSampler()
	file := "./_files/Towers_Schiphol_small.jpg"
	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	// width less
	err := mw.ReadImage(file)
	require.NoError(t, err)
	format := NewFormat(config.ImageStorageSamplerFormatConfig{
		Width:      150,
		ReduceOnly: true,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 101)
	require.EqualValues(t, mw.GetImageHeight(), 149)
	mw.Clear()

	// not less
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		Width:      50,
		ReduceOnly: true,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 50)
	require.EqualValues(t, mw.GetImageHeight(), 74)
	mw.Clear()

	// width less, reduceOnly off
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		Width:      150,
		ReduceOnly: false,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 150)
	require.EqualValues(t, mw.GetImageHeight(), 221)
	mw.Clear()

	// not less, reduceOnly off
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		Width:      50,
		ReduceOnly: false,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 50)
	require.EqualValues(t, mw.GetImageHeight(), 74)
	mw.Clear()
}

func TestReduceOnlyByHeightWorks(t *testing.T) {
	sampler := NewSampler()
	file := "./_files/Towers_Schiphol_small.jpg"
	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	// height less
	err := mw.ReadImage(file)
	require.NoError(t, err)
	format := NewFormat(config.ImageStorageSamplerFormatConfig{
		Height:     200,
		ReduceOnly: true,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 101)
	require.EqualValues(t, mw.GetImageHeight(), 149)

	mw.Clear()
	// not less
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		Height:     100,
		ReduceOnly: true,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 68)
	require.EqualValues(t, mw.GetImageHeight(), 100)
	mw.Clear()

	// height less, reduceOnly off
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		Height:     200,
		ReduceOnly: false,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 136)
	require.EqualValues(t, mw.GetImageHeight(), 200)
	mw.Clear()

	// not less, reduceOnly off
	err = mw.ReadImage(file)
	require.NoError(t, err)
	format = NewFormat(config.ImageStorageSamplerFormatConfig{
		Height:     100,
		ReduceOnly: false,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 68)
	require.EqualValues(t, mw.GetImageHeight(), 100)
	mw.Clear()
}

func TestAnimationPreservedDueResample(t *testing.T) {
	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	err := mw.ReadImage("./_files/icon-animation.gif")
	require.NoError(t, err)

	sampler := NewSampler()

	format := NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType: config.FitTypeInner,
		Width:   200,
		Height:  200,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)

	require.Less(t, uint(1), mw.GetNumberImages())

	require.EqualValues(t, mw.GetImageWidth(), 200)
	require.EqualValues(t, mw.GetImageHeight(), 200)

	mw.Clear()
}

func TestResizeGif(t *testing.T) {
	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	err := mw.ReadImage("./_files/rudolp-jumping-rope.gif")
	require.NoError(t, err)

	sampler := NewSampler()

	format := NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeInner,
		Width:      80,
		Height:     80,
		Background: "transparent",
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)

	require.Less(t, uint(1), mw.GetNumberImages())

	require.EqualValues(t, mw.GetImageWidth(), 80)
	require.EqualValues(t, mw.GetImageHeight(), 80)

	mw.Clear()
}

func TestResizeGifWithProportionsConstraints(t *testing.T) {
	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	err := mw.ReadImage("./_files/rudolp-jumping-rope.gif")
	require.NoError(t, err)

	sampler := NewSampler()

	format := NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:    config.FitTypeInner,
		Width:      456,
		Background: "",
		Widest:     16.0 / 9.0,
		Highest:    9.0 / 16.0,
		ReduceOnly: true,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)

	require.Less(t, uint(1), mw.GetNumberImages())

	require.EqualValues(t, mw.GetImageWidth(), 456)
	require.EqualValues(t, mw.GetImageHeight(), 342)

	mw.Clear()
}

func TestVerticalProportional(t *testing.T) {
	sampler := NewSampler()

	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	// both size less
	err := mw.ReadImage("./_files/mazda3_sedan_us-spec_11.jpg")
	require.NoError(t, err)
	format := NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:          config.FitTypeInner,
		Width:            200,
		Height:           200,
		ReduceOnly:       true,
		ProportionalCrop: true,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)

	require.EqualValues(t, mw.GetImageWidth(), 200)
	require.EqualValues(t, mw.GetImageHeight(), 200)
	mw.Clear()
}

func TestHorizontalProportional(t *testing.T) {
	sampler := NewSampler()

	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	// both size less
	err := mw.ReadImage("./_files/mazda3_sedan_us-spec_11.jpg")
	require.NoError(t, err)
	format := NewFormat(config.ImageStorageSamplerFormatConfig{
		FitType:          config.FitTypeInner,
		Width:            400,
		Height:           200,
		ReduceOnly:       true,
		ProportionalCrop: true,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)

	require.EqualValues(t, mw.GetImageWidth(), 400)
	require.EqualValues(t, mw.GetImageHeight(), 200)
	mw.Clear()
}

/**
 * @throws Sampler\Exception
 * @throws ImagickException
 */
func TestWidest(t *testing.T) {
	sampler := NewSampler()

	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	// height less
	err := mw.ReadImage("./_files/wide-image.png") //1000x229
	require.NoError(t, err)

	format := NewFormat(config.ImageStorageSamplerFormatConfig{
		Widest: 4.0 / 3.0,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 305)
	require.EqualValues(t, mw.GetImageHeight(), 229)
	mw.Clear()
}

func TestHighest(t *testing.T) {
	sampler := NewSampler()

	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	// height less
	err := mw.ReadImage("./_files/Towers_Schiphol_small.jpg") //101x149
	require.NoError(t, err)

	format := NewFormat(config.ImageStorageSamplerFormatConfig{
		Highest: 1.0,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 101)
	require.EqualValues(t, mw.GetImageHeight(), 101)
	mw.Clear()
}

func TestPngAvatar(t *testing.T) {
	sampler := NewSampler()

	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	// height less
	err := mw.ReadImage("./_files/test.png")
	require.NoError(t, err)
	defer mw.Clear()

	format := NewFormat(config.ImageStorageSamplerFormatConfig{
		Width:      70,
		Height:     70,
		Background: "transparent",
		Strip:      true,
	})
	mw, err = sampler.ConvertImage(mw, Crop{}, *format)
	require.NoError(t, err)
	require.EqualValues(t, mw.GetImageWidth(), 70)
	require.EqualValues(t, mw.GetImageHeight(), 70)
}
