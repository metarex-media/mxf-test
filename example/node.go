package example

import (
	"fmt"
	"io"
	"os"

	mxf2go "github.com/metarex-media/mxf-to-go"
	. "github.com/onsi/gomega"
	"gitlab.com/mm-eng/mxftest"
)

func RunNodeDemo() {
	tf := "../testdata/demoReports/goodISXD.mxf"
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

// NodeSpecifications returns a specifications
// for testing the node to ISXD specifications for examples
func NodeSpecifications(sc mxftest.SniffContext) mxftest.Specifications {

	return *mxftest.NewSpecification(
		// assign the node test
		mxftest.WithNodeTests(mxftest.NodeTest{UL: mxf2go.GISXDUL[13:], Test: nodeISXDDescriptor}),
	)

}

func nodeISXDDescriptor(doc io.ReadSeeker, isxdDesc *mxftest.Node, primer map[string]string) func(t mxftest.Test) {

	return func(t mxftest.Test) {

		// check we have the node
		t.Test("Checking that the ISXD descriptor is present in the header metadata", mxftest.NewSpecificationDetails("RDD47:2018", "9.2", "shall", 1),
			t.Expect(isxdDesc).ShallNot(BeNil()),
		)

		if isxdDesc != nil {
			// decode the group into its parts
			isxdDecode, err := mxftest.DecodeGroupNode(doc, isxdDesc, primer)

			t.Test("Checking that the data essence coding field is present in the ISXD descriptor", mxftest.NewSpecificationDetails("RDD47:2018", "9.3", "shall", 1),
				t.Expect(err).Shall(BeNil()),
				t.Expect(isxdDecode["DataEssenceCoding"]).Shall(Equal(mxf2go.TAUID{
					Data1: 101591860,
					Data2: 1025,
					Data3: 261,
					Data4: mxf2go.TUInt8Array8{14, 9, 6, 6, 0, 0, 0, 0},
				})))
		}
	}
}
