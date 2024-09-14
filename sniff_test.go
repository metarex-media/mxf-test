package mxftest

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSniff(t *testing.T) {

	expected := []map[string]*SniffResult{
		{"ContentType": {Key: "ContentType", Field: "demo", Certainty: 0}, "test": {Key: "test", Field: "pass", Data: demoDataKey, Certainty: 100}},
		{"ContentType": {Key: "ContentType", Field: "demo", Certainty: 0}, "test": {Key: "test", Field: "pass", Data: demoDataKey, Certainty: 100}},
	}
	dataExpec := []map[*DataIdentifier][]Sniffer{
		{mockPasser(true, demoDataKey): {mockSniffer("test", "pass", 100)}},
		{mockPasser(true, demoDataKey): {mockSniffer("test", "pass", 100)}, mockPasser(false, demoDataKeyFail): {mockSniffer("test", "pass", 100)}},
	}

	for i, e := range expected {

		res := Sniff([]byte{}, dataExpec[i])

		Convey("Checking the sniff function runs as intended, for various data types", t, func() {
			Convey(fmt.Sprintf("Running with mock tests that should produce a sniff result of %v", e), func() {
				Convey("The expected map matches the expected output", func() {
					So(res, ShouldResemble, e)
				})
			})
		})
	}

	res := Sniff([]byte{}, map[*DataIdentifier][]Sniffer{mockPasser(false, demoDataKeyFail): {mockSniffer("test", "pass", 100)}})

	Convey("Checking the sniff function returns an empty map when no data types are found", t, func() {
		Convey("running with a tests that fail straight away", func() {
			Convey("The returned map is empty", func() {
				So(res, ShouldResemble, make(map[string]*SniffResult))
			})
		})
	})

}

const (
	demoDataKey     CType = "demo"
	demoDataKeyFail CType = "fail"
)

// return a function with a preset pass or fail
func mockPasser(pass bool, dtype CType) *DataIdentifier {
	return &DataIdentifier{
		DataFunc:    func(b []byte) bool { return pass },
		ContentType: dtype,
	}
}

// assign the intended outputs skipping the sniffing stage
func mockSniffer(key, out string, certain float64) Sniffer {
	outFunc := func(_ []byte) SniffResult {
		return SniffResult{Key: key, Field: out, Certainty: certain}
	}

	return &outFunc
}
