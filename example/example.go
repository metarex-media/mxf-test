// package example contains example implementations of the mxftest repo
package example

import (
	"fmt"
	"io"

	mxf2go "github.com/metarex-media/mxf-to-go"
	. "github.com/onsi/gomega"
	"gitlab.com/mm-eng/mxftest"
	"gitlab.com/mm-eng/mxftest/xmlhandle"
)

// ISXDDoc is the document the specs for
// ISXD are found in and used for these tests.
const ISXDDoc = "RDD47:2018"

// ISXDSpecifications returns all the specifications
// associated with ISXD
func ISXDSpecifications(sc mxftest.SniffContext) mxftest.Specifications {

	return *mxftest.NewSpecification(
		// Assign a sniff test
		mxftest.WithSniffTest(mxftest.SniffTest{DataID: xmlhandle.DataIdentifier, Sniffs: []mxftest.Sniffer{xmlhandle.PathSniffer(sc, "/*"), xmlhandle.PathSniffer(sc, "namespace-uri(/*)")}}),
		// Assign any node tests
		mxftest.WithNodeTests(mxftest.NodeTest{UL: mxf2go.GISXDUL[13:], Test: testISXDDescriptor}),
		// Assign all the partitions
		mxftest.WithPartitionTests(
			mxftest.PartitionTest{PartitionType: mxftest.Header, Test: genericCountCheck},
			mxftest.PartitionTest{PartitionType: mxftest.Body, Test: testFrameWrapped},
			mxftest.PartitionTest{PartitionType: mxftest.GenericBody, Test: testGenericPartition},
		),
		// Assign all the structure tests
		mxftest.WithStructureTests(checkStructure, checkDataTypes, checkNameSpaces),
		//
		mxftest.WithNodeTags(mxftest.NodeTest{UL: mxf2go.GISXDUL[13:], Test: ISXDNodeTag}),
	)

}

func ISXDNodeTag(doc io.ReadSeeker, isxdDesc *mxftest.Node, primer map[string]string) func(t mxftest.Test) {

	return func(t mxftest.Test) {
		// all thats needed for isxd is a descriptor
		t.Test("Checking that the ISXD descriptor is present in the header metadata", mxftest.NewSpecificationDetails(ISXDDoc, "9.2", "shall", 1),
			t.Expect(isxdDesc).ShallNot(BeNil()),
		)

	}
}

func testISXDDescriptor(doc io.ReadSeeker, isxdDesc *mxftest.Node, primer map[string]string) func(t mxftest.Test) {

	return func(t mxftest.Test) {

		// rdd-47:2009/11.5.3/shall/4
		t.Test("Checking that the ISXD descriptor is present in the header metadata", mxftest.NewSpecificationDetails(ISXDDoc, "9.2", "shall", 1),
			t.Expect(isxdDesc).ShallNot(BeNil()),
		)

		if isxdDesc != nil {
			// decode the group
			isxdDecode, err := mxftest.DecodeGroupNode(doc, isxdDesc, primer)

			t.Test("Checking that the data essence coding field is present in the ISXD descriptor", mxftest.NewSpecificationDetails(ISXDDoc, "9.3", "shall", 1),
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

// check the generic body partition count and layout to the ISXD spec
func genericCountCheck(doc io.ReadSeeker, header *mxftest.PartitionNode) func(t mxftest.Test) {

	return func(t mxftest.Test) {

		genericParts, err := header.Parent.Search("select * from partitions where type = " + mxftest.GenericStreamPartition)

		t.Test("Checking there is no error getting the generic partition streams", mxftest.NewSpecificationDetails(ISXDDoc, "5.4", "shall", 1),
			t.Expect(err).To(BeNil()),
		)

		if len(genericParts) > 0 {
			// ibly run if there's any generic essence
			// update to a partitionsearch

			staticTracks, err := header.Search("select * from metadata where UL = " + mxf2go.GStaticTrackUL[13:])
			t.Test("Checking that a single static track is present in the header metadata", mxftest.NewSpecificationDetails(ISXDDoc, "5.4", "shall", 1),
				t.Expect(err).To(BeNil()),
				t.Expect(len(staticTracks)).Shall(Equal(1)),
			)

			if len(staticTracks) == 1 {
				staticTrack := staticTracks[0]

				t.Test("Checking that the static track is not nil", mxftest.NewSpecificationDetails(ISXDDoc, "5.4", "shall", 1),
					t.Expect(staticTrack).ShallNot(BeNil()),
				)

				sequence, err := staticTrack.Search("select * where UL = " + mxf2go.GSequenceUL[13:])
				t.Test("Checking that the static track points to a single sequence", mxftest.NewSpecificationDetails(ISXDDoc, "5.4", "shall", 2),
					t.Expect(err).Shall(BeNil()),
					t.Expect(len(sequence)).Shall(Equal(1), fmt.Sprintf("Wanted 1 sequence item, received %v", len(sequence))),
				)

				if len(sequence) == 1 {
					t.Test("Checking that the static track sequence has as many sequence children as partitions", mxftest.NewSpecificationDetails(ISXDDoc, "5.4", "shall", 2),
						t.Expect(len(sequence[0].Children)).Shall(Equal(len(genericParts))),
					)
				}
			}
		}

	}
	// test ISXD descriptor

}

// this is a header test
func testFrameWrapped(doc io.ReadSeeker, header *mxftest.PartitionNode) func(t mxftest.Test) {
	return func(t mxftest.Test) {

		if len(header.Essence) > 0 {

			badKeys, err := header.Search("select * from essence where UL <> " + mxf2go.FrameWrappedISXDData.UL[13:])

			t.Test("Checking that the only ISXD essence keys are found in body partitions", mxftest.NewSpecificationDetails(ISXDDoc, "7.5", "shall", 1),
				t.Expect(err).Shall(BeNil()),
				t.Expect(len(badKeys)).Shall(Equal(0), fmt.Sprintf("%v other essence keys found", len(badKeys))),
			)

			if len(badKeys) != 0 {

				fwPattern := header.Props.EssenceOrder
				breakPoint := 0
				// check each header against the pattern.
				var extractErr error
				//
				for i, e := range header.Essence {
					ess, err := mxftest.NodeToKLV(doc, e)
					if mxftest.FullNameMask(ess.Key) != fwPattern[i%len(fwPattern)] {
						breakPoint = e.Key.Start
						break
					}

					if err != nil {
						extractErr = err
						break
					}

				}

				t.Test("Checking that the content package order are regular throughout the essence stream", mxftest.NewSpecificationDetails(ISXDDoc, "7.5", "shall", 1),
					t.Expect(extractErr).Shall(BeNil()),
					t.Expect(breakPoint).Shall(Equal(0), fmt.Sprintf("irregular key found at byte offset %v", breakPoint)),
				)
			}
		}
	}
}

func testGenericPartition(doc io.ReadSeeker, header *mxftest.PartitionNode) func(t mxftest.Test) {
	return func(t mxftest.Test) {

		headerKLV, err := mxftest.NodeToKLV(doc, &mxftest.Node{Key: header.Key, Length: header.Length, Value: header.Value})
		mp := mxftest.PartitionExtract(headerKLV)

		t.Test("Checking that the index byte count for the generic header is 0", mxftest.NewSpecificationDetails(ISXDDoc, "7.5", "shall", 1),
			t.Expect(err).Shall(BeNil()),
			t.Expect(mp.IndexByteCount).Shall(Equal(uint64(0)), "index byte count not 0"),
		)

		t.Test("Checking that the header metadata byte count for the generic header is 0", mxftest.NewSpecificationDetails(ISXDDoc, "7.5", "shall", 1),
			t.Expect(mp.HeaderByteCount).Shall(Equal(uint64(0)), "header metadata byte count not 0"),
		)

		t.Test("Checking that the index SID for the generic header is 0", mxftest.NewSpecificationDetails(ISXDDoc, "7.5", "shall", 1),
			t.Expect(mp.IndexSID).Shall(Equal(uint32(0)), "index SID not 0"),
		)

		t.Test("checking the partition key meets the expected value of "+mxf2go.GGenericStreamPartitionUL[13:], mxftest.NewSpecificationDetails(ISXDDoc, "7.5", "shall", 1),
			t.Expect(mxftest.FullNameMask(headerKLV.Key, 5)).Shall(Equal(mxf2go.GGenericStreamPartitionUL[13:])),
		)

		// 060e2b34.0101010c.0d010509.01000000 as the value is not used in the registers (yet?)
		gpEssKey := "060e2b34.0101010c.0d010509.01000000"
		invalidKeys, err := header.Search("select * from essence where ul <> " + gpEssKey)
		// 09.01 - 1001 -little endin & 01 - makrer bit
		// can be shown as this but is not in the essence
		// 060e2b34.0101010c.0d01057f.7f000000

		t.Test("checking the essence keys all have the value of "+gpEssKey, mxftest.NewSpecificationDetails(ISXDDoc, "7.5", "shall", 1),
			t.Expect(err).Shall(BeNil()),
			t.Expect(len(invalidKeys)).Shall(Equal(0), fmt.Sprintf("%v other essence keys found", len(invalidKeys))),
		)
	}
}

func checkStructure(doc io.ReadSeeker, mxf *mxftest.MXFNode) func(t mxftest.Test) {
	return func(t mxftest.Test) {

		// find the generic paritions

		genericParts, gpErr := mxf.Search("select * from partitions where type = " + mxftest.GenericStreamPartition)
		// find the generic partitions positions
		GenericCountPositions := make([]int, len(genericParts))
		for i, gcp := range genericParts {
			GenericCountPositions[i] = gcp.PartitionPos
		}

		endPos := len(mxf.Partitions)
		footerParts, footErr := mxf.Search("select * from partitions where type = " + mxftest.FooterPartition)
		if len(footerParts) != 0 {
			endPos--
		}

		ripParts, ripErr := mxf.Search("select * from partitions where type = " + mxftest.RIPPartition)
		if len(ripParts) != 0 {
			endPos--
		}

		expectedParts := make([]int, len(GenericCountPositions))
		for j := range expectedParts {
			expectedParts[j] = endPos - len(expectedParts) + j
		}
		t.Test("Checking that the generic partition positions match the expected positions at the end of the file", mxftest.NewSpecificationDetails(ISXDDoc, "5.4", "shall", 3),
			t.Expect(gpErr).To(BeNil()),
			t.Expect(footErr).To(BeNil()),
			t.Expect(ripErr).To(BeNil()),
			t.Expect(expectedParts).Shall(Equal(GenericCountPositions)),
		)
	}
}

func checkDataTypes(_ io.ReadSeeker, mxf *mxftest.MXFNode) func(t mxftest.Test) {
	return func(t mxftest.Test) {
		nonXMLCount := 0

		//

		// loop through all the essence checking what data was found when
		// we had a peak at it
		partitions, bErr := mxf.Search("select * from partitions where essence <> 0")
		var xmlSearchErr error
		for _, parts := range partitions {
			nonXml, xmlSearchErr := parts.Search(fmt.Sprintf("select * from essence where sniff:%s <> %s", mxftest.ContentTypeKey, xmlhandle.MIME))
			if xmlSearchErr != nil {
				break
			}
			nonXMLCount += len(nonXml)
		}

		t.Test("Checking only xml data is contained in the ISXD file", mxftest.NewSpecificationDetails(ISXDDoc, "5.3", "shall", 1),
			t.Expect(bErr).Shall(BeNil()),
			t.Expect(xmlSearchErr).Shall(BeNil()),
			t.Expect(nonXMLCount).Shall(Equal(0), fmt.Sprintf("%v non xml entries found", nonXMLCount)),
		)

		if nonXMLCount == 0 {
			roots := make(map[string]bool)

			// loop through all the essence checking what data was found when
			// we had a peak at it
			bodyParts, bErr := mxf.Search("select * from partitions where essence <> 0 and type <> " + mxftest.GenericStreamPartition)
			for _, parts := range bodyParts {
				for _, ess := range parts.Essence {
					// checking the root
					if snif, ok := ess.Sniffs["/*"]; ok {
						roots[snif.Field] = true
					}
				}
			}

			t.Test("Checking every XML file has the same root element", mxftest.NewSpecificationDetails(ISXDDoc, "5.3", "shall", 2),
				t.Expect(bErr).Shall(BeNil()),
				t.Expect(len(roots)).To(Equal(1), fmt.Sprintf("%v xml roots found, wanted 1", len(roots))),
			)
		}
	}
}

func checkNameSpaces(doc io.ReadSeeker, mxf *mxftest.MXFNode) func(t mxftest.Test) {

	return func(t mxftest.Test) {

		headers, searchErr := mxf.Search("select * from partitions where metadata <> 0")
		if len(headers) == 0 {
			// no headers so we can't do the rest of the test
			return
		}

		// make a header using the closest partition to the end
		header := headers[len(headers)-1]

		isxdDesc, isxdErr := header.Search("select * from metadata where UL = " + mxf2go.GISXDUL[13:])

		t.Test("Checking that the ISXD descriptor is present", mxftest.NewSpecificationDetails(ISXDDoc, "9.2", "shall", 1),
			t.Expect(searchErr).To(BeNil()),
			t.Expect(isxdErr).To(BeNil()),
			t.Expect(isxdDesc).ToNot(BeNil()),
		)

		if isxdDesc != nil {
			// decode the group
			isxdDecode, err := mxftest.DecodeGroupNode(doc, isxdDesc[0], header.Props.Primer)

			t.Test("Checking that the NameSpaceURI field is present in the ISXD descriptor", mxftest.NewSpecificationDetails(ISXDDoc, "9.3", "table4", 3),
				t.Expect(err).Shall(BeNil()),
				t.Expect(isxdDecode["NamespaceURIUTF8"]).ShallNot(BeNil()),
			)

			if ns, ok := isxdDecode["NamespaceURIUTF8"]; ok {

				invalidNSCount := 0
				var nsErr error

				bodies, bErr := mxf.Search("select * from partitions where essence <> 0 AND type <> " + mxftest.GenericStreamPartition)
				for _, b := range bodies {
					// check each partition for
					invalidNS, nsErr := b.Search("select * from essence where sniff:namespace-uri(/*) <> " + ns.(string))
					if nsErr != nil {
						break
					}
					// checking for the namespace that's given
					invalidNSCount += len(invalidNS)
				}

				t.Test(fmt.Sprintf("Checking that the NameSpaceURI field of %s matches the values given in the essence across the file", ns), mxftest.NewSpecificationDetails(ISXDDoc, "5.3", "shall", 2),
					t.Expect(bErr).Shall(BeNil()),
					t.Expect(nsErr).Shall(BeNil()),
					t.Expect(invalidNSCount).Shall(Equal(0), fmt.Sprintf("expected 0 invalid namespaces that did not match %s got %v", ns, invalidNSCount)),
				)
			}

		}

	}

}
