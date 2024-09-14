package mxftest

import (
	"fmt"
	"io"
	"reflect"
	"slices"

	"github.com/metarex-media/mrx-tool/klv"
)

/*
MRXTest tests an MRX file against the specifications given to it, if no
specifications are passed then no tests are run.
These test results are then logged as an yaml file to the io.Writer.
*/
func MRXTest(doc io.ReadSeeker, w io.Writer, testspecs ...Specifications) error {

	klvChan := make(chan *klv.KLV, 1000)

	// get the specifications here
	// testspecs = append(testspecs) //, ISXDSpecifications(), GenericSpecifications())

	// get an identical map of the base tests and
	// the skipped specifications.
	base, skips := generateSpecifications(testspecs...)

	// generate the AST, assigning the tests
	ast, genErr := MakeAST(doc, klvChan, 10, base)

	if genErr != nil {
		return genErr
	}

	// runTags to find which test we actually run
	runTags(doc, ast, skips)
	// testStructure
	tc := NewTestContext(w)

	// load in default of 377 checker etc
	tc.Header("testing mxf file structure", func(t Test) {
		for _, structure := range ast.Tests.tests {
			if *structure.runTest {
				str := *structure.test
				str(doc, ast)(t)
			}
		}
	})

	// 	tc.structureTest(doc, ast, specifications...)

	for _, part := range ast.Partitions {

		// check the essence in each partitoin?
		switch part.Props.PartitionType {
		case HeaderPartition, FooterPartition:

			if len(part.HeaderMetadata) > 0 {
				// delete the map key for tests of this type
				delete(skips.Part, string(Header))

				tc.Header(fmt.Sprintf("testing header metadata of a %s partition at offset %v", part.Props.PartitionType, part.Key.Start), func(t Test) {

					for _, child := range part.HeaderMetadata {
						testChildNodes(doc, child, part.Props.Primer, t, skips)
					}
				})

				tc.Header(fmt.Sprintf("testing header properties of a %s partition at offset %v", part.Props.PartitionType, part.Key.Start), func(t Test) {
					for _, child := range part.Tests.tests {
						if *child.runTest {
							childer := *child.test
							childer(doc, part)(t)
							if !t.testPass() {
								part.FlagFail()
							}
						}
					}
				})
			}
		//	tc.headerTest(doc, part, specifications...)
		case BodyPartition, GenericStreamPartition:
			// delete the skipped partition to prove it has run
			if part.Props.PartitionType == BodyPartition {
				delete(skips.Part, string(Body))
			} else {
				delete(skips.Part, string(GenericBody))
			}

			tc.Header(fmt.Sprintf("testing essence properties at %s partition at offset %v", part.Props.PartitionType, part.Key.Start), func(t Test) {
				for _, tests := range part.Tests.tests {
					if *tests.runTest {
						test := *tests.test
						test(doc, part)(t)
						if !t.testPass() {
							part.FlagFail()
						}
					}
				}
			})
		//	tc.essTest(doc, part, specifications...)
		case RIPPartition:
			// not sure what happens here yet
			// @TODO update this to be a valid checker
		}

	}

	// check for any left over keys in
	for k := range skips.Node {
		tc.RegisterSkippedTest(k, "a skipped node test")
	}

	for k := range skips.Part {
		tc.RegisterSkippedTest(k, "a skipped partition test")
	}

	return tc.EndTest()
}

// run tags runs the specification tags before the actual tests,
// to filter out invalid tests.
func runTags(doc io.ReadSeeker, ast *MXFNode, skips Specifications) {
	tc := NewTestContext(io.Discard)
	defer tc.EndTest()

	// load in default of 377 checker etc
	tc.Header("testing mxf file structure", func(t Test) {
		for _, structure := range ast.markerTests.tests {

			str := *structure.test
			str(doc, ast)(t)
			if !t.testPass() {
				*structure.runTest = false
			}
		}
	})

	for _, part := range ast.Partitions {

		// check the essence in each partitoin?
		switch part.Props.PartitionType {
		case HeaderPartition, FooterPartition:
			delete(skips.markerPart, string(Header))

			if len(part.HeaderMetadata) > 0 {
				// delete the map key for tests of this type

				tc.Header(fmt.Sprintf("testing header metadata of a %s partition at offset %v", part.Props.PartitionType, part.Key.Start), func(t Test) {

					for _, child := range part.HeaderMetadata {
						testChildNodesTags(doc, child, part.Props.Primer, t, skips)
					}
				})

				tc.Header(fmt.Sprintf("testing header properties of a %s partition at offset %v", part.Props.PartitionType, part.Key.Start), func(t Test) {
					for _, child := range part.markerTests.tests {

						childer := *child.test
						childer(doc, part)(t)
						if !t.testPass() {
							*child.runTest = false
						}

					}
				})
			}
		//	tc.headerTest(doc, part, specifications...)
		case BodyPartition, GenericStreamPartition:
			// delete the skipped partition to prove it has run
			if part.Props.PartitionType == BodyPartition {
				delete(skips.markerPart, string(Body))
			} else {
				delete(skips.markerPart, string(GenericBody))
			}
			tc.Header(fmt.Sprintf("testing essence properties at %s partition at offset %v", part.Props.PartitionType, part.Key.Start), func(t Test) {
				for _, tests := range part.markerTests.tests {

					test := *tests.test
					test(doc, part)(t)
					if !t.testPass() {
						*tests.runTest = false
					}

				}
			})
		//	tc.essTest(doc, part, specifications...)
		case RIPPartition:
			// not sure what happens here yet
		}

	}

	// fail all skipped tests

	for k, nodeTests := range skips.markerNode {
		fmt.Println(k)
		for _, test := range nodeTests {
			*test.runTest = false
		}
	}

	for _, partTests := range skips.markerPart {
		for _, test := range partTests {
			*test.runTest = false
		}
	}

}

// testChildNodes run any tests on the metadata and their children
func testChildNodesTags(doc io.ReadSeeker, node *Node, primer map[string]string, t Test, skips Specifications) {

	if node == nil {
		return
	}

	for _, tester := range node.markerTests.testsWithPrimer {
		delete(skips.markerNode, node.Properties.UL())

		test := *tester.test
		test(doc, node, primer)(t)
		if !t.testPass() {
			*tester.runTest = false
		}

	}

	for _, child := range node.Children {
		testChildNodesTags(doc, child, primer, t, skips)
	}
}

func generateSpecifications(testspecs ...Specifications) (base, skips Specifications) {
	base = Specifications{
		Node:       make(map[string][]markedTestWithPrimer[Node]),
		Part:       make(map[string][]markedTest[PartitionNode]),
		MXF:        make([]markedTest[MXFNode], 0),
		markerNode: make(map[string][]markedTestWithPrimer[Node]),
		markerPart: make(map[string][]markedTest[PartitionNode]),
		markerMXF:  make([]markedTest[MXFNode], 0),
		sniffTests: map[*DataIdentifier][]Sniffer{},
	}

	for _, ts := range testspecs {
		markTrue := true
		// assign the nodes
		for key, node := range ts.Node {
			for i, n := range node {
				node[i] = markedTestWithPrimer[Node]{test: n.test, runTest: &markTrue}
			}
			base.Node[key] = append(base.Node[key], node...)
		}
		for key, node := range ts.markerNode {
			for i, n := range node {
				node[i] = markedTestWithPrimer[Node]{test: n.test, runTest: &markTrue}
			}

			base.markerNode[key] = append(base.markerNode[key], node...)
		}

		// assign the partition tests
		for key, node := range ts.Part {
			for i, n := range node {
				node[i] = markedTest[PartitionNode]{test: n.test, runTest: &markTrue}
			}
			base.Part[key] = append(base.Part[key], node...)
		}
		// assign the marked tests
		for key, node := range ts.markerPart {
			for i, n := range node {
				node[i] = markedTest[PartitionNode]{test: n.test, runTest: &markTrue}
			}
			base.markerPart[key] = append(base.markerPart[key], node...)
		}

		// add the sniff tests
		// avoid replications of sniff tests
		if !reflect.DeepEqual(ts.SniffTests, SniffTest{}) {

			for _, sniffTests := range ts.SniffTests.Sniffs {
				// check each sniffer and make sure its not duplicated for that data test
				if !slices.Contains(base.sniffTests[&ts.SniffTests.DataID], sniffTests) {
					base.sniffTests[&ts.SniffTests.DataID] = append(base.sniffTests[&ts.SniffTests.DataID], sniffTests)
				}
			}

		}

		// COPY STRUCTUAL TESTS
		for i, n := range ts.MXF {
			ts.MXF[i] = markedTest[MXFNode]{test: n.test, runTest: &markTrue}
		}
		base.MXF = append(base.MXF, ts.MXF...)

		for i, n := range ts.markerMXF {
			ts.markerMXF[i] = markedTest[MXFNode]{test: n.test, runTest: &markTrue}
		}
		base.markerMXF = append(base.markerMXF, ts.markerMXF...)
	}

	skips = cloneSpeciifcation(base)

	return base, skips
}

// shallow clone of the nodes
func cloneSpeciifcation(base Specifications) Specifications {
	skips := Specifications{Node: make(map[string][]markedTestWithPrimer[Node]),
		markerNode: map[string][]markedTestWithPrimer[Node]{},
		Part:       make(map[string][]markedTest[PartitionNode]),
		markerPart: make(map[string][]markedTest[PartitionNode]),
	}
	for k, v := range base.Node {
		skips.Node[k] = v
	}

	for k, v := range base.markerNode {
		skips.markerNode[k] = v
	}

	// partitions
	for k, v := range base.Part {
		skips.Part[k] = v
	}

	for k, v := range base.markerPart {
		skips.markerPart[k] = v
	}

	return skips
}

// testChildNodes run any tests on the metadata and their children
func testChildNodes(doc io.ReadSeeker, node *Node, primer map[string]string, t Test, skips Specifications) {

	if node == nil {
		return
	}

	for _, tester := range node.Tests.testsWithPrimer {
		delete(skips.Node, node.Properties.UL())
		if *tester.runTest {
			test := *tester.test
			test(doc, node, primer)(t)
			if !t.testPass() {
				node.FlagFail()
			}
		}
	}

	for _, child := range node.Children {
		testChildNodes(doc, child, primer, t, skips)
	}
}

/*
Specifications contains all the information for a test specification.

It contains:

  - The tests for each MXF Node
  - Tags to ensure the specification runs on the correct files
  - Data Sniff tests

These are all optional fields, a specification utilises as
many or as few fields as required to validate the file.
*/
type Specifications struct {
	// node specifications for groups, map is UL node test
	Node       map[string][]markedTestWithPrimer[Node]
	markerNode map[string][]markedTestWithPrimer[Node]
	// test aprtitions the partition tyoe is the map key
	Part       map[string][]markedTest[PartitionNode]
	markerPart map[string][]markedTest[PartitionNode]
	// array of mxf structural tests
	MXF       []markedTest[MXFNode]
	markerMXF []markedTest[MXFNode]
	// Sniff Tests to check the data
	SniffTests SniffTest
	sniffTests map[*DataIdentifier][]Sniffer
}

// NewSpecification generates a new Specifications object.
// It is tailored to the options provided to generate custom specifications and the
// order in which the options are specified are the order in which the executed.
// If no options are provided and empty specifications object is returned.
// An empty specification is still a valid specification
func NewSpecification(options ...func(*Specifications)) *Specifications {

	// make the empty specifications body
	spc := &Specifications{
		Node:       make(map[string][]markedTestWithPrimer[Node]),
		Part:       make(map[string][]markedTest[PartitionNode]),
		MXF:        make([]markedTest[MXFNode], 0),
		markerNode: make(map[string][]markedTestWithPrimer[Node]),
		markerPart: make(map[string][]markedTest[PartitionNode]),
		markerMXF:  make([]markedTest[MXFNode], 0),
	}
	// run through and apply the options
	for _, opt := range options {
		opt(spc)
	}
	return spc
}

// WithStructureTests adds the structure tests to the specification.
// It does not check for repeats so make sure you do not repeat tests.
func WithStructureTests(structureTests ...func(doc io.ReadSeeker, mxf *MXFNode) func(t Test)) func(s *Specifications) {

	return func(s *Specifications) {
		// append each one as a pointer
		for _, st := range structureTests {
			s.MXF = append(s.MXF, markedTest[MXFNode]{test: &st})
		}
	}
}

// WithStructureTag adds the tests to the specification tag.
func WithStructureTag(structureTags ...func(doc io.ReadSeeker, mxf *MXFNode) func(t Test)) func(s *Specifications) {

	return func(s *Specifications) {
		// append each one as a pointer
		for _, st := range structureTags {
			s.markerMXF = append(s.markerMXF, markedTest[MXFNode]{test: &st})
		}
	}
}

// PartitionTest contains the partition type and the test for a partition test.
// The partition types identifies the type of partition this test is targeting.
type PartitionTest struct {
	//
	PartitionType PartitionType
	// The test function
	Test func(doc io.ReadSeeker, partition *PartitionNode) func(t Test)
}

// WithPartitionTests adds the partition tests to the specification
func WithPartitionTests(partitionTests ...PartitionTest) func(s *Specifications) {

	return func(s *Specifications) {
		// append each one as a pointer
		for _, st := range partitionTests {
			// append the test
			s.Part[string(st.PartitionType)] = append(s.Part[string(st.PartitionType)], markedTest[PartitionNode]{test: &st.Test})

		}
	}
}

// WithPartitionTags adds the partition tests to the specification Tag
func WithPartitionTags(partitionTags ...PartitionTest) func(s *Specifications) {

	return func(s *Specifications) {
		// append each one as a pointer
		for _, st := range partitionTags {
			// append the test
			s.markerPart[string(st.PartitionType)] = append(s.markerPart[string(st.PartitionType)], markedTest[PartitionNode]{test: &st.Test})

		}
	}
}

// NodeTest contains the key and a test for a partition test
type NodeTest struct {
	// UL is the Universal Label of the Node being targeted.
	// e.g. 060e2b34.02530105.0e090502.00000000
	// All Universal labels take this form
	UL string
	// The test function
	Test func(doc io.ReadSeeker, node *Node, primer map[string]string) func(t Test)
}

// WithNodeTests adds the Node tests to the specifications object
func WithNodeTests(nodeTests ...NodeTest) func(s *Specifications) {

	return func(s *Specifications) {
		// append each one as a pointer

		for _, st := range nodeTests {
			// append the test
			s.Node[string(st.UL)] = append(s.Node[string(st.UL)], markedTestWithPrimer[Node]{test: &st.Test})
		}
	}
}

// WithSniffTest adds the SniffTest for the specification.
// That is the test that is run on any data found within the file.
func WithSniffTest(sniffTest SniffTest) func(s *Specifications) {
	return func(s *Specifications) {
		s.SniffTests = sniffTest
	}
}

// WithNodeTags adds the Node tests to the specifications object
func WithNodeTags(nodeTags ...NodeTest) func(s *Specifications) {
	return func(s *Specifications) {
		for _, t := range nodeTags {
			s.markerNode[string(t.UL)] = append(s.markerNode[string(t.UL)], markedTestWithPrimer[Node]{test: &t.Test})
		}
	}
}
