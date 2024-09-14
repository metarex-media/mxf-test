package example

import (
	"fmt"
	"io"
	"os"

	"gitlab.com/mm-eng/mxftest"
	"gitlab.com/mm-eng/mxftest/jsonhandle"

	_ "embed"

	. "github.com/onsi/gomega"
)

func RunGPSDemo() {
	tf := "./testdata/gpsdemo.mxf"
	sc := mxftest.NewSniffContext()

	f, err := os.Open(tf)
	if err != nil {
		fmt.Printf("error opening test file %s: %v\n", tf, err)
		return
	}
	defer f.Close() // Ensure the file is closed

	// Create the output file
	outf, err := os.Create(fmt.Sprintf("%s.yml", tf))
	if err != nil {
		fmt.Printf("error creating output file %s.yml: %v\n", tf, err)
		return
	}
	defer outf.Close() // Ensure the file is closed

	// Run the MRX test
	err = mxftest.MRXTest(f, outf, NodeSpecifications(sc))
	if err != nil {
		fmt.Printf("error running MRX test for file %s: %v\n", tf, err)
		return
	}

	fmt.Println("successfully generated ", fmt.Sprintf("%s.yml", tf))
}

//go:embed testdata/gpsSchema.json
var gpsSchema []byte

// GPSSpecifications returns all the specifications
// associated with the demo MRX.123.456.789.GPS namespace
func GPSSpecifications(sc mxftest.SniffContext) mxftest.Specifications {

	schemaSniff, _ := jsonhandle.SchemaCheck(sc, gpsSchema, "GPSSchema")

	return *mxftest.NewSpecification(
		// Assign a sniff test
		mxftest.WithSniffTest(mxftest.SniffTest{DataID: jsonhandle.DataIdentifier, Sniffs: []mxftest.Sniffer{schemaSniff}}),
		// Assign a partitionTest
		mxftest.WithPartitionTests(
			mxftest.PartitionTest{PartitionType: mxftest.Body, Test: testBodyPartitionForGPS},
		),
	)

}

func testBodyPartitionForGPS(_ io.ReadSeeker, header *mxftest.PartitionNode) func(t mxftest.Test) {
	return func(t mxftest.Test) {

		// check all the files are ttml
		nongps, gpsSearchErr := header.Search("select * from essence where sniff:GPSSchema <> pass")

		t.Test("checking that the partition only contains gps json files", mxftest.NewSpecificationDetails("DemoSpec", "X.X", "shall", 1),
			t.Expect(gpsSearchErr).Shall(BeNil()),
			t.Expect(len(nongps)).Shall(Equal(0), "Non GPS files found in the partition"),
			t.Expect(len(header.Essence)).ShallNot(Equal(0), "no essence found in the generic partition"),
		)
	}
}
