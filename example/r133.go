package example

import (
	"io"

	mxftest "github.com/metarex-media/mxf-test"
	"github.com/metarex-media/mxf-test/xmlhandle"
	. "github.com/onsi/gomega"
)

// R133Specifications returns all the specifications
// associated with 5.1 R133
func R133Specifications(sc mxftest.SniffContext) mxftest.Specifications {
	return *mxftest.NewSpecification(
		// Assign a sniff test
		mxftest.WithSniffTest(mxftest.SniffTest{DataID: xmlhandle.DataIdentifier, Sniffs: []mxftest.Sniffer{xmlhandle.PathSniffer(sc, "/*")}}),
		// Assign all the partitions
		mxftest.WithPartitionTests(
			mxftest.PartitionTest{PartitionType: mxftest.GenericBody, Test: testGenericPartitionForTT},
		),
		// Assign all the structure tests
		mxftest.WithStructureTests(checkStructure2),
	)
}

func checkStructure2(_ io.ReadSeeker, mxf *mxftest.MXFNode) func(t mxftest.Test) {
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

func testGenericPartitionForTT(_ io.ReadSeeker, header *mxftest.PartitionNode) func(t mxftest.Test) {
	return func(t mxftest.Test) {
		// check all the files are xml

		nonttml, ttmlSearchErr := header.Search("select * from essence where sniff:/* <> tt")
		t.Test("checking that the partition only contains ttml files", mxftest.NewSpecificationDetails("EBUR133:2012", "5.1", "shall", 1),
			t.Expect(ttmlSearchErr).Shall(BeNil()),
			t.Expect(len(nonttml)).Shall(Equal(0), "uh oh, does not contain TTML"),
			t.Expect(len(header.Essence)).ShallNot(Equal(0), "no essence found in the generic partition"),
		)
	}
}

/*

410 tests?

check the partition body contents.
check essence keys


*/
