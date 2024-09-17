// xmlhandle contains the xml data interactions, for using in sniff tests of MXF data.
package xmlhandle

import (
	"bytes"
	"encoding/xml"
	"io"

	"github.com/antchfx/xmlquery"
	mxftest "github.com/metarex-media/mxf-test"
)

const (
	// Content is the xml ContentType
	Content mxftest.CType = "text/xml"
)

// xml identifier function, returns true of the data stream is
// valid xml
func xMLIdentifier(data []byte) bool {

	// to avoid json or yaml slipping through identify the first char
	if len(data) < 4 {
		return false
	}
	// check that it starts with the right key at leasr
	start := data[:4]
	if start[0] != '<' {
		return false
	}

	// create the decoder for the bytes
	decoder := xml.NewDecoder(bytes.NewBuffer(data))
	var err error

	for err == nil {
		err = decoder.Decode(new(interface{}))
	}

	// did it EOF and is valid or was it bad XML?
	return err == io.EOF

}

// DataIdentifier is the xml identifier function
var DataIdentifier = mxftest.DataIdentifier{DataFunc: xMLIdentifier, ContentType: Content}

type contKey struct {
	path, functionName string
}

// PathSniffer searches an XML document for that path
// and stores the key value of the Node
//
// It searches using the xPath library https://github.com/antchfx/xpath
/*

Common searches include:

- "/*" - find the root element
- "namespace-uri(/*)" - find the namespace of the root element
*/
func PathSniffer(sc mxftest.SniffContext, path string) mxftest.Sniffer {

	pathKey := contKey{path: path, functionName: "the path sniffer function using xpath"}
	sniffFunc := sc.GetData(pathKey)

	if sniffFunc != nil {
		return sniffFunc.(mxftest.Sniffer)
	}

	var xmlSniff mxftest.Sniffer

	// short cut to avoid the blocker
	if path == "namespace-uri(/*)" {

		mid := func(data []byte) mxftest.SniffResult {
			doc, _ := xmlquery.Parse(bytes.NewBuffer(data))
			// this means find the root of ttml is tt
			out := xmlquery.FindOne(doc, "/*")

			if out != nil {
				// loop through the attributes searching for xmlns
				for _, attr := range out.Attr {
					if attr.Name.Local == "xmlns" {
						return mxftest.SniffResult{Key: path, Field: attr.Value, Certainty: 100}
					}
				}
			}

			return mxftest.SniffResult{}
		}
		xmlSniff = &mid
	} else {

		mid := func(data []byte) mxftest.SniffResult {
			doc, _ := xmlquery.Parse(bytes.NewBuffer(data))
			// this means find the root of ttml is tt
			out := xmlquery.FindOne(doc, path)

			if out == nil {

				return mxftest.SniffResult{}
			}

			var value string
			switch out.Type {
			case xmlquery.AttributeNode:
				value = out.InnerText()
			default:
				value = out.Data
			}

			return mxftest.SniffResult{Key: path, Field: value, Certainty: 100}
		}

		xmlSniff = &mid
	}

	sc.CacheData(pathKey, xmlSniff)

	return xmlSniff
}
