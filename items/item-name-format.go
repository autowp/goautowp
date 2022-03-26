package items

import (
	"fmt"
	"strconv"
	"time"
)

const textMonthFormat = "%02d."

type ItemNameFormatter struct{}

type ItemNameFormatterOptions struct {
	BeginModelYear         int
	EndModelYear           int
	BeginModelYearFraction string
	EndModelYearFraction   string
	Spec                   string
	SpecFull               string
	Body                   string
	Name                   string
	BeginYear              int
	EndYear                int
	Today                  *bool
	BeginMonth             int
	EndMonth               int
}

func (s *ItemNameFormatter) Format(item ItemNameFormatterOptions, language string) string {
	result := item.Name

	if len(item.Spec) > 0 {
		result += " [" + item.Spec + "]"
	}

	if len(item.Body) > 0 {
		result += " (" + item.Body + ")"
	}

	by := item.BeginYear
	bm := item.BeginMonth
	ey := item.EndYear
	em := item.EndMonth

	bmy := item.BeginModelYear
	emy := item.EndModelYear

	bmyf := item.BeginModelYearFraction
	emyf := item.EndModelYearFraction

	bs := by / 100
	es := ey / 100

	useModelYear := bmy > 0 || emy > 0

	equalS := bs > 0 && es > 0 && (bs == es)
	equalY := equalS && by > 0 && ey > 0 && (by == ey)
	equalM := equalY && bm > 0 && em > 0 && (bm == em)

	if useModelYear {
		result = s.getModelYearsPrefix(bmy, bmyf, emy, emyf, item.Today, language) + " " + result
	}

	if by > 0 || ey > 0 {
		result += " '" + s.renderYears(
			item.Today,
			by,
			bm,
			ey,
			em,
			equalS,
			equalY,
			equalM,
			language,
		)
	}

	return result
}

func (s *ItemNameFormatter) getModelYearsPrefix(
	begin int,
	beginFraction string,
	end int,
	endFraction string,
	today *bool,
	language string,
) string {
	if end == begin && beginFraction == endFraction {
		return fmt.Sprintf("%d%s", begin, endFraction)
	}

	bms := begin / 100
	ems := end / 100

	if bms == ems {
		return fmt.Sprintf("%d%s–%02d%s", begin, beginFraction, end%100, endFraction)
	}

	if begin <= 0 {
		return fmt.Sprintf("????–%02d%s", end, endFraction)
	}

	if end > 0 {
		return fmt.Sprintf("%d%s–%d%s", begin, beginFraction, end, endFraction)
	}

	if today == nil || !*today {
		return fmt.Sprintf("%d%s–??", begin, beginFraction)
	}

	currentYear := time.Now().Year()

	if begin >= currentYear {
		return fmt.Sprintf("%d%s", begin, beginFraction)
	}

	return fmt.Sprintf("%d%s–%s", begin, beginFraction, "pr.")
	// $this->translate('present-time-abbr', $language);
}

func (s *ItemNameFormatter) renderYears(
	today *bool,
	by int,
	bm int,
	ey int,
	em int,
	equalS bool,
	equalY bool,
	equalM bool,
	language string,
) string {
	if equalM {
		return fmt.Sprintf(textMonthFormat+"%d", bm, by)
	}

	if equalY {
		if bm > 0 && em > 0 {
			return s.monthsRange(bm, em) + "." + strconv.Itoa(by)
		}

		return strconv.Itoa(by)
	}

	if equalS {
		result1 := ""
		if bm > 0 {
			result1 = fmt.Sprintf(textMonthFormat, bm)
		}
		result1 += strconv.Itoa(by) + "–"

		result2 := ""
		if em > 0 {
			result2 = fmt.Sprintf(textMonthFormat, em)
		}

		result3 := ""
		if em > 0 {
			result3 = strconv.Itoa(ey)
		} else {
			result3 = fmt.Sprintf("%02d", ey%100)
		}

		return result1 + result2 + result3
	}

	result1 := ""
	if bm > 0 {
		result1 = fmt.Sprintf(textMonthFormat, bm)
	}
	result2 := "????"
	if by > 0 {
		result2 = strconv.Itoa(by)
	}

	result3 := ""
	if ey > 0 {
		if em > 0 {
			result3 = fmt.Sprintf(textMonthFormat, em)
		}
		result3 = "–" + result3 + strconv.Itoa(ey)
	} else {
		result3 = s.missedEndYearYearsSuffix(today, by, language)
	}

	return result1 + result2 + result3
}

func (s *ItemNameFormatter) missedEndYearYearsSuffix(today *bool, by int, language string) string {
	currentYear := time.Now().Year()

	if by >= currentYear {
		return ""
	}

	if today != nil && *today {
		// s.translate('present-time-abbr', $language)
		return "–pr."
	}

	return "–????"

}

func (s *ItemNameFormatter) monthsRange(from int, to int) string {
	result1 := "??"
	if from > 0 {
		result1 = fmt.Sprintf("%02d", from)
	}

	result2 := "??"
	if to > 0 {
		result2 = fmt.Sprintf("%02d", to)
	}

	return result1 + "–" + result2
}
