package jsonhandle

import (
	"encoding/json"

	mxftest "github.com/metarex-media/mxf-test"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const (
	// Content is the JSON content type
	Content mxftest.CType = "application/json"
)

// DataIdentifier is the JSON identifier function
var DataIdentifier = mxftest.DataIdentifier{DataFunc: jSONIdentifier, ContentType: Content}

func jSONIdentifier(dataStream []byte) bool {
	var js json.RawMessage

	return json.Unmarshal(dataStream, &js) == nil
}

type contKey struct {
	path, functionName string
}

// SchemaCheck creates a sniffer test where the bytes are compared against a **local** schema.
// If the object passes it is returned with the field of "pass",
// otherwise an empty object is returned, with no inclination of why it failed the test.
//
// This is quite a slow snifftest as it checks all the data is valid, so do not expect quick results for:
// - Large JSON files
// - Large and complex schemas
func SchemaCheck(sc mxftest.SniffContext, schemaFile []byte, key string) (mxftest.Sniffer, error) {

	// check if this functions has been made before
	pathKey := contKey{path: key, functionName: "the schema checker using "}
	sniffFunc := sc.GetData(pathKey)

	if sniffFunc != nil {
		return sniffFunc.(mxftest.Sniffer), nil
	}

	// create the schema compiler outside of the function to speed
	// up the program
	c := jsonschema.NewCompiler()

	var schMid any
	err := json.Unmarshal(schemaFile, &schMid)
	if err != nil {
		return nil, err
	}

	c.AddResource("", schMid)
	sch, err := c.Compile("")

	if err != nil {
		return nil, err
	}

	jsonSniff := func(b []byte) mxftest.SniffResult {

		var doc any
		err := json.Unmarshal(b, &doc)

		if err != nil {
			return mxftest.SniffResult{}
		}

		err = sch.Validate(doc)

		if err == nil {
			return mxftest.SniffResult{Key: key, Certainty: 100, Field: "pass"}
		}

		return mxftest.SniffResult{}
	}

	// cache the function
	var jSniff mxftest.Sniffer = &jsonSniff
	sc.CacheData(pathKey, jSniff)

	return jSniff, nil
}
