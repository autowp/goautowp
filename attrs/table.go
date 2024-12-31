package attrs

import (
	"bytes"
	"embed"
	"html/template"

	"github.com/autowp/goautowp/util"
)

//go:embed *.tmpl
var TemplatesFS embed.FS

var hideAttrs = []int64{
	frontWheelTyreWidthAttr, frontWheelTyreSeriesAttr, frontWheelRadiusAttr, frontWheelRimWidthAttr,
	rearWheelTyreWidthAttr, rearWheelTyreSeriesAttr, rearWheelRadiusAttr, rearWheelRimWidthAttr,
	EnginePlacementPlacementAttr, EnginePlacementOrientationAttr,
	bootVolumeMinAttr, bootVolumeMaxAttr,
	fuelTankPrimaryAttr, fuelTankSecondaryAttr,
	EngineConfigurationCylindersCountAttr, EngineConfigurationCylindersLayoutAttr, EngineConfigurationValvesCountAttr,
	gearboxTypeAttr, gearboxGearsAttr, gearboxNameAttr,
}

var renderMap = map[int64]Renderer{
	frontWheelAttr: Wheel{
		TyreWidth:  frontWheelTyreWidthAttr,
		TyreSeries: frontWheelTyreSeriesAttr,
		Radius:     frontWheelRadiusAttr,
		Rimwidth:   frontWheelRimWidthAttr,
	},
	rearWheelAttr: Wheel{
		TyreWidth:  rearWheelTyreWidthAttr,
		TyreSeries: rearWheelTyreSeriesAttr,
		Radius:     rearWheelRadiusAttr,
		Rimwidth:   rearWheelRimWidthAttr,
	},
	enginePlacementAttr: EnginePlacement{
		Placement:   EnginePlacementPlacementAttr,
		Orientation: EnginePlacementOrientationAttr,
	},
	bootVolumeAttr: BootVolume{
		Min: bootVolumeMinAttr,
		Max: bootVolumeMaxAttr,
	},
	fuelTankAttr: FuelTank{
		Primary:   fuelTankPrimaryAttr,
		Secondary: fuelTankSecondaryAttr,
	},
	engineConfigurationAttr: EngineConfiguration{
		CylindersCount:  EngineConfigurationCylindersCountAttr,
		CylindersLayout: EngineConfigurationCylindersLayoutAttr,
		ValvesCount:     EngineConfigurationValvesCountAttr,
	},
	gearboxAttr: Gearbox{
		Type:  gearboxTypeAttr,
		Gears: gearboxGearsAttr,
		Name:  gearboxNameAttr,
	},
}

type CarSpecTable struct {
	Items      []CarSpecTableItem
	Attributes []*AttributeRow
	Units      map[int64]I18nUnit
}

type CarSpecTableItemImage struct {
	Src    string
	Width  int
	Height int
}

type CarSpecTableItem struct {
	ID                 int64
	NameHTML           template.HTML
	YearsHTML          template.HTML
	TopPictureURL      string
	TopPictureImage    *CarSpecTableItemImage
	BottomPictureURL   string
	BottomPictureImage *CarSpecTableItemImage
	Values             map[int64]string
}

type Cell struct {
	ItemID  int64
	Value   template.HTML
	Colspan uint
}

type TemplateData struct {
	Items           []CarSpecTableItem
	Attrs           []TemplateAttr
	ItemsLenPlusOne int
}

type TemplateAttr struct {
	Name      string
	Padding   int
	Cells     []Cell
	HasChilds bool
	HasValues bool
}

func (s *CarSpecTable) Cells(attribute *AttributeRow) ([]Cell, bool) {
	cells := make([]Cell, 0)
	hasValues := false

	for _, item := range s.Items {
		value := s.renderValue(attribute, item.Values)
		isSame := false

		if value != "" {
			hasValues = true
		}

		lastColIdx := len(cells) - 1
		if lastColIdx >= 0 {
			lastCol := cells[lastColIdx]
			isSame = lastCol.Value == value
		}

		if isSame {
			cells[lastColIdx].Colspan++
		} else {
			cells = append(cells, Cell{
				ItemID:  item.ID,
				Value:   value,
				Colspan: 1,
			})
		}
	}

	return cells, hasValues
}

func (s *CarSpecTable) Render() (string, error) {
	tmpl, err := template.New("specs.tmpl").ParseFS(TemplatesFS, "specs.tmpl")
	if err != nil {
		return "", err
	}

	templateAttrs := make([]TemplateAttr, 0)

	for _, attribute := range s.Attributes {
		if !util.Contains(hideAttrs, attribute.ID) {
			cells, hasValues := s.Cells(attribute)
			templateAttrs = append(templateAttrs, TemplateAttr{
				Name:      attribute.NameTranslated,
				Cells:     cells,
				Padding:   5 + attribute.Deep*16, //nolint: mnd
				HasChilds: len(attribute.Childs) > 0,
				HasValues: hasValues,
			})
		}
	}

	buf := new(bytes.Buffer)

	err = tmpl.Execute(buf, TemplateData{
		Items:           s.Items,
		Attrs:           templateAttrs,
		ItemsLenPlusOne: len(s.Items) + 1,
	})
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (s *CarSpecTable) renderValue(attribute *AttributeRow, values map[int64]string) template.HTML {
	renderer, ok := renderMap[attribute.ID]
	if !ok {
		renderer = DefaultValue{}
	}

	return renderer.Render(attribute, values, s.Units) // , itemTypeId, itemId
}
