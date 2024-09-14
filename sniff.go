package mxftest

import "context"

// CType is used for declaring content types of a data type.
// These are a separate type to make them easy to identify with autocomplete etc.
type CType string

const (
	// ContentTypeKey is the key used for storing the ContentType of data from a sniff test.
	// It can be used with the node search functions
	ContentTypeKey = "ContentType"
)

// DataIdentifier contains the data type
// and function for checking if the data stream is that
// type.
type DataIdentifier struct {
	DataFunc    func([]byte) bool
	ContentType CType
}

// Sniff checks a stream of bytes to find the data type.
// it then performs any further sniff tests based on the type of data it finds,
// if it finds any data types that match.
// It stops searching after finding a data type that matches.
func Sniff(data []byte, sniffers map[*DataIdentifier][]Sniffer) map[string]*SniffResult {

	// do the datatype test first
	// if its not that data type bin it off
	sniffRes := make(map[string]*SniffResult)
	for dataType, sniffs := range sniffers {

		dt := *dataType
		if !dt.DataFunc(data) {
			// check the next datatype
			continue
		}

		sniffRes[ContentTypeKey] = &SniffResult{Key: ContentTypeKey, Field: string(dt.ContentType)}

		for _, sniff := range sniffs {
			snif := *sniff
			res := snif(data)
			// if there's something more than 0% certainty add it
			if res.Certainty > 0 {
				// do some stuff here
				// @TODO implement stopping things from being overwritten
				res.Data = dt.ContentType
				sniffRes[res.Key] = &res
			}
		}
		// return after the valid datatype was found?
		return sniffRes
	}

	return sniffRes
}

// SniffTest contains an identifier
// and the tests to run on any data that is identified.
type SniffTest struct {
	DataID DataIdentifier
	Sniffs []Sniffer
}

// SniffResult is the result of the sniff test.
type SniffResult struct {
	// The sniff test key
	Key string
	// The sniff test result field
	Field string
	// The data of the sniff test
	Data CType
	// what certainty did the sniff test past as a %
	Certainty float64
}

// Sniffer takes a stream of bytes, sniffs it (a quick look at the data)
// then returns a result of the sniff.
type Sniffer *func(data []byte) SniffResult

func NewSniffContext() SniffContext {
	c := context.Background()

	return SniffContext{c: &c}
}

// SniffContext is an array of bytes for
// preventing multiple sniff functions
type SniffContext struct {
	c *context.Context
}

// GetData returns the the data for a sniff context
// if no data is present a nil object is returned.
func (s *SniffContext) GetData(key any) any {
	cont := *s.c
	return cont.Value(key)
}

// CacheData, caches a item in the sniff context,
// it can be retrieved with GetData()
func (s *SniffContext) CacheData(key, data any) {
	midC := context.WithValue(*s.c, key, data)
	*s.c = midC
}

// Sniffer takes a stream of bytes, sniffs it (a quick look at the data)
// then returns a result of the sniff.
type SnifferUpdate struct {
	Sniff           func(data []byte) SniffResult
	SniffProperties props
}

// optional properties for identifying clashing functions?
type props struct {
	Desc          string
	FuncSignature string // an xxhash64 of the funcs name and its parameters or something
}
