package yangtree

// type YErrCode int

// const (
// 	YErrorInvalidSchema YErrCode = iota
// 	YErrorNullNode
// )

// var yerrStr = [...]string{
// 	"invalid schema",
// 	"null data node",
// }

// func (s YErrCode) String() string { return yerrStr[s%3] }

// type YError struct {
// 	Code   YErrCode
// 	schema *yang.Entry
// 	data   DataNode
// 	value  string
// }

// func (yerr *YError) Error() string {
// 	if yerr == nil {
// 		return ""
// 	}
// 	return yerr.Code.String()
// }

// func NewError(code YErrCode, data ...interface{}) *YError {
// 	err := &YError{
// 		Code: code,
// 	}
// 	for i := range data {
// 		switch v := data[i].(type) {
// 		case *yang.Entry:
// 			err.schema = v
// 		case DataNode:
// 			err.data = v
// 		case string:
// 			err.value = v
// 		}
// 	}
// 	return err
// }
