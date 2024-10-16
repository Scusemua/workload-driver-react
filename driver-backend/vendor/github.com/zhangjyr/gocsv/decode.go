package gocsv

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"reflect"
)

const (
	ctxKeyHeaderLabels ctxKey = "gocsv.decode.header"
)

var (
	ErrEmptyCSVFile = errors.New("empty csv file given")
	ErrNoStructTags = errors.New("no csv struct tags found")
)

// Decoder .
type Decoder interface {
	getCSVRows() ([][]string, error)
}

// SimpleDecoder .
type SimpleDecoder interface {
	getCSVRow() ([]string, error)
	getCSVRows() ([][]string, error)
}

type CSVReader interface {
	Read() ([]string, error)
	ReadAll() ([][]string, error)
}

type csvDecoder struct {
	CSVReader
}

func newSimpleDecoderFromReader(r io.Reader) SimpleDecoder {
	return &csvDecoder{getCSVReader(r)}
}

// NewSimpleDecoderFromCSVReader creates a SimpleDecoder, which may be passed
// to the UnmarshalDecoder* family of functions, from a CSV reader. Note that
// encoding/csv.Reader implements CSVReader, so you can pass one of those
// directly here.
func NewSimpleDecoderFromCSVReader(r CSVReader) SimpleDecoder {
	return &csvDecoder{r}
}

func (c *csvDecoder) getCSVRows() ([][]string, error) {
	return c.ReadAll()
}

func (c *csvDecoder) getCSVRow() ([]string, error) {
	return c.Read()
}

func mismatchStructFields(structInfo []fieldInfo, headers []string) []string {
	missing := make([]string, 0)
	if len(structInfo) == 0 {
		return missing
	}

	headerMap := make(map[string]struct{}, len(headers))
	for idx := range headers {
		headerMap[headers[idx]] = struct{}{}
	}

	for _, info := range structInfo {
		found := false
		for _, key := range info.keys {
			if _, ok := headerMap[key]; ok {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, info.keys...)
		}
	}
	return missing
}

func mismatchHeaderFields(structInfo []fieldInfo, headers []string) []string {
	missing := make([]string, 0)
	if len(headers) == 0 {
		return missing
	}

	keyMap := make(map[string]struct{})
	for _, info := range structInfo {
		for _, key := range info.keys {
			keyMap[key] = struct{}{}
		}
	}

	for _, header := range headers {
		if _, ok := keyMap[header]; !ok {
			missing = append(missing, header)
		}
	}
	return missing
}

func maybeMissingStructFields(structInfo []fieldInfo, headers []string) error {
	missing := mismatchStructFields(structInfo, headers)
	if len(missing) != 0 {
		return fmt.Errorf("found unmatched struct field with tags %v", missing)
	}
	return nil
}

// Check that no header name is repeated twice
func maybeDoubleHeaderNames(headers []string) error {
	headerMap := make(map[string]bool, len(headers))
	for _, v := range headers {
		if _, ok := headerMap[v]; ok {
			return fmt.Errorf("repeated header name: %v", v)
		}
		headerMap[v] = true
	}
	return nil
}

// apply normalizer func to headers
func normalizeHeaders(headers []string) []string {
	out := make([]string, len(headers))
	for i, h := range headers {
		out[i] = normalizeName(h)
	}
	return out
}

func readTo(decoder SimpleDecoder, out interface{}) error {
	return readToWithErrorHandler(decoder, nil, out)
}

func readToWithErrorHandler(decoder SimpleDecoder, errHandler ErrorHandler, out interface{}) error {
	_, err := readToWithHeaders(context.Background(), decoder, errHandler, out, false, nil)
	return err
}

func readToWithHeaders(ctx context.Context, decoder SimpleDecoder, errHandler ErrorHandler, out interface{}, allowScalar bool, headers []string) (context.Context, error) {
	scalar := false
	outValue, outType := getConcreteReflectValueAndType(out) // Get the concrete type (not pointer) (Slice<?> or Array<?>)
	if err := ensureOutType(outType); err != nil {
		if !allowScalar {
			return ctx, err
		} else {
			scalar = true
			// Reset out value and typle
			outValue = reflect.ValueOf(out)
			outType = outValue.Type()
		}
	}
	outUnitWasPointer, outUnitType := getConcreteContainerUnitType(outType, scalar) // Get the concrete inner type (not pointer) (Container<"?">)
	if err := ensureOutInnerType(outUnitType); err != nil {
		return ctx, err
	} else if scalar && !outUnitWasPointer {
		return ctx, fmt.Errorf("cannot use %v, require *%v", outUnitType, outUnitType)
	}

	csvHeadersLabels, ok := ctx.Value(ctxKeyHeaderLabels).(map[int]*fieldInfo)
	if !ok {
		var err error
		if headers == nil {
			headers, err = decoder.getCSVRow()
			if err != nil {
				return ctx, err
			}
		}

		csvHeadersLabels, err = parseHeader(headers, outUnitType)
		if err != nil {
			return ctx, err
		}

		ctx = context.WithValue(ctx, ctxKeyHeaderLabels, csvHeadersLabels)
	}

	if scalar {
		csvRow, err := decoder.getCSVRow()
		if err != nil {
			return ctx, err
		}
		return ctx, parse(0, csvRow, &outValue, outUnitWasPointer, csvHeadersLabels)
	} else {
		csvRows, err := decoder.getCSVRows() // Get the CSV csvRows
		if err != nil {
			return ctx, err
		}
		if len(csvRows) == 0 {
			return ctx, ErrEmptyCSVFile
		}
		if err := ensureOutCapacity(&outValue, len(csvRows)); err != nil { // Ensure the container is big enough to hold the CSV content
			return ctx, err
		}

		for i, csvRow := range csvRows {
			outInner := createNewOutInner(outUnitWasPointer, outUnitType)
			parseError := parse(i+2, csvRow, &outInner, outUnitWasPointer, csvHeadersLabels)
			if parseError != nil && (errHandler == nil || !errHandler(parseError.(*csv.ParseError))) {
				return ctx, parseError
			}
			outValue.Index(i).Set(outInner)
		}
		return ctx, nil
	}
}

func readEach(decoder SimpleDecoder, c interface{}) error {
	return readEachWithHeaders(context.Background(), decoder, c, nil)
}

func readEachWithHeaders(ctx context.Context, decoder SimpleDecoder, c interface{}, headers []string) error {
	outValue, outType := getConcreteReflectValueAndType(c) // Get the concrete type (not pointer)
	if outType.Kind() != reflect.Chan {
		return fmt.Errorf("cannot use %v with type %s, only channel supported", c, outType)
	}
	defer outValue.Close()

	if headers == nil {
		var err error
		headers, err = decoder.getCSVRow()
		if err != nil {
			return err
		}
	}

	outInnerWasPointer, outInnerType := getConcreteContainerUnitType(outType) // Get the concrete inner type (not pointer) (Container<"?">)
	if err := ensureOutInnerType(outInnerType); err != nil {
		return err
	}

	csvHeadersLabels, err := parseHeader(headers, outInnerType)
	if err != nil {
		return err
	}

	i := 0
	for {
		select {
		case <-ctx.Done():
			return context.Canceled
		default:
		}

		line, err := decoder.getCSVRow()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		outInner := createNewOutInner(outInnerWasPointer, outInnerType)
		parseError := parse(i+2, line, &outInner, outInnerWasPointer, csvHeadersLabels)
		if parseError != nil {
			return parseError
		}
		outValue.Send(outInner)
		i++
	}
	return nil
}

func readEachWithoutHeaders(decoder SimpleDecoder, c interface{}) error {
	outValue, outType := getConcreteReflectValueAndType(c) // Get the concrete type (not pointer) (Slice<?> or Array<?>)
	if err := ensureOutType(outType); err != nil {
		return err
	}
	defer outValue.Close()

	outInnerWasPointer, outInnerType := getConcreteContainerUnitType(outType) // Get the concrete inner type (not pointer) (Container<"?">)
	if err := ensureOutInnerType(outInnerType); err != nil {
		return err
	}
	outInnerStructInfo := getStructInfo(outInnerType) // Get the inner struct info to get CSV annotations
	if len(outInnerStructInfo.Fields) == 0 {
		return ErrNoStructTags
	}

	i := 0
	for {
		line, err := decoder.getCSVRow()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		outInner := createNewOutInner(outInnerWasPointer, outInnerType)
		for j, csvColumnContent := range line {
			fieldInfo := outInnerStructInfo.Fields[j]
			if err := setInnerField(&outInner, outInnerWasPointer, fieldInfo.IndexChain, csvColumnContent, fieldInfo.omitEmpty); err != nil { // Set field of struct
				return &csv.ParseError{
					Line:   i + 2, //add 2 to account for the header & 0-indexing of arrays
					Column: j + 1,
					Err:    err,
				}
			}
		}
		outValue.Send(outInner)
		i++
	}
	return nil
}

func readToWithoutHeaders(decoder Decoder, out interface{}) error {
	outValue, outType := getConcreteReflectValueAndType(out) // Get the concrete type (not pointer) (Slice<?> or Array<?>)
	if err := ensureOutType(outType); err != nil {
		return err
	}
	outInnerWasPointer, outInnerType := getConcreteContainerUnitType(outType) // Get the concrete inner type (not pointer) (Container<"?">)
	if err := ensureOutInnerType(outInnerType); err != nil {
		return err
	}
	csvRows, err := decoder.getCSVRows() // Get the CSV csvRows
	if err != nil {
		return err
	}
	if len(csvRows) == 0 {
		return ErrEmptyCSVFile
	}
	if err := ensureOutCapacity(&outValue, len(csvRows)+1); err != nil { // Ensure the container is big enough to hold the CSV content
		return err
	}
	outInnerStructInfo := getStructInfo(outInnerType) // Get the inner struct info to get CSV annotations
	if len(outInnerStructInfo.Fields) == 0 {
		return ErrNoStructTags
	}

	for i, csvRow := range csvRows {
		outInner := createNewOutInner(outInnerWasPointer, outInnerType)
		for j, csvColumnContent := range csvRow {
			fieldInfo := outInnerStructInfo.Fields[j]
			if err := setInnerField(&outInner, outInnerWasPointer, fieldInfo.IndexChain, csvColumnContent, fieldInfo.omitEmpty); err != nil { // Set field of struct
				return &csv.ParseError{
					Line:   i + 1,
					Column: j + 1,
					Err:    err,
				}
			}
		}
		outValue.Index(i).Set(outInner)
	}

	return nil
}

// Check if the outType is an array or a slice
func ensureOutType(outType reflect.Type) error {
	switch outType.Kind() {
	case reflect.Slice:
		fallthrough
	case reflect.Chan:
		fallthrough
	case reflect.Array:
		return nil
	}
	return fmt.Errorf("cannot use " + outType.String() + ", only slice or array supported")
}

// Check if the outInnerType is of type struct
func ensureOutInnerType(outInnerType reflect.Type) error {
	switch outInnerType.Kind() {
	case reflect.Struct:
		return nil
	}
	return fmt.Errorf("cannot use " + outInnerType.String() + ", only struct supported")
}

func ensureOutCapacity(out *reflect.Value, csvLen int) error {
	switch out.Kind() {
	case reflect.Array:
		if out.Len() < csvLen { // Array is not big enough to hold the CSV content (arrays are not addressable)
			return fmt.Errorf("array capacity problem: cannot store %d %s in %s", csvLen-1, out.Type().Elem().String(), out.Type().String())
		}
	case reflect.Slice:
		if !out.CanAddr() && out.Len() < csvLen { // Slice is not big enough tho hold the CSV content and is not addressable
			return fmt.Errorf("slice capacity problem and is not addressable (did you forget &?)")
		} else if out.CanAddr() && out.Len() < csvLen {
			out.Set(reflect.MakeSlice(out.Type(), csvLen, csvLen)) // Slice is not big enough, so grows it
		}
	}
	return nil
}

func getCSVFieldPosition(key string, structInfo *structInfo, curHeaderCount int) *fieldInfo {
	matchedFieldCount := 0
	for _, field := range structInfo.Fields {
		if field.matchesKey(key) {
			if matchedFieldCount >= curHeaderCount {
				return &field
			}
			matchedFieldCount++
		}
	}
	return nil
}

func createNewOutInner(outInnerWasPointer bool, outInnerType reflect.Type) reflect.Value {
	if outInnerWasPointer {
		return reflect.New(outInnerType)
	}
	return reflect.New(outInnerType).Elem()
}

func parseHeader(headers []string, outInnerType reflect.Type) (map[int]*fieldInfo, error) {
	outInnerStructInfo := getStructInfo(outInnerType) // Get the inner struct info to get CSV annotations
	if len(outInnerStructInfo.Fields) == 0 {
		return nil, ErrNoStructTags
	}

	headers = normalizeHeaders(headers)

	csvHeadersLabels := make(map[int]*fieldInfo, len(outInnerStructInfo.Fields)) // Used to store the correspondance header <-> position in CSV

	headerCount := map[string]int{}
	for i, csvColumnHeader := range headers {
		curHeaderCount := headerCount[csvColumnHeader]
		if fieldInfo := getCSVFieldPosition(csvColumnHeader, outInnerStructInfo, curHeaderCount); fieldInfo != nil {
			csvHeadersLabels[i] = fieldInfo
			if ShouldAlignDuplicateHeadersWithStructFieldOrder {
				curHeaderCount++
				headerCount[csvColumnHeader] = curHeaderCount
			}
		}
	}

	if FailIfUnmatchedStructTags {
		if err := maybeMissingStructFields(outInnerStructInfo.Fields, headers); err != nil {
			return csvHeadersLabels, err
		}
	}
	if FailIfDoubleHeaderNames {
		if err := maybeDoubleHeaderNames(headers); err != nil {
			return csvHeadersLabels, err
		}
	}

	return csvHeadersLabels, nil
}

func parse(lineNo int, csvRow []string, outInner *reflect.Value, outInnerWasPointer bool, csvHeadersLabels map[int]*fieldInfo) error {
	for j, csvColumnContent := range csvRow {
		if fieldInfo, ok := csvHeadersLabels[j]; ok { // Position found accordingly to header name
			value := csvColumnContent
			if value == "" {
				value = fieldInfo.defaultValue
			}
			if err := setInnerField(outInner, outInnerWasPointer, fieldInfo.IndexChain, value, fieldInfo.omitEmpty); err != nil { // Set field of struct
				return &csv.ParseError{
					Line:   lineNo,
					Column: j + 1,
					Err:    err,
				}
			}
		}
	}
	return nil
}

func setInnerField(outInner *reflect.Value, outInnerWasPointer bool, index []int, value string, omitEmpty bool) error {
	oi := *outInner
	if outInnerWasPointer {
		// initialize nil pointer
		if oi.IsNil() {
			setField(oi, "", omitEmpty)
		}
		oi = outInner.Elem()
	}
	// because pointers can be nil need to recurse one index at a time and perform nil check
	if len(index) > 1 {
		nextField := oi.Field(index[0])
		return setInnerField(&nextField, nextField.Kind() == reflect.Ptr, index[1:], value, omitEmpty)
	}
	return setField(oi.FieldByIndex(index), value, omitEmpty)
}
