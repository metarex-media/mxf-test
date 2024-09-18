package mxftest

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/metarex-media/mrx-tool/klv"
	mxf2go "github.com/metarex-media/mxf-to-go"

	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/yaml.v3"
)

func TestAST(t *testing.T) {

	mxfToTest := []string{"./testdata/demoReports/goodISXD.mxf",
		"./testdata/demoReports/veryBadISXD.mxf", "./testdata/demoReports/badISXD.mxf"}
	for _, mxf := range mxfToTest {

		doc, docErr := os.Open(mxf)

		klvChan := make(chan *klv.KLV, 1000)

		// generate the AST, assigning the tests
		ast, genErr := MakeAST(doc, klvChan, 10, *NewSpecification())

		astBytes, yamErr := yaml.Marshal(ast)
		f, _ := os.Create(fmt.Sprintf("%v-ast.yaml", mxf))
		f.Write(astBytes)
		expecBytes, expecErr := os.ReadFile(fmt.Sprintf("%v-ast.yaml", mxf))
		htest := sha256.New()
		htest.Write(astBytes)
		hnormal := sha256.New()
		hnormal.Write(expecBytes)

		Convey("Checking AST maps are consistent and are in the expected form", t, func() {
			Convey(fmt.Sprintf("generating an AST of %s, which is saved as a yaml", mxf), func() {
				Convey("The generated yaml matches the expected yaml", func() {

					So(docErr, ShouldBeNil)
					So(genErr, ShouldBeNil)
					So(yamErr, ShouldBeNil)
					So(expecErr, ShouldBeNil)
					So(fmt.Sprintf("%x", htest.Sum(nil)), ShouldResemble, fmt.Sprintf("%x", hnormal.Sum(nil)))
				})
			})
		})
	}
}

func TestASTErrors(t *testing.T) {
	headerKey := []byte{06, 0x0e, 0x2b, 0x34, 02, 05, 01, 01, 0x0d, 01, 02, 01, 01, 02, 01, 00}
	badInputs := [][]byte{{}, make([]byte, 100),
		{06, 0x0e, 0x2b, 0x34, 02, 05, 01, 01, 0x0d, 01, 02, 01, 01, 02, 01, 00, 0, 0, 0, 0, 0, 0},
		append(headerKey, make([]byte, 10000)...)}

	expectedErrMess := []string{"empty data stream", "Buffer stream unexpectedly closed, was expecting at least 1 more bytes",
		"Buffer stream unexpectedly closed, was expecting at least 11 more bytes", "Buffer stream unexpectedly closed, was expecting at least 13 more bytes"}

	for i, input := range badInputs {

		doc := bytes.NewReader(input)
		klvChan := make(chan *klv.KLV, 1000)
		_, genErr := MakeAST(doc, klvChan, 10, *NewSpecification())

		Convey("Checking errors are returned when making the AST", t, func() {
			Convey(fmt.Sprintf("generating the ast with a byte stream with an expected error of %s", expectedErrMess[i]), func() {
				Convey("The generated error is expected", func() {
					So(genErr, ShouldResemble, fmt.Errorf(expectedErrMess[i]))
				})
			})
		})
	}
}

/*
tests make one that passes and fais for each type then run the tests with an
expected outcome.

Run at the main part of the functions, look for corner cases etc
*/
func TestTests(t *testing.T) {

	// Set up some specifications that always pass
	setUps := []Specifications{
		*NewSpecification(),
		*NewSpecification(WithNodeTests(makeNodeTest(true))),
		*NewSpecification(WithPartitionTests(makePartitionTest(true))),
		*NewSpecification(WithStructureTests(makeStructureTest(true))),
		*NewSpecification(WithStructureTests(makeStructureTest(true)), WithNodeTests(makeNodeTest(true)), WithPartitionTests(makePartitionTest(true))),
	}

	for _, s := range setUps {
		f, _ := os.Open("./testdata/all.mxf")

		var buf bytes.Buffer
		testErr := MRXTest(f, &buf, s)

		var rep Report
		marshErr := yaml.Unmarshal(buf.Bytes(), &rep)

		Convey("Checking that tests that whe all tests pass the global pass reflects this", t, func() {
			Convey(fmt.Sprintf("running with the following specifications %v", s), func() {
				Convey("No error is returned and the global test passes", func() {
					So(testErr, ShouldBeNil)
					So(marshErr, ShouldBeNil)
					So(rep.TestPass, ShouldBeTrue)
					So(len(rep.SkippedTests), ShouldEqual, 0)
				})
			})
		})

		f.Close()
	}

	setUpFails := []Specifications{
		*NewSpecification(WithNodeTests(makeNodeTest(false))),
		*NewSpecification(WithPartitionTests(makePartitionTest(false))),
		*NewSpecification(WithStructureTests(makeStructureTest(false))),
		*NewSpecification(WithStructureTests(makeStructureTest(false)), WithNodeTests(makeNodeTest(false)), WithPartitionTests(makePartitionTest(false))),
	}

	for _, s := range setUpFails {
		f, _ := os.Open("./testdata/all.mxf")

		var buf bytes.Buffer
		testErr := MRXTest(f, &buf, s)
		fmt.Println(buf.String(), "HERE")
		var rep Report
		marshErr := yaml.Unmarshal(buf.Bytes(), &rep)

		Convey("Checking that tests that fail trigger the global fail", t, func() {
			Convey(fmt.Sprintf("running with the following specifications %v", s), func() {
				Convey("No error is returned and the global test fails", func() {
					So(testErr, ShouldBeNil)
					So(marshErr, ShouldBeNil)
					So(rep.TestPass, ShouldBeFalse)
					So(len(rep.SkippedTests), ShouldEqual, 0)
				})
			})
		})

		f.Close()
	}

}

func TestTags(t *testing.T) {
	setUpFails := []Specifications{
		*NewSpecification(WithNodeTags(makeNodeTest(false)), WithNodeTests(makeNodeTest(false))),
		*NewSpecification(WithPartitionTags(makePartitionTest(false)), WithNodeTests(makeNodeTest(false))),
		*NewSpecification(WithPartitionTags(makePartitionTest(false)), WithNodeTests(makeNodeTest(false))),
		*NewSpecification(WithStructureTag(makeStructureTest(false)), WithNodeTests(makeNodeTest(false))),
	}

	for _, s := range setUpFails {
		f, _ := os.Open("./testdata/all.mxf")

		var buf bytes.Buffer
		testErr := MRXTest(f, &buf, s)

		var rep Report
		marshErr := yaml.Unmarshal(buf.Bytes(), &rep)

		Convey("Checking that tags successfully fail", t, func() {
			Convey(fmt.Sprintf("running with the following specifications %v, where the tag is always set to fail", s), func() {
				Convey("No error is returned and the global test passes, because the failing test is not run", func() {
					So(testErr, ShouldBeNil)
					So(marshErr, ShouldBeNil)
					So(rep.TestPass, ShouldBeTrue)
					So(len(rep.SkippedTests), ShouldEqual, 0)
				})
			})
		})
		f.Close()
	}
}

func TestSearches(t *testing.T) {

	targetNode := &Node{Sniffs: map[string]*SniffResult{"dummy": {Key: "test", Field: "testfield"}}, Properties: EssenceProperties{EssUL: "060e2b34.027f0101.0d010101.01010f00"}}
	badSibling := &Node{Properties: EssenceProperties{EssUL: "anInvalidField"}}
	middleNode := &Node{Children: []*Node{targetNode, badSibling}, Properties: EssenceProperties{EssUL: "middle"}}
	parentNode := &Node{Children: []*Node{middleNode}, Properties: EssenceProperties{EssUL: "Parent"}}

	searches := []string{"select * where UL = 060e2b34.027f0101.0d010101.01010f00",
		"select * where sniff:dummy = testfield",
		"select * where UL <> anInvalidField"}

	expected := [][]*Node{
		{targetNode},
		{targetNode},
		{middleNode, targetNode},
	}

	for i, s := range searches {
		found, err := parentNode.Search(s)

		Convey("Checking the Node search functions", t, func() {
			Convey(fmt.Sprintf("running a search of %s", s), func() {
				Convey("No error is returned and the expected nodes are returned", func() {
					So(err, ShouldBeNil)
					So(found, ShouldResemble, expected[i])
				})
			})
		})
	}

	badSearches := []string{"selec * where UL = 060e2b34.027f0101.0d010101.01010f00",
		"select * where unknownField = testfield",
		"select * where UL <=> anInvalidField"}
	badError := []string{"first word not select", "unknown field \"unknownField\"", "unknown comparison operator \"<=>\""}

	for i, bs := range badSearches {
		found, err := parentNode.Search(bs)

		Convey("Checking the Node search functions return the correct errors", t, func() {
			Convey(fmt.Sprintf("running a search of %s", bs), func() {
				Convey("An error is returned with no nodes", func() {
					So(err, ShouldResemble, fmt.Errorf(badError[i]))
					So(len(found), ShouldEqual, 0)
				})
			})
		})
	}

	dummyPart := &PartitionNode{Essence: []*Node{parentNode}, HeaderMetadata: []*Node{parentNode}, Props: PartitionProperties{PartitionType: string(Header)}}
	partSearches := []string{"select * from metadata where UL = 060e2b34.027f0101.0d010101.01010f00",
		"select * from metadata where UL <> anInvalidField",
		"select * from essence where UL = 060e2b34.027f0101.0d010101.01010f00",
		"select * from essence where sniff:dummy = testfield",
		"select * from essence where UL <> anInvalidField"}

	partExpected := [][]*Node{
		{targetNode},
		{parentNode, middleNode, targetNode},
		{targetNode},
		{targetNode},
		{parentNode, middleNode, targetNode},
	}

	for i, s := range partSearches {
		found, err := dummyPart.Search(s)

		Convey("Checking the Partition search functions", t, func() {
			Convey(fmt.Sprintf("running a search of %s", s), func() {
				Convey("No error is returned and the expected nodes are returned", func() {
					So(err, ShouldBeNil)
					So(found, ShouldResemble, partExpected[i])
				})
			})
		})
	}

	badPartSearches := []string{"selec * where UL = 060e2b34.027f0101.0d010101.01010f00",
		"select * from essence where unknownField = testfield",
		"select * from essence where UL <=> anInvalidField"}
	badPartError := []string{"first word not select", "unknown field \"unknownField\"", "unknown comparison operator \"<=>\""}

	for i, bs := range badPartSearches {
		found, err := dummyPart.Search(bs)

		Convey("Checking the partition search functions return the correct errors", t, func() {
			Convey(fmt.Sprintf("running a search of %s", bs), func() {
				Convey("An error is returned with no nodes", func() {
					So(err, ShouldResemble, fmt.Errorf(badPartError[i]))
					So(len(found), ShouldEqual, 0)
				})
			})
		})
	}

	dummyMXF := MXFNode{Partitions: []*PartitionNode{dummyPart, {}, {}, {}}}

	mxfSearches := []string{
		"select * from partition where type = " + string(Header),
		"select * from partition where metadata <> 0",
		"select * from partition where metadata <> 0 AND essence <> 0"}

	mxfExpected := [][]*PartitionNode{
		{dummyPart},
		{dummyPart},
		{dummyPart},
	}

	for i, s := range mxfSearches {
		found, err := dummyMXF.Search(s)

		Convey("Checking the mxf search functions", t, func() {
			Convey(fmt.Sprintf("running a search of %s", s), func() {
				Convey("No error is returned and the expected nodes are returned", func() {
					So(err, ShouldBeNil)
					So(found, ShouldResemble, mxfExpected[i])
				})
			})
		})
	}

	badMXFSearches := []string{"selec * where UL = 060e2b34.027f0101.0d010101.01010f00",
		"select * from partition where unknownField = testfield",
		"select * from partition where metadata <=> anInvalidField"}
	badMXFError := []string{"first word is not select, please use the correct formatting", "unknown field \"unknownField\"", "unknown comparison operator \"<=>\""}

	for i, bs := range badMXFSearches {
		found, err := dummyMXF.Search(bs)

		Convey("Checking the partition search functions return the correct errors", t, func() {
			Convey(fmt.Sprintf("running a search of %s", bs), func() {
				Convey("An error is returned with no nodes", func() {
					So(err, ShouldResemble, fmt.Errorf(badMXFError[i]))
					So(len(found), ShouldEqual, 0)
				})
			})
		})
	}
}

func TestDecodes(t *testing.T) {

	// set up dummy data to be encoded/decoded
	primer := mxf2go.NewPrimer()

	shortHandObject := mxf2go.GISXDStruct{
		ContainerFormat:  []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
		InstanceID:       [16]uint8{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
		NamespaceURIUTF8: []rune("example.com/test"),
		SampleRate:       mxf2go.TRational{Numerator: 1, Denominator: 24}}

	shortHandBytes, _ := shortHandObject.Encode(primer)

	// conver the priemr to the style used in this package
	decodePrimer := make(map[string]string)
	for full, short := range primer.Tags {
		decodePrimer[fmt.Sprintf("%02x%02x", short[0], short[1])] = FullNameMask([]byte(full))
	}

	fullNameObject := mxf2go.GBadRequestResponseStruct{ASMBadRequestCopy: []byte("some text as a space"), ASMResponse: 2}
	fullNameBytes, _ := fullNameObject.Encode()
	//	&Node{Key: Position{End: 16}, Length: Position{Start: 16, End: 17}, Value: Position{Start: 17, End: len(dd)}}

	// set up the byte streams to be decoded
	nodes := []*Node{
		{Key: Position{End: 16}, Length: Position{Start: 16, End: 17}, Value: Position{Start: 17, End: len(shortHandBytes)}},
		{Key: Position{End: 16}, Length: Position{Start: 16, End: 17}, Value: Position{Start: 17, End: len(fullNameBytes)}},
	}

	byteStreams := []io.ReadSeeker{
		bytes.NewReader(shortHandBytes),
		bytes.NewReader(fullNameBytes),
	}

	expectedOut := []map[string]any{
		{"ContainerFormat": mxf2go.TWeakReference{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}, "DataEssenceCoding": mxf2go.TAUID{Data1: 0, Data2: 0, Data3: 0, Data4: [8]uint8{0, 0, 0, 0, 0, 0, 0, 0}}, "InstanceID": mxf2go.TUUID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}, "NamespaceURIUTF8": "example.com/test", "SampleRate": mxf2go.TRational{Numerator: 1, Denominator: 24}},
		{"ASMBadRequestCopy": mxf2go.TDataValue{115, 111, 109, 101, 32, 116, 101, 120, 116, 32, 97, 115, 32, 97, 32, 115, 112, 97, 99, 101}, "ASMResponse": uint8(2)}}

	for i, byteStream := range byteStreams {
		out, err := DecodeGroupNode(byteStream, nodes[i], decodePrimer)

		Convey("Checking the node decoder can decode mxf byte streams into objects", t, func() {
			Convey("decoding the byte stream", func() {
				Convey("The object is decoded without error", func() {
					So(err, ShouldBeNil)
					So(out, ShouldResemble, expectedOut[i])
				})
			})
		})
	}

}
