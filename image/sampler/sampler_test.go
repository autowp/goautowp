package sampler

import (
	"testing"

	"github.com/autowp/goautowp/config"
	"github.com/stretchr/testify/require"
	"gopkg.in/gographics/imagick.v2/imagick"
)

const towerFilePath = "./_files/Towers_Schiphol_small.jpg"

func TestShouldResizeOddWidthPictureStrictlyToTargetWidthByOuterFitType(t *testing.T) {
	t.Parallel()

	sampler := NewSampler()
	mw := imagick.NewMagickWand()

	defer mw.Destroy()

	err := mw.ReadImage(towerFilePath)
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
	t.Parallel()

	sampler := NewSampler()
	mw := imagick.NewMagickWand()

	defer mw.Destroy()

	err := mw.ReadImage(towerFilePath)
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

func TestReduceOnlyWorks(t *testing.T) { //nolint:maintidx
	t.Parallel()

	sampler := NewSampler()
	mw := imagick.NewMagickWand()

	defer mw.Destroy()

	tests := []struct {
		formatConfig config.ImageStorageSamplerFormatConfig
		width        int
		height       int
	}{
		// both size less
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				FitType:    config.FitTypeInner,
				Width:      150,
				Height:     200,
				ReduceOnly: true,
			},
			width:  101,
			height: 149,
		},
		// height less
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				FitType:    config.FitTypeInner,
				Width:      50,
				Height:     200,
				ReduceOnly: true,
			},
			width:  50,
			height: 74,
		},
		// not less
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				FitType:    config.FitTypeInner,
				Width:      50,
				Height:     100,
				ReduceOnly: true,
			},
			width:  50,
			height: 100,
		},
		// both size less, reduceOnly off
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				FitType:    config.FitTypeInner,
				Width:      150,
				Height:     200,
				ReduceOnly: false,
			},
			width:  150,
			height: 200,
		},
		// width less, reduceOnly off
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				FitType:    config.FitTypeInner,
				Width:      150,
				Height:     100,
				ReduceOnly: false,
			},
			width:  150,
			height: 100,
		},
		// height less, reduceOnly off
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				FitType:    config.FitTypeInner,
				Width:      50,
				Height:     200,
				ReduceOnly: false,
			},
			width:  50,
			height: 200,
		},
		// not less, reduceOnly off
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				FitType:    config.FitTypeInner,
				Width:      50,
				Height:     100,
				ReduceOnly: false,
			},
			width:  50,
			height: 100,
		},
		// FitTypeOuter
		// both size less
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				FitType:    config.FitTypeOuter,
				Width:      150,
				Height:     200,
				ReduceOnly: true,
			},
			width:  150,
			height: 200,
		},
		// width less
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				FitType:    config.FitTypeOuter,
				Width:      150,
				Height:     100,
				ReduceOnly: true,
			},
			width:  150,
			height: 100,
		},
		// height less
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				FitType:    config.FitTypeOuter,
				Width:      50,
				Height:     200,
				ReduceOnly: true,
			},
			width:  50,
			height: 200,
		},
		// not less
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				FitType:    config.FitTypeOuter,
				Width:      50,
				Height:     100,
				ReduceOnly: true,
			},
			width:  50,
			height: 100,
		},
		// both size less, reduceOnly off
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				FitType:    config.FitTypeOuter,
				Width:      150,
				Height:     200,
				ReduceOnly: false,
			},
			width:  150,
			height: 200,
		},
		// width less, reduceOnly off
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				FitType:    config.FitTypeOuter,
				Width:      150,
				Height:     100,
				ReduceOnly: false,
			},
			width:  150,
			height: 100,
		},
		// height less, reduceOnly off
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				FitType:    config.FitTypeOuter,
				Width:      50,
				Height:     200,
				ReduceOnly: false,
			},
			width:  50,
			height: 200,
		},
		// not less, reduceOnly off
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				FitType:    config.FitTypeOuter,
				Width:      50,
				Height:     100,
				ReduceOnly: false,
			},
			width:  50,
			height: 100,
		},
		// ReduceOnlyWithMaximumFit
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				FitType:    config.FitTypeMaximum,
				Width:      150,
				Height:     200,
				ReduceOnly: true,
			},
			width:  101,
			height: 149,
		},
		// width less
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				FitType:    config.FitTypeMaximum,
				Width:      150,
				Height:     100,
				ReduceOnly: true,
			},
			width:  68,
			height: 100,
		},
		// height less
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				FitType:    config.FitTypeMaximum,
				Width:      50,
				Height:     200,
				ReduceOnly: true,
			},
			width:  50,
			height: 74,
		},
		// not less
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				FitType:    config.FitTypeMaximum,
				Width:      50,
				Height:     100,
				ReduceOnly: true,
			},
			width:  50,
			height: 74,
		},
		// both size less, reduceOnly off
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				FitType:    config.FitTypeMaximum,
				Width:      150,
				Height:     200,
				ReduceOnly: false,
			},
			width:  136,
			height: 200,
		},
		// width less, reduceOnly off
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				FitType:    config.FitTypeMaximum,
				Width:      150,
				Height:     100,
				ReduceOnly: false,
			},
			width:  68,
			height: 100,
		},
		// height less, reduceOnly off
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				FitType:    config.FitTypeMaximum,
				Width:      50,
				Height:     200,
				ReduceOnly: false,
			},
			width:  50,
			height: 74,
		},
		// not less, reduceOnly off
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				FitType:    config.FitTypeMaximum,
				Width:      50,
				Height:     100,
				ReduceOnly: false,
			},
			width:  50,
			height: 74,
		},
		// ReduceOnlyByWidth
		// width less
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				Width:      150,
				ReduceOnly: true,
			},
			width:  101,
			height: 149,
		},
		// not less
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				Width:      50,
				ReduceOnly: true,
			},
			width:  50,
			height: 74,
		},
		// width less, reduceOnly off
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				Width:      150,
				ReduceOnly: false,
			},
			width:  150,
			height: 221,
		},
		// not less, reduceOnly off
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				Width:      50,
				ReduceOnly: false,
			},
			width:  50,
			height: 74,
		},
		// ReduceOnlyByHeight
		// height less
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				Height:     200,
				ReduceOnly: true,
			},
			width:  101,
			height: 149,
		},
		// not less
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				Height:     100,
				ReduceOnly: true,
			},
			width:  68,
			height: 100,
		},
		// height less, reduceOnly off
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				Height:     200,
				ReduceOnly: false,
			},
			width:  136,
			height: 200,
		},
		// not less, reduceOnly off
		{
			formatConfig: config.ImageStorageSamplerFormatConfig{
				Height:     100,
				ReduceOnly: false,
			},
			width:  68,
			height: 100,
		},
	}

	for _, tt := range tests {
		err := mw.ReadImage(towerFilePath)
		require.NoError(t, err)

		format := NewFormat(tt.formatConfig)
		mw, err = sampler.ConvertImage(mw, Crop{}, *format)
		require.NoError(t, err)
		require.EqualValues(t, mw.GetImageWidth(), tt.width)
		require.EqualValues(t, mw.GetImageHeight(), tt.height)
		mw.Clear()
	}
}

func TestAnimationPreservedDueResample(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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

func TestWidest(t *testing.T) {
	t.Parallel()

	sampler := NewSampler()

	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	// height less
	err := mw.ReadImage("./_files/wide-image.png") // 1000x229
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
	t.Parallel()

	sampler := NewSampler()

	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	// height less
	err := mw.ReadImage(towerFilePath) // 101x149
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
	t.Parallel()

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
