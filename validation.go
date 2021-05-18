package yangtree

import (
	"fmt"
	"reflect"
	"regexp"
	"unicode/utf8"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
)

func Validate(node DataNode) []error {
	checkAll := true
	for n := node; n != nil; n = n.Parent() {
		err := validateDataNode(node, node.Schema().Type, checkAll)
		if err != nil {
			return err
		}
		checkAll = false
	}
	return nil
}

func validateDataNode(node DataNode, typ *yang.YangType, checkAll bool) []error {
	var errors []error
	// when, must statements must be test for the validation.
	whenstr, ok := node.Schema().GetWhenXPath()
	if ok {
		condition, err := evaluatePathExpr(node, whenstr)
		if err != nil {
			errors = append(errors, err)
		} else if !condition {
			errors = append(errors, fmt.Errorf("when %q condition failed", whenstr))
		}
	}
	mustlist := GetMust(node.Schema())
	for i := range mustlist {
		mustXPath, ok := mustlist[i].Source.Arg()
		if ok {
			condition, err := evaluatePathExpr(node, mustXPath)
			if err != nil {
				if mustlist[i].ErrorMessage.Name != "" {
					errors = append(errors, fmt.Errorf(mustlist[i].ErrorMessage.Name))
				} else {
					errors = append(errors, err)
				}
			} else if !condition {
				errors = append(errors, fmt.Errorf("must %q condition failed", whenstr))
			}
		}
	}
	// if len(mustlist) > 0 {
	// 	fmt.Println(mustlist[0].Source.Arg())
	// }
	switch n := node.(type) {
	case *DataBranch:
		// check the validation of the children
		if checkAll {
			for i := range n.children {
				err := validateDataNode(n.children[i], n.children[i].Schema().Type, checkAll)
				errors = append(errors, err...)
			}
		}
		return errors
	default:
		switch typ.Kind {
		// case yang.Ystring, yang.Ybinary:
		// case yang.Ybool:
		// case yang.Yempty:
		// case yang.Yint8, yang.Yint16, yang.Yint32, yang.Yint64, yang.Yuint8, yang.Yuint16, yang.Yuint32, yang.Yuint64:
		// case yang.Ybits, yang.Yenum:
		// case yang.Yidentityref:
		// case yang.Ydecimal64:
		// case yang.Yunion:
		// case yang.Ynone:
		case yang.YinstanceIdentifier:
			ref, err := node.Find(node.ValueString())
			if err != nil {
				return append(errors, err)
			}
			if len(ref) == 0 {
				return append(errors, fmt.Errorf("data instance not present to %q", node.Path()))
			}
		case yang.Yleafref:
			ref, err := node.Find(typ.Path)
			if err != nil {
				return append(errors, err)
			}
			nodeValue := node.ValueString()
			for i := range ref {
				if ref[i].ValueString() == nodeValue {
					return nil
				}
			}
			return append(errors, fmt.Errorf("invalid leafref '%v'", nodeValue))
		default:
		}
	}
	return errors
}

// Refer to:
// https://tools.ietf.org/html/rfc6020#section-9.4.
// github.com/openconfig/ygot/ytypes/string_type.go

func ValidateSchema(schema *yang.Entry) error {
	return nil
}

// validateString validates value, which must be a Go string type, against the
// given schema.
func validateString(schema *yang.Entry, value interface{}) error {
	// Check that the schema itself is valid.
	if err := validateStringSchema(schema); err != nil {
		return err
	}

	vv := reflect.ValueOf(value)

	// Check that type of value is the type expected from the schema.
	if vv.Kind() != reflect.String {
		return fmt.Errorf("non string type %T with value %v for schema %s", value, value, schema.Name)
	}

	// This value could be a union typedef string, so convert it to make
	// sure it's the primitive string type.
	stringVal := vv.Convert(reflect.TypeOf("")).Interface().(string)

	// Check that the length is within the allowed range.
	allowedRanges := schema.Type.Length
	strLen := uint64(utf8.RuneCountInString(stringVal))
	if !lengthOk(allowedRanges, strLen) {
		return fmt.Errorf("length %d is outside range %v for schema %s", strLen, allowedRanges, schema.Name)
	}

	// Check that the value satisfies any regex patterns.
	patterns, isPOSIX := util.SanitizedPattern(schema.Type)
	for _, p := range patterns {
		var r *regexp.Regexp
		var err error
		if isPOSIX {
			r, err = regexp.CompilePOSIX(p)
		} else {
			r, err = regexp.Compile(p)
		}
		if err != nil {
			return err
		}
		if !r.MatchString(stringVal) {
			return fmt.Errorf("%q does not match regular expression pattern %q for schema %s", stringVal, r, schema.Name)
		}
	}

	return nil
}

// validateStringSlice validates value, which must be a Go string slice type,
// against the given schema.
func validateStringSlice(schema *yang.Entry, value interface{}) error {
	// Check that the schema itself is valid.
	if err := validateStringSchema(schema); err != nil {
		return err
	}

	// Check that type of value is the type expected from the schema.
	slice, ok := value.([]string)
	if !ok {
		return fmt.Errorf("non []string type %T with value %v for schema %s", value, value, schema.Name)
	}

	// Each slice element must be valid and unique.
	tbl := make(map[string]bool, len(slice))
	for i, val := range slice {
		if err := validateString(schema, val); err != nil {
			return fmt.Errorf("invalid element at index %d: %v for schema %s", i, err, schema.Name)
		}
		if tbl[val] {
			return fmt.Errorf("duplicate string: %q for schema %s", val, schema.Name)
		}
		tbl[val] = true
	}
	return nil
}

// validateStringSchema validates the given string type schema. This is a sanity
// check validation rather than a comprehensive validation against the RFC.
// It is assumed that such a validation is done when the schema is parsed from
// source YANG.
func validateStringSchema(schema *yang.Entry) error {
	if schema == nil {
		return fmt.Errorf("string schema is nil")
	}
	if schema.Type == nil {
		return fmt.Errorf("string schema %s Type is nil", schema.Name)
	}
	if schema.Type.Kind != yang.Ystring {
		return fmt.Errorf("string schema %s has wrong type %v", schema.Name, schema.Type.Kind)
	}

	patterns, isPOSIX := util.SanitizedPattern(schema.Type)
	for _, p := range patterns {
		var err error
		if isPOSIX {
			_, err = regexp.CompilePOSIX(p)
		} else {
			_, err = regexp.Compile(p)
		}
		if err != nil {
			return fmt.Errorf("error generating regexp %s %v for schema %s", p, err, schema.Name)
		}
	}

	return validateLengthSchema(schema)
}

//lint:file-ignore U1000 Ignore all unused code, it represents generated code.

// validateLengthSchema validates whether the given schema has a valid length
// specification.
func validateLengthSchema(schema *yang.Entry) error {
	if len(schema.Type.Length) == 0 {
		return nil
	}
	for _, r := range schema.Type.Length {
		// This is a limited sanity check. It's assumed that a full check is
		// done in the goyang parser.
		minLen, maxLen := r.Min, r.Max
		if minLen.Kind != yang.MinNumber && minLen.Kind != yang.Positive {
			return fmt.Errorf("length Min must be Positive or MinNumber: %v for schema %s", minLen, schema.Name)
		}
		if maxLen.Kind != yang.MaxNumber && maxLen.Kind != yang.Positive {
			return fmt.Errorf("length Max must be Positive or MaxNumber: %v for schema %s", minLen, schema.Name)
		}
		if maxLen.Less(minLen) {
			return fmt.Errorf("schema has bad length min[%v] > max[%v] for schema %s", minLen, maxLen, schema.Name)
		}
	}

	return nil
}

// lengthOk reports whether the given value of length falls within the ranges
// allowed by yrs. Always returns true is yrs is empty.
func lengthOk(yrs yang.YangRange, val uint64) bool {
	return isInRanges(yrs, yang.FromUint(val))
}

// isInRanges reports whether the given value falls within the ranges allowed by
// yrs. Always returns true is yrs is empty.
func isInRanges(yrs yang.YangRange, val yang.Number) bool {
	if len(yrs) == 0 {
		return true
	}
	for _, yr := range yrs {
		if isInRange(yr, val) {
			return true
		}
	}
	return false
}

// isInRange reports whether the given value falls within the range allowed by
// yr.
func isInRange(yr yang.YRange, val yang.Number) bool {
	return (val.Less(yr.Max) || val.Equal(yr.Max)) &&
		(yr.Min.Less(val) || yr.Min.Equal(val))
}
