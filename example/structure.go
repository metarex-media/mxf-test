package example

import (
	"fmt"
	"io"
	"os"

	_ "embed"

	mxftest "github.com/metarex-media/mxf-test"
	. "github.com/onsi/gomega"
)

func RunStructureDemo() {
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
	err = mxftest.MRXTest(f, outf, ExampleStructureSpecifications(sc))
	if err != nil {
		fmt.Printf("error running MRX test for file %s: %v\n", tf, err)
		return
	}

	fmt.Println("successfully generated ", fmt.Sprintf("%s.yml", tf))
}

// ExampleStructureSpecifications returns an example structure
// specification associated with 5.1 R133
func ExampleStructureSpecifications(sc mxftest.SniffContext) mxftest.Specifications {
	return *mxftest.NewSpecification(
		// Assign all the structure tests
		mxftest.WithStructureTests(checkGPStructure),
	)
}

func checkGPStructure(_ io.ReadSeeker, mxf *mxftest.MXFNode) func(t mxftest.Test) {
	return func(t mxftest.Test) {

		// find the generic paritions
		genericParts, gpErr := mxf.Search("select * from partitions where type = " + mxftest.GenericStreamPartition)
		// find the generic partitions positions
		GenericCountPositions := make([]int, len(genericParts))
		for i, gcp := range genericParts {
			GenericCountPositions[i] = gcp.PartitionPos
		}

		// is there a footer partition?
		endPos := len(mxf.Partitions)
		footerParts, footErr := mxf.Search("select * from partitions where type = " + mxftest.FooterPartition)
		if len(footerParts) != 0 {
			endPos--
		}

		// is there a Random Index Partition
		ripParts, ripErr := mxf.Search("select * from partitions where type = " + mxftest.RIPPartition)
		if len(ripParts) != 0 {
			endPos--
		}

		// calculate the expected partitions
		expectedParts := make([]int, len(GenericCountPositions))
		for j := range expectedParts {
			expectedParts[j] = endPos - len(expectedParts) + j
		}

		// run the test comparing the positions
		t.Test("Checking that the generic partition positions match the expected positions at the end of the file", mxftest.NewSpecificationDetails("EBUR133:2012", "5.1", "shall", 3),
			t.Expect(gpErr).To(BeNil()),
			t.Expect(footErr).To(BeNil()),
			t.Expect(ripErr).To(BeNil()),
			t.Expect(expectedParts).Shall(Equal(GenericCountPositions)),
		)
	}
}
