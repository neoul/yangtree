// tozip - generates go byte arrary of the input yang files.
package tozip

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func Zip(file string) ([]byte, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	fbody, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	// Create a buffer to write our archive to.
	buf := new(bytes.Buffer)

	// Create a new zip archive.
	w := gzip.NewWriter(buf)
	w.Write(fbody)

	// Make sure to check the error on Close.
	err = w.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Unzip(gzj []byte) ([]byte, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(gzj))
	if err != nil {
		return nil, err
	}
	defer gzr.Close()
	s, err := ioutil.ReadAll(gzr)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func GenerateGoCodes(in []byte) string {
	var encodedString strings.Builder
	encodedString.WriteString("[]byte{\n")
	for i := 0; i < len(in); i++ {
		encodedString.WriteString(fmt.Sprintf("0x%x", in[i]))
		if i < len(in)-1 {
			encodedString.WriteString(", ")
		}
		if i%20 == 19 {
			encodedString.WriteString("\n")
		}
	}
	encodedString.WriteString("}")
	return encodedString.String()
}

// var metayang = []byte{
// 	0x1f, 0x8b, 0x8, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0x84, 0x56, 0xdb, 0x6e, 0x1b, 0xb1, 0x11, 0x7d, 0xf7, 0x57,
// 	0xc, 0xfc, 0x22, 0x1b, 0x90, 0x56, 0x4e, 0x90, 0xa4, 0xed, 0x26, 0x8, 0xec, 0xf8, 0x86, 0xb6, 0xb2, 0x53, 0xc4, 0x2e,
// 	0xdc, 0x3e, 0x8e, 0xc8, 0x59, 0x2d, 0x63, 0x2e, 0xb9, 0x20, 0x67, 0x25, 0x2b, 0x41, 0xfe, 0xbd, 0x18, 0xee, 0xd5, 0x4e,
// 	0x8c, 0xfa, 0xc9, 0x5c, 0xce, 0x9c, 0xb9, 0x9d, 0x33, 0x54, 0xe5, 0x75, 0x63, 0x9, 0xc, 0x71, 0xb1, 0xd8, 0xa3, 0xdb,
// 	0x2c, 0x2a, 0x62, 0xd4, 0xc8, 0x8, 0x3f, 0xf, 0xe, 0x0, 0x1c, 0x56, 0x14, 0x6b, 0x54, 0x4, 0x87, 0x4d, 0x70, 0xb9,
// 	0x98, 0xe5, 0x35, 0x6, 0xac, 0x62, 0xfe, 0x54, 0xd9, 0xdc, 0xc5, 0x5c, 0x9c, 0xf2, 0xdf, 0xdd, 0xf, 0x3f, 0x8a, 0x7b,
// 	0x1d, 0xa8, 0x30, 0x4f, 0x70, 0x58, 0xe9, 0xf6, 0xec, 0xc3, 0x6, 0x9d, 0xf9, 0x81, 0x6c, 0xbc, 0x3b, 0x0, 0x0, 0x38,
// 	0xfc, 0xfb, 0xe5, 0xfd, 0x15, 0xdc, 0x5e, 0xde, 0xdf, 0x7c, 0xbd, 0x80, 0xa3, 0xdb, 0xcb, 0xfb, 0xf3, 0xaf, 0xb7, 0x57,
// 	0x70, 0x21, 0xf1, 0x6f, 0xbc, 0x26, 0x6b, 0xdc, 0x6, 0x56, 0xe8, 0x36, 0xd, 0x6e, 0xe8, 0x18, 0x1e, 0x7c, 0x78, 0x94,
// 	0x2f, 0xd7, 0xc1, 0x37, 0x75, 0x8b, 0xa8, 0xbc, 0x63, 0x54, 0xdc, 0x82, 0x3d, 0x5c, 0xc3, 0x3, 0xad, 0x73, 0x0, 0xf8,
// 	0x54, 0x32, 0xd7, 0x31, 0x5f, 0x2e, 0x25, 0x17, 0xe, 0xa8, 0x1e, 0x29, 0x64, 0x92, 0x65, 0xe6, 0xc3, 0x66, 0xb9, 0xdb,
// 	0x2c, 0x1d, 0x71, 0xe5, 0xf5, 0xf2, 0xf3, 0x41, 0xf2, 0x84, 0x87, 0x6b, 0x58, 0x99, 0xc8, 0x39, 0xc0, 0xa7, 0xa, 0x8d,
// 	0x65, 0x9f, 0xb7, 0x6, 0xa7, 0xbd, 0xcf, 0xc4, 0xf0, 0xbc, 0x44, 0x13, 0x72, 0x58, 0xf9, 0x6, 0xbe, 0x50, 0xd8, 0x50,
// 	0x68, 0x6f, 0xc6, 0xbf, 0x1e, 0xc2, 0xae, 0xd3, 0xf5, 0xa9, 0xc5, 0xb5, 0xcb, 0x1c, 0xf1, 0xef, 0x18, 0xff, 0x24, 0xc7,
// 	0xf0, 0x80, 0x1c, 0xc9, 0xbd, 0x6, 0xf2, 0xb8, 0x4b, 0xd7, 0xa7, 0xdf, 0x1b, 0x67, 0x6a, 0xa, 0x53, 0x9c, 0x4b, 0x6d,
// 	0xd8, 0x7, 0x29, 0x77, 0x85, 0xda, 0x44, 0x8b, 0x5b, 0x58, 0x95, 0x9e, 0x1f, 0xf1, 0xd5, 0x84, 0xd2, 0xed, 0xa9, 0x33,
// 	0x2a, 0x53, 0x3f, 0x3e, 0xb7, 0xd, 0xd4, 0x14, 0x55, 0x30, 0xf5, 0x38, 0x91, 0xfb, 0xd2, 0x44, 0xf8, 0xef, 0xd9, 0xed,
// 	0x35, 0x74, 0xdc, 0xd0, 0x54, 0x18, 0x47, 0x11, 0xd0, 0xc1, 0x8c, 0x9e, 0x98, 0x5c, 0x34, 0xde, 0xcd, 0x20, 0x32, 0x32,
// 	0x55, 0x52, 0x0, 0x97, 0xc8, 0x80, 0xd6, 0xfa, 0x5d, 0x6c, 0x23, 0x17, 0x3e, 0xb4, 0x5e, 0x32, 0xac, 0x81, 0x50, 0xe8,
// 	0x9c, 0xe7, 0x34, 0xfa, 0x98, 0x75, 0x15, 0x9c, 0xfb, 0x7a, 0x1f, 0xcc, 0xa6, 0x64, 0x38, 0x52, 0xc7, 0xf0, 0xf6, 0xe4,
// 	0xcd, 0x7, 0x48, 0x84, 0xb8, 0xf, 0x4d, 0x64, 0x40, 0xa7, 0x81, 0x4b, 0x82, 0x9a, 0x42, 0xf4, 0x2e, 0x82, 0xd1, 0xe4,
// 	0xd8, 0x14, 0x86, 0x34, 0x60, 0x17, 0x9, 0x1b, 0x2e, 0x7d, 0x88, 0xe0, 0x8b, 0x64, 0xa9, 0xbc, 0xa6, 0xc, 0xe0, 0xcc,
// 	0x5a, 0x48, 0xb0, 0x11, 0x2, 0x45, 0xa, 0x5b, 0xd2, 0x7d, 0xc4, 0x6f, 0xa4, 0x4d, 0xe4, 0x60, 0xd6, 0x8d, 0x24, 0x92,
// 	0x42, 0x34, 0x91, 0xc0, 0x38, 0x88, 0xbe, 0x9, 0x8a, 0xd2, 0x97, 0xb5, 0x71, 0x18, 0xf6, 0x52, 0x46, 0x15, 0xe7, 0xb0,
// 	0x33, 0x5c, 0x82, 0xef, 0xa6, 0x2c, 0x7, 0xdf, 0xb0, 0xf4, 0xc6, 0x14, 0x46, 0xa5, 0x72, 0xe6, 0x60, 0xa2, 0x24, 0x59,
// 	0x19, 0x66, 0xd2, 0x50, 0x37, 0x21, 0x36, 0x28, 0x7d, 0xf1, 0xf3, 0x4, 0x17, 0x9b, 0xf5, 0x77, 0x52, 0x72, 0x6e, 0x31,
// 	0x24, 0x53, 0x6b, 0x14, 0xb9, 0x48, 0xc0, 0x14, 0xaa, 0xd8, 0xb2, 0xd8, 0x38, 0xd2, 0x60, 0xdc, 0x3c, 0xdd, 0xdf, 0x99,
// 	0xaa, 0xb6, 0x6d, 0xad, 0x5f, 0xee, 0x2e, 0x60, 0xd5, 0x99, 0x47, 0xe2, 0xa1, 0xc5, 0x5c, 0x4a, 0xda, 0x77, 0xa4, 0x52,
// 	0x25, 0xef, 0x32, 0xd5, 0x77, 0x61, 0x6c, 0xe1, 0x2c, 0xc2, 0x8a, 0x36, 0x68, 0xe1, 0x5f, 0xc1, 0x6f, 0x8d, 0xcc, 0x2d,
// 	0xf6, 0x6d, 0xb0, 0xc8, 0x32, 0x1d, 0xf6, 0xad, 0xf9, 0x85, 0x57, 0x8d, 0xc, 0xb3, 0xbb, 0x3f, 0x12, 0xfd, 0xe4, 0xcb,
// 	0x25, 0xb, 0xa, 0xd1, 0x28, 0x9d, 0x2e, 0xef, 0x85, 0x71, 0x85, 0x3f, 0xee, 0x9b, 0x9a, 0x28, 0xb3, 0xa5, 0x20, 0x1,
// 	0xda, 0x24, 0x5e, 0x50, 0x48, 0xfa, 0x83, 0x81, 0xe5, 0xee, 0xdb, 0xd5, 0x39, 0xfc, 0xe5, 0x6f, 0xef, 0xdf, 0x3e, 0x8f,
// 	0xb3, 0xdb, 0xed, 0xb2, 0x50, 0xa8, 0x5, 0x25, 0x4a, 0xa7, 0x48, 0x12, 0x61, 0x19, 0xa, 0x25, 0xc6, 0xc7, 0x1f, 0x21,
// 	0x12, 0xa5, 0xe2, 0xc4, 0xdf, 0x70, 0x24, 0x5b, 0x8c, 0x5c, 0x2b, 0x1a, 0x6b, 0xc1, 0xa6, 0x42, 0x9d, 0x67, 0xa3, 0x28,
// 	0x66, 0x2d, 0xb9, 0x3, 0xb5, 0x55, 0x27, 0x6a, 0x2d, 0x4e, 0xfe, 0xba, 0x38, 0x79, 0xf, 0x3f, 0x93, 0xdf, 0x4b, 0xda,
// 	0xcb, 0x2a, 0x72, 0x86, 0xd, 0xda, 0xc1, 0x49, 0x30, 0xe4, 0x22, 0x50, 0x41, 0x81, 0x9c, 0xa2, 0xde, 0xb0, 0x2f, 0x21,
// 	0x87, 0x8b, 0x9e, 0xe4, 0x32, 0xe6, 0x7f, 0x47, 0xf9, 0xef, 0xa6, 0xa7, 0x7b, 0xa2, 0x8d, 0x34, 0x21, 0xc1, 0xfc, 0x92,
// 	0x74, 0x6, 0xf5, 0x4c, 0xb4, 0xd0, 0xe5, 0x83, 0x61, 0x93, 0xfa, 0x9f, 0x36, 0xee, 0xc7, 0xd7, 0x52, 0x4c, 0x8d, 0x9e,
// 	0xa0, 0x24, 0xcd, 0xfd, 0x7f, 0xb9, 0x81, 0x19, 0xd6, 0xcb, 0x64, 0x2a, 0x31, 0x93, 0xc9, 0x11, 0xcc, 0x2a, 0x9d, 0x8f,
// 	0xc6, 0x53, 0x61, 0x2b, 0x74, 0x80, 0x75, 0x4d, 0x18, 0xc0, 0x3b, 0xbb, 0xef, 0x31, 0x90, 0xd3, 0x24, 0xd8, 0xd7, 0x60,
// 	0x69, 0x4b, 0x56, 0xc6, 0x8a, 0xcf, 0xe6, 0xed, 0x83, 0x90, 0xbe, 0x3d, 0xcc, 0xc1, 0x64, 0x94, 0xcd, 0xc1, 0x70, 0xef,
// 	0xbf, 0x26, 0xe5, 0x2b, 0x59, 0x28, 0xe0, 0x68, 0x7, 0x68, 0x99, 0x82, 0x43, 0x36, 0xdb, 0x24, 0x43, 0x41, 0x3e, 0xfb,
// 	0x72, 0x7b, 0x5, 0x75, 0xf0, 0xba, 0x69, 0xb9, 0x1d, 0x4, 0xb3, 0xf0, 0xc3, 0xa2, 0x9d, 0xad, 0xbd, 0xde, 0x2f, 0x22,
// 	0x57, 0x1c, 0x67, 0x70, 0xd4, 0x2b, 0xe0, 0xcd, 0x3b, 0xf1, 0xef, 0x86, 0x73, 0x32, 0x90, 0xb3, 0x2d, 0x72, 0xe8, 0x6f,
// 	0xa7, 0x91, 0xd7, 0x8b, 0xee, 0xd7, 0x9d, 0x58, 0xc9, 0x30, 0x7a, 0x94, 0xce, 0x71, 0x74, 0xca, 0x0, 0xee, 0xf6, 0xe9,
// 	0xf5, 0x31, 0xa, 0xad, 0xdd, 0x4b, 0x89, 0xc2, 0xf4, 0xae, 0x15, 0xc3, 0xb6, 0xa, 0xc3, 0xb6, 0x82, 0xe, 0x5c, 0x4f,
// 	0x85, 0xfb, 0x21, 0x7b, 0x3b, 0xd1, 0xc5, 0xc9, 0x98, 0xf6, 0xd9, 0x33, 0x9a, 0xf4, 0x9e, 0x89, 0x56, 0x49, 0x60, 0x7f,
// 	0x5e, 0xc6, 0xc6, 0x95, 0x14, 0xc, 0xf, 0x11, 0xfb, 0x32, 0xda, 0x57, 0x5c, 0x78, 0xea, 0xb9, 0xa4, 0x90, 0x56, 0xe,
// 	0x3d, 0x31, 0x14, 0xc1, 0x57, 0xc9, 0xe8, 0x99, 0x5e, 0x1d, 0xec, 0x4a, 0xa3, 0xca, 0x1e, 0xa4, 0x2d, 0xac, 0x4b, 0xe1,
// 	0x79, 0x63, 0x13, 0xd7, 0x78, 0x5f, 0xd3, 0xef, 0xd, 0x82, 0x2d, 0xda, 0x26, 0x89, 0x3f, 0xd6, 0xa4, 0xda, 0x65, 0xd6,
// 	0x4d, 0x38, 0x4e, 0x1a, 0xbb, 0xc3, 0x3d, 0x60, 0xcb, 0x62, 0x4, 0x4b, 0x58, 0xb4, 0x98, 0xce, 0x6b, 0x82, 0x26, 0x69,
// 	0x2a, 0xd, 0x4c, 0x62, 0x4c, 0xa, 0x7d, 0x9e, 0x45, 0xa4, 0xa, 0x1d, 0x1b, 0x15, 0xff, 0x90, 0xc5, 0x58, 0xb3, 0xee,
// 	0x96, 0x5c, 0xfb, 0x5d, 0xd8, 0xbd, 0x1e, 0xb2, 0x18, 0x53, 0x1c, 0x63, 0x16, 0x5e, 0x4, 0x26, 0xa7, 0xc8, 0xe8, 0x34,
// 	0x6, 0xdd, 0x76, 0x29, 0x36, 0xeb, 0x21, 0x8f, 0x8, 0x47, 0x68, 0xed, 0x20, 0x8d, 0x40, 0xe0, 0x93, 0x64, 0xd1, 0x1e,
// 	0xe7, 0x30, 0x9b, 0x68, 0x78, 0x36, 0x87, 0x99, 0x29, 0x16, 0x5, 0x21, 0x37, 0x81, 0xe4, 0x34, 0x2c, 0x96, 0xd9, 0x7c,
// 	0x60, 0xb6, 0xe0, 0x36, 0x71, 0xd6, 0x3e, 0x1c, 0xb3, 0xc6, 0x19, 0x8e, 0xb3, 0x9, 0x25, 0x20, 0xbd, 0x66, 0x21, 0x95,
// 	0xd7, 0x38, 0x45, 0x11, 0x62, 0x53, 0xd7, 0x3e, 0x70, 0xd7, 0x3e, 0xd9, 0xb2, 0x46, 0x35, 0x16, 0xc3, 0xb4, 0x3, 0xeb,
// 	0x41, 0xba, 0xc6, 0x29, 0xdb, 0xe8, 0xbe, 0xbe, 0x17, 0xd3, 0x7e, 0xd9, 0xb8, 0x71, 0xe4, 0x80, 0x95, 0x77, 0x9b, 0x29,
// 	0xa5, 0x50, 0x6f, 0x29, 0xb0, 0x89, 0xa4, 0x9f, 0xad, 0x94, 0x39, 0x50, 0xb6, 0x11, 0xb1, 0x3b, 0x40, 0xe8, 0x7f, 0xce,
// 	0x7d, 0x2a, 0xc9, 0x5a, 0xff, 0xb9, 0x77, 0xaf, 0x28, 0x46, 0xdc, 0xa4, 0x25, 0xd1, 0xb1, 0x21, 0x21, 0x58, 0xb3, 0xe,
// 	0xf2, 0xea, 0x1e, 0x8d, 0xa, 0xee, 0xa4, 0x3b, 0x66, 0xa4, 0xd0, 0x4d, 0x92, 0x90, 0xf1, 0x1, 0x32, 0xa3, 0x2a, 0x49,
// 	0xcb, 0x4b, 0x86, 0x6e, 0xf, 0xc6, 0xc9, 0xac, 0x14, 0xb5, 0x4b, 0x69, 0x64, 0xd2, 0x44, 0x79, 0xe8, 0xf6, 0x7f, 0x58,
// 	0x87, 0xed, 0xcf, 0x18, 0xd1, 0xef, 0x58, 0xda, 0x7a, 0xdf, 0xb2, 0x35, 0x35, 0x7d, 0x1c, 0xc3, 0x7f, 0x6e, 0x56, 0x40,
// 	0x4e, 0x79, 0xdd, 0xaf, 0xfe, 0x7f, 0xdc, 0x7d, 0xbd, 0x1d, 0xbf, 0x48, 0xe4, 0xc9, 0xf2, 0x15, 0x4e, 0x8c, 0xd1, 0x7b,
// 	0x8c, 0xfe, 0x11, 0xc9, 0xba, 0xe7, 0xe1, 0xd7, 0xc1, 0xff, 0x2, 0x0, 0x0, 0xff, 0xff, 0x2f, 0x5a, 0x79, 0xe6, 0x8b,
// 	0xb, 0x0, 0x0}
