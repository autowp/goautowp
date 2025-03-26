package goautowp

import "github.com/autowp/goautowp/attrs"

func extractAttrValue(value attrs.Value) AttrValueValue {
	return AttrValueValue{
		Valid:       value.Valid,
		FloatValue:  value.FloatValue,
		IntValue:    value.IntValue,
		BoolValue:   value.BoolValue,
		ListValue:   value.ListValue,
		StringValue: value.StringValue,
		IsEmpty:     value.IsEmpty,
	}
}
