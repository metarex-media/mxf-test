// package mxftest contains the test functions and interfaces
// for testing mxf/mrx files against their specifications.
// And for developing new tests for testing the mxf files.
package mxftest

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"

	"github.com/metarex-media/mrx-tool/klv"
	mxf2go "github.com/metarex-media/mxf-to-go"

	"golang.org/x/sync/errgroup"
)

// Node is a object in the abstact syntax tree
// it can be a child, a parent or both.
type Node struct {
	Key, Length, Value Position
	Properties         MXFProperty
	// talk through the children role with Bruce
	// but keep as this
	Tests       tests[Node]
	markerTests tests[Node]
	Children    []*Node
	Sniffs      map[string]*SniffResult `yaml:"-"`
}

// Nodes are the different nodes in the Abstract syntax tree
type Nodes interface {
	Node | PartitionNode | MXFNode
}

// Parent is for declaring the parent of a node
// without out giving full control of that Node.
type Parent interface {
	FlagFail() // a function that recursively calls the parents when a test is failed
}

// Flag fail sets the test pass to fail
// then calls the same on its parent.
func (n *Node) FlagFail() {

	n.Tests.TestStatus.Pass = false
	n.Tests.parent.FlagFail()

}

// Flag fail sets the test pass to fail
// then calls the same on its parent.
func (p *PartitionNode) FlagFail() {
	p.Tests.TestStatus.Pass = false
	p.Tests.parent.FlagFail()
}

// Flag fail sets the test pass to fail
func (m *MXFNode) FlagFail() {
	m.Tests.TestStatus.Pass = false
}

// tests are the tests to be run on a node
// and the results of the test
type tests[N Nodes] struct {
	parent          Parent                    `yaml:"-"`
	tests           []markedTest[N]           // []*func(doc io.ReadSeeker, header *N) func(t Test)
	testsWithPrimer []markedTestWithPrimer[N] // []*func(doc io.ReadSeeker, header *N, primer map[string]string) func(t Test)
	TestStatus      testStatus
}

type markedTest[N Nodes] struct {
	test    *func(doc io.ReadSeeker, header *N) func(t Test)
	runTest *bool
}

type markedTestWithPrimer[N Nodes] struct {
	test    *func(doc io.ReadSeeker, header *N, primer map[string]string) func(t Test)
	runTest *bool
}

// test status is a simple wrapper of the test outcome
type testStatus struct {
	Pass bool
}

// MXFNode is the parent node of the MXF file.
// it contains its partitions as children
// and the list of tests to run on the node.
type MXFNode struct {
	Partitions  []*PartitionNode
	Tests       tests[MXFNode]
	markerTests tests[MXFNode]
}

// PartitionNode is the node for every MXF partition/
// It contains the different types of content as different arrays
// of nodes.
type PartitionNode struct {
	Parent             *MXFNode `yaml:"-"`
	Key, Length, Value Position
	HeaderMetadata     []*Node
	Essence            []*Node
	IndexTable         *Node
	Props              PartitionProperties
	Tests              tests[PartitionNode]
	markerTests        tests[PartitionNode]
	PartitionPos       int
}

// Position contains the start and end position
// of the Node in the byte stream.
type Position struct {
	Start, End int
}

// MXFProperty contains the properties of and MXF object
type MXFProperty interface {
	// symbol returns the MXF UL associated with the node.
	// if there is one
	UL() string
	// ID returns the ID associated with the property
	ID() string
	// Returns the type of that node
	// e.g. essence, partition or the group type like Descriptivemetadata
	Label() []string
}

// EssenceProperties contains the properties of an essence object
type EssenceProperties struct {
	EssUL string
}

// ID returns the of the essence, it always returns ""
func (e EssenceProperties) ID() string {

	return ""
}

// Label returns the labels associated with the essence.
// it always returns []string{"essence"}
func (e EssenceProperties) Label() []string {

	return []string{"essence"}
}

// UL returns the Universal Label of the essence
func (e EssenceProperties) UL() string {
	return e.EssUL
}

// GroupProperties contains the properties of an group object
type GroupProperties struct {
	UUID           mxf2go.TUUID
	UniversalLabel string
	GroupLabel     []string
}

// ID returns the of the group, formatted as
// "00000000.00000000.00000000.00000000"
func (gp GroupProperties) ID() string {
	var fullUUID string
	for _, uid := range gp.UUID {
		fullUUID += fmt.Sprintf("%02x", uid)
	}
	return fullUUID
}

// UL returns the Universal Label of the group
func (gp GroupProperties) UL() string {
	return gp.UniversalLabel
}

// Label returns an labels associated with a group
func (gp GroupProperties) Label() []string {
	return gp.GroupLabel
}

// PartitionProperties contains the properties of a partition object
type PartitionProperties struct {
	PartitionCount int // the count of the partition along the MXF
	PartitionType  string
	Primer         map[string]string
	EssenceOrder   []string
}

// ID returns the ID associated with a partition,
// it always returns ""
func (p PartitionProperties) ID() string {

	return ""
}

// Label returns the labels associated with the partition.
// it always returns []string{"partition"}
func (p PartitionProperties) Label() []string {

	return []string{"partition"}
}

// Symbol returns the type of the partition
func (p PartitionProperties) Symbol() string {
	// fmt.Println(p.PartitionType)
	return p.PartitionType
}

/*
Search follows SQL syntax for nodes within a nodes children

e.g. select * where UL = 060e2b34.027f0101.0d010101.01010f00

"from" is not used as there are no tables within a node,
the node is a table unto itself.

Available fields are:

  - ul
  - sniff:{field name} - e.g. sniff:/root searches the sniff value of root

The search command is not case sensitive
*/
func (n Node) Search(searchfield string) ([]*Node, error) {

	command := strings.Split(searchfield, " ")

	if len(command) < 6 {
		return nil, fmt.Errorf("malformed command of %s expected \"select * where header = property\" as a minimum command", searchfield)
	}

	if strings.ToLower(command[0]) != "select" {
		return nil, fmt.Errorf("first word not select")
	}

	out := make([]*Node, 0)
	for _, search := range n.Children {
		founds, err := recurseSearch(search, command[3], command[4], command[5])
		if err != nil {
			return nil, err
		}
		// search through the children as well
		out = append(out, founds...)
	}

	return out, nil
}

/*
Search follows SQL syntax for nodes within a partition

e.g. select * from essence where UL <> 060e2b34.01020105.0e090502.017f017f

Available tables are:

  - essence
  - metadata

Available fields are:

  - ul
  - sniff:{field name} - e.g. sniff:/root searches the sniff value of root

The search command is not case sensitive
*/
func (p PartitionNode) Search(searchfield string) ([]*Node, error) {
	// lowercase as ULs are lower case when searching
	command := strings.Split(searchfield, " ")

	if len(command) < 4 {
		return nil, fmt.Errorf("malformed command of %s expected \"select * from field\" as a minimum command", searchfield)
	}

	if command[0] != "select" {
		return nil, fmt.Errorf("first word not select")
	}

	// worry about this later
	// if command[1] != "*"

	var searchFields []*Node
	switch command[3] {
	case strings.ToLower("essence"):
		searchFields = p.Essence
	case strings.ToLower("metadata"):
		searchFields = p.HeaderMetadata
	default:
		return nil, fmt.Errorf("invalid field of \"%s\"", command[3])
	}

	switch len(command) {
	case 4:
		return searchFields, nil
	case 8:
		// keep on trucking
	default:
		return nil, fmt.Errorf("malformed command of %s expected \"select * from field where x = y\" as a minimum command", searchfield)
	}

	out := make([]*Node, 0)
	for _, search := range searchFields {
		founds, err := recurseSearch(search, command[5], command[6], command[7])
		if err != nil {
			return nil, err
		}
		// search through the children as well
		out = append(out, founds...)
	}
	return out, nil
}

func recurseSearch(node *Node, field, equate, target string) ([]*Node, error) {

	if node == nil {
		return nil, nil
	}
	out := make([]*Node, 0)

	// search through the children as well
	var compareField string

	switch {
	case strings.ToLower(field) == "ul":

		compareField = node.Properties.UL()
	case strings.Contains(field, "sniff:"):

		if len(node.Sniffs) != 0 {
			if field, ok := node.Sniffs[field[6:]]; ok {
				compareField = field.Field
			}
		}

	default:

		return nil, fmt.Errorf("unknown field \"%v\"", field)
	}

	var pass bool
	switch equate {
	case "=":
		pass = (compareField == target)
	case "<>":

		pass = (compareField != target)
	default:
		return nil, fmt.Errorf("unknown comparison operator \"%v\"", equate)
	}

	if pass {
		out = append(out, node)
	}

	for _, child := range node.Children {
		founds, err := recurseSearch(child, field, equate, target)
		if err != nil {
			return nil, err
		}
		// search through the children as well
		out = append(out, founds...)
	}

	return out, nil
}

/*
Search follows SQL for finding things within a partition
e.g. select * from essence where partition <> header

Available tables are:

  - partition

Available fields are:

  - essence - the count of essence
  - type - the partition types
  - metadata - the count of metadata

Available operators are:

  - AND - all conditions have to be true

The search command is not case sensitive
*/
func (m MXFNode) Search(searchfield string) ([]*PartitionNode, error) {
	// lowercase as ULs are lower case when searching
	command := strings.Split(searchfield, " ")

	if len(command) < 4 {
		return nil, fmt.Errorf("malformed command of %s expected \"select * from field\" as a minimum command", searchfield)
	}

	if command[0] != "select" {
		return nil, fmt.Errorf("first word is not select, please use the correct formatting")
	}

	// worry about this later
	// if command[1] != "*"

	var searchFields []*PartitionNode
	switch strings.ToLower(command[3]) {
	case "partition", "partitions":
		searchFields = m.Partitions
	default:
		return nil, fmt.Errorf("invalid field of \"%s\"", command[3])
	}

	switch len(command) {
	case 4:
		return searchFields, nil

	default:
		if len(command)%4 != 0 {
			return nil, fmt.Errorf("malformed command of %s expected \"select * from field where x = y\" as a minimum command", searchfield)
		}
	}

	// break the command into several bits

	testFields := make([][4]string, len(command[4:])/4)

	// set the commands

	for i := 0; i < len(command[4:])/4; i++ {
		if i == 0 {
			testFields[i] = [4]string{"and", command[5], command[6], command[7]}
		} else {
			testFields[i] = [4]string{command[4+i*4], command[5+i*4], command[6+i*4], command[7+i*4]}
		}
	}

	out := make([]*PartitionNode, 0)
	for _, search := range searchFields {
		GlobalPass := true
		for _, tf := range testFields {
			pass, err := compare(search, tf[1], tf[2], tf[3])

			if err != nil {
				return nil, err
			}

			switch strings.ToLower(tf[0]) {
			case "and":
				GlobalPass = (GlobalPass && pass)
			default:
				return nil, fmt.Errorf("unknown comparison %v", tf[0])
			}

			// skip the search
			if !GlobalPass {
				break
			}

		}

		if GlobalPass {
			out = append(out, search)
		}
	}
	return out, nil
}

func compare(search *PartitionNode, field, operator, target string) (bool, error) {

	var compareField string

	switch strings.ToLower(field) {
	case "type":
		compareField = search.Props.PartitionType
	case "essence":
		compareField = fmt.Sprintf("%v", len(search.Essence))
	case "metadata":
		compareField = fmt.Sprintf("%v", len(search.HeaderMetadata))
	default:
		return false, fmt.Errorf("unknown field \"%v\"", field)
	}

	var pass bool
	switch operator {
	case "=":
		pass = (compareField == target)
	case "<>":
		pass = (compareField != target)
	default:
		return pass, fmt.Errorf("unknown comparison operator \"%v\"", operator)

	}

	return pass, nil
}

type refAndChild struct {
	child bool
	ref   [][]byte
}

// Make AST generates an Abstract Syntax Tree (AST) of an MXF file.
//
// As part of the AST tests are assigned to the nodes in the tree, these tests are
// declared as specifications.
func MakeAST(stream io.Reader, buffer chan *klv.KLV, size int, specs Specifications) (*MXFNode, error) { // wg *sync.WaitGroup, buffer chan packet, errChan chan error) {

	// use errs to handle errors while runnig concurrently
	errs, _ := errgroup.WithContext(context.Background())

	// initiate the klv stream
	errs.Go(func() error {
		return klv.StartKLVStream(stream, buffer, size)
	})

	mxf := &MXFNode{Partitions: make([]*PartitionNode, 0), Tests: tests[MXFNode]{TestStatus: testStatus{true}, tests: specs.MXF},
		markerTests: tests[MXFNode]{TestStatus: testStatus{true}, tests: specs.markerMXF}}
	var currentPartitionNode *PartitionNode
	// /	var currentPartition int
	var primer map[string]string

	errs.Go(func() error {

		defer func() {
			// this only runs when an error occurs to stop blocking
			_, klvOpen := <-buffer
			for klvOpen {
				_, klvOpen = <-buffer
			}

		}()

		// get the first bit of stream
		klvItem, klvOpen := <-buffer

		offset := 0
		var patternTally bool
		// handle each klv packet
		for klvOpen {

			// check if it is a partition key
			// if not its presumed to be essence

			if FullNameMask(klvItem.Key, 7, 13, 14) == "060e2b34.0205017f.0d010201.017f7f00" {
				if currentPartitionNode != nil {
					mxf.Partitions = append(mxf.Partitions, currentPartitionNode)
				}

				// extract the partition
				currentPartitionNode = extractPartition(klvItem, mxf, &patternTally, primer, specs, offset)

				// create a reference map for every node that is found
				refMap := make(map[*Node]refAndChild)
				offset += klvItem.TotalLength()

				partitionLayout := PartitionExtract(klvItem)

				metaByteCount := 0
				idMap := make(map[string]*Node) // assign the ids of the map
				for metaByteCount < int(partitionLayout.HeaderByteCount) {
					metadata, open := <-buffer

					if !open {
						return fmt.Errorf("error when using klv data klv stream interrupted")
					}
					// decode the essence here

					mdNode := &Node{
						Key:    Position{Start: offset, End: offset + len(metadata.Key)},
						Length: Position{Start: offset + len(metadata.Key), End: offset + len(metadata.Key) + len(metadata.Length)},
						Value:  Position{Start: offset + len(metadata.Key) + len(metadata.Length), End: offset + metadata.TotalLength()},
						Tests:  tests[Node]{TestStatus: testStatus{true}},
					}

					refMap[mdNode] = refAndChild{}

					// @TODO include KLV fill packets
					_, skip := decodeBuilder(metadata.Key[5])

					if skip {

						// unpack the primer

						if FullNameMask(metadata.Key, 5) == "060e2b34.027f0101.0d010201.01050100" {
							out := make(map[string]string)
							primerUnpack(metadata.Value, out)
							primer = out
							mdNode.Properties = GroupProperties{UniversalLabel: "060e2b34.027f0101.0d010201.01050100"}
							currentPartitionNode.Props.Primer = primer
						}
						// want to loop through them all?

					} else {
						// extract the metadata form the klv
						metadataNodeExtraction(metadata, mdNode, refMap, idMap, primer, specs)

						// "urn:smpte:ul:060e2b34.01010101.01011502.00000000"
					}

					offset += metadata.TotalLength()
					metaByteCount += metadata.TotalLength()

					// currentNode.Children = append(currentNode.Children, flushNode)

				}

				// thread the partition afterwards
				// first by finding the references
				// and marking if something is a child
				for n, refs := range refMap {
					for _, ref := range refs.ref {
						child := idMap[string(ref)]
						mid := refMap[child]
						mid.child = true
						refMap[child] = mid
						if child != nil {

							child.Tests.parent = n
						}
						n.Children = append(n.Children, child)
					}
				}

				// then by assigning all the parents
				for n, refs := range refMap {

					if !refs.child {
						n.Tests.parent = currentPartitionNode
						currentPartitionNode.HeaderMetadata = append(currentPartitionNode.HeaderMetadata, n)
					}
				}

				// order the map by appearance order
				slices.SortFunc(currentPartitionNode.HeaderMetadata, func(a, b *Node) int {
					return a.Key.Start - b.Key.Start
				})

				if partitionLayout.IndexTable {
					//	index table is after all the metadata
					index, open := <-buffer

					if !open {
						return fmt.Errorf("error parsing stream channel unexpectedly closed")
					}
					currentPartitionNode.IndexTable = &Node{
						Key:    Position{Start: offset, End: offset + len(index.Key)},
						Length: Position{Start: offset + len(index.Key), End: offset + len(index.Key) + len(index.Length)},
						Value:  Position{Start: offset + len(index.Key) + len(index.Length), End: offset + index.TotalLength()},
						Tests:  tests[Node]{TestStatus: testStatus{true}, parent: currentPartitionNode},
					}
					offset += index.TotalLength()

					//	fmt.Println(md.currentContainer.IndexTable)
				}

				//	currentPartitionNode.HeaderMetadata = append(currentPartitionNode.HeaderMetadata, currentPartitionNode)
			} else {

				if currentPartitionNode == nil {
					return fmt.Errorf("invalid mxf file structure, essence encountered before any partitions")
				}
				// extract the essence
				essNode := extractEssenceNode(klvItem, currentPartitionNode, offset, &patternTally)
				// sniff the data based on the specifications
				// n := time.Now()
				out := Sniff(klvItem.Value, specs.sniffTests)
				essNode.Sniffs = out

				currentPartitionNode.Essence = append(currentPartitionNode.Essence, essNode)
				offset += klvItem.TotalLength()
				// throw a warning here saying expected partition got KEY : fullname

			}

			// get the next item for a loop
			klvItem, klvOpen = <-buffer
		}

		if currentPartitionNode != nil {
			mxf.Partitions = append(mxf.Partitions, currentPartitionNode)
		}

		if offset == 0 {
			return fmt.Errorf("no mxf data found in byte stream")
		}
		return nil
	})

	// post processing data if the klv hasn't returned an error
	// count of partitions
	err := errs.Wait()
	if err != nil {
		return nil, err
	}

	// b, _ := yaml.Marshal(mxf)
	// dest.Write(b)
	// fmt.Println(mxf)
	// assign after the yaml to stop endless recursion
	for _, p := range mxf.Partitions {
		p.Parent = mxf
	}
	return mxf, nil
}

// extract the essence as a Node
func extractEssenceNode(klvItem *klv.KLV, currentPartitionNode *PartitionNode, offset int, patternTally *bool) *Node {
	name := fullName(klvItem.Key)
	_, ok := mxf2go.EssenceLookUp["urn:smpte:ul:"+name]

	if len(currentPartitionNode.Props.EssenceOrder) != 0 {
		if currentPartitionNode.Props.EssenceOrder[0] == name {
			*patternTally = false
		} else if *patternTally {
			currentPartitionNode.Props.EssenceOrder = append(currentPartitionNode.Props.EssenceOrder, name)
		}
	} else {
		currentPartitionNode.Props.EssenceOrder = append(currentPartitionNode.Props.EssenceOrder, name)
	}

	if !ok {
		// check for a 7f masked version at the final byte
		klvItem.Key[15] = 0x7f
		_, ok = mxf2go.EssenceLookUp["urn:smpte:ul:"+fullName(klvItem.Key)]
		if !ok {
			// check for a 7f masked version at the final byte and the 14th byte
			klvItem.Key[13] = 0x7f
			_, ok = mxf2go.EssenceLookUp["urn:smpte:ul:"+fullName(klvItem.Key)]
			if ok {
				name = fullName(klvItem.Key)
			}
		} else {
			name = fullName(klvItem.Key)
		}
	}

	// the output symbol is the name of the key

	return &Node{
		Key:        Position{Start: offset, End: offset + len(klvItem.Key)},
		Length:     Position{Start: offset + len(klvItem.Key), End: offset + len(klvItem.Key) + len(klvItem.Length)},
		Value:      Position{Start: offset + len(klvItem.Key) + len(klvItem.Length), End: offset + klvItem.TotalLength()},
		Properties: EssenceProperties{EssUL: name},
		Children:   make([]*Node, 0),
		Tests:      tests[Node]{TestStatus: testStatus{true}, parent: currentPartitionNode},
	}

}

func extractPartition(klvItem *klv.KLV, mxf *MXFNode, patternTally *bool, primer map[string]string, specs Specifications, offset int) *PartitionNode {
	partition := &PartitionNode{

		Key:            Position{Start: offset, End: offset + len(klvItem.Key)},
		Length:         Position{Start: offset + len(klvItem.Key), End: offset + len(klvItem.Key) + len(klvItem.Length)},
		Value:          Position{Start: offset + len(klvItem.Key) + len(klvItem.Length), End: offset + klvItem.TotalLength()},
		HeaderMetadata: make([]*Node, 0),
		Essence:        make([]*Node, 0),
		Parent:         mxf,
		Tests:          tests[PartitionNode]{TestStatus: testStatus{true}, parent: mxf},
		markerTests:    tests[PartitionNode]{},
		PartitionPos:   len(mxf.Partitions),
	}
	*patternTally = true

	// create a reference map for every node that is found

	// test the previous partitions essence as the final step
	// if len(contents.RipLayout) == 0 and the cache length !=0 emit an error that essence was found first

	partProps := PartitionProperties{PartitionCount: len(mxf.Partitions), EssenceOrder: make([]string, 0)}

	switch klvItem.Key[13] {
	case 17:
		partProps.PartitionType = RIPPartition
	case 02:
		// header
		partProps.PartitionType = HeaderPartition
		partition.Tests.tests = append(partition.Tests.tests, specs.Part[string(Header)]...)
		partition.markerTests.tests = append(partition.markerTests.tests, specs.markerPart[string(Header)]...)
	case 03:
		// body
		if klvItem.Key[14] == 17 {
			partProps.PartitionType = GenericStreamPartition
			partition.Tests.tests = append(partition.Tests.tests, specs.Part[string(GenericBody)]...)
			partition.markerTests.tests = append(partition.markerTests.tests, specs.markerPart[string(GenericBody)]...)
		} else {
			partProps.PartitionType = BodyPartition
			partition.Tests.tests = append(partition.Tests.tests, specs.Part[string(Body)]...)
			partition.markerTests.tests = append(partition.markerTests.tests, specs.markerPart[string(Body)]...)
		}
	case 04:
		// footer
		partProps.PartitionType = FooterPartition
		partition.Tests.tests = append(partition.Tests.tests, specs.Part[string(Header)]...)
		partition.markerTests.tests = append(partition.markerTests.tests, specs.markerPart[string(Header)]...)
	default:
		// is nothing
		partProps.PartitionType = "invalid"

	}
	// primer will get updated because of pointer magic
	partProps.Primer = primer
	partition.Props = partProps

	return partition
}

func metadataNodeExtraction(metadata *klv.KLV, mdNode *Node, refMap map[*Node]refAndChild, idMap map[string]*Node, primer map[string]string, specs Specifications) {

	dec, _ := decodeBuilder(metadata.Key[5])

	decoders, ok := mxf2go.Groups["urn:smpte:ul:"+fullName(metadata.Key)]

	if !ok {
		metadata.Key[5] = 0x7f
		decoders, ok = mxf2go.Groups["urn:smpte:ul:"+fullName(metadata.Key)]
	}
	if !ok {
		metadata.Key[13] = 0x7f
		decoders, ok = mxf2go.Groups["urn:smpte:ul:"+fullName(metadata.Key)]
	}

	// assign the generic name as the key
	key := fullName(metadata.Key)
	mdNode.Properties = GroupProperties{UniversalLabel: key}
	// find the groups first

	if ok {
		if nodeTests, ok := specs.Node[key]; ok {
			mdNode.Tests = tests[Node]{testsWithPrimer: nodeTests, TestStatus: testStatus{true}}
		}

		if nodeMarkerTests, ok := specs.markerNode[key]; ok {

			mdNode.markerTests = tests[Node]{testsWithPrimer: nodeMarkerTests, TestStatus: testStatus{true}}
		}
	}
	pos := 0
	for pos < len(metadata.Value) {
		key, klength := dec.keyFunc(metadata.Value[pos : pos+dec.keyLen])
		length, lenlength := dec.lengthFunc(metadata.Value[pos+dec.keyLen : pos+dec.keyLen+dec.lengthLen])
		if klength != 16 {
			key = primer[key]
		}

		// @TODO inlude the key for other AUIDs and ObjectIDs as part of the process
		switch key {
		// the instance ID key
		case "060e2b34.01010101.01011502.00000000":
			out, _ := mxf2go.DecodeTUUID(metadata.Value[pos+dec.keyLen+dec.lengthLen : pos+dec.keyLen+dec.lengthLen+length])
			mid := mdNode.Properties.(GroupProperties)
			mid.UUID = out.(mxf2go.TUUID)
			mdNode.Properties = mid
			UUID := out.(mxf2go.TUUID)
			idMap[string(UUID[:])] = mdNode

		default:

			if ok {
				// check the decoder for the field
				decodeF, ok := decoders.Group["urn:smpte:ul:"+key]

				if ok {

					b, _ := decodeF.Decode(metadata.Value[pos+dec.keyLen+dec.lengthLen : pos+dec.keyLen+dec.lengthLen+length])
					strongRefs := ReferenceExtract(b, StrongRef)
					if len(strongRefs) > 0 {
						mid := refMap[mdNode]
						mid.ref = append(mid.ref, strongRefs...)
						refMap[mdNode] = mid
					} else {
						weakRefs := ReferenceExtract(b, WeakRef)
						if len(weakRefs) != 0 {
							outString := make([]string, len(weakRefs))
							for i, wr := range weakRefs {
								outString[i] = fullName(wr)
							}

							mid := mdNode.Properties.(GroupProperties)
							mid.GroupLabel = outString
							mdNode.Properties = mid
						}
					}
				}
			}
		}
		pos += klength + length + lenlength
	}

	// "urn:smpte:ul:060e2b34.01010101.01011502.00000000"

}
func primerUnpack(input []byte, shorthand map[string]string) {

	order := binary.BigEndian
	count := order.Uint32(input[0:4])
	length := order.Uint32(input[4:8]) // if length isn't 18 explode

	offset := 8
	for i := uint32(0); i < count; i++ {
		// fmt.Printf("%x: %v\n", input[offset:offset+2], fullName(input[offset+2:offset+18]))
		short := fmt.Sprintf("%04x", input[offset:offset+2])
		shorthand[short] = fullName(input[offset+2 : offset+18])
		offset += int(length)
	}

}

func oneNameKL(namebytes []byte) (string, int) {
	if len(namebytes) != 1 {
		return "", 0
	}

	return fmt.Sprintf("%02x", namebytes[0:1:1]), 1
}

/*
	func oneLengthKL(lengthbytes []byte) (int, int) {
		if len(lengthbytes) != 1 {
			return 0, 0
		}

		return int(lengthbytes[0]), 1
	}
*/
func twoNameKL(namebytes []byte) (string, int) {
	if len(namebytes) != 2 {
		return "", 0
	}

	return fmt.Sprintf("%04x", namebytes[0:2:2]), 2
}

func twoLengthKL(lengthbytes []byte) (int, int) {
	if len(lengthbytes) != 2 {
		return 0, 0
	}

	length := order.Uint16(lengthbytes[0:2:2])

	return int(length), 2
}

func fullNameKL(namebytes []byte) (string, int) {

	if len(namebytes) != 16 {
		return "", 0
	}

	return fmt.Sprintf("%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x%02x",
		namebytes[0], namebytes[1], namebytes[2], namebytes[3], namebytes[4], namebytes[5], namebytes[6], namebytes[7],
		namebytes[8], namebytes[9], namebytes[10], namebytes[11], namebytes[12], namebytes[13], namebytes[14], namebytes[15]), 16
}

type keyLength struct {
	keyLen, lengthLen int
	lengthFunc        func([]byte) (int, int)
	keyFunc           func([]byte) (string, int)
}

// decodeBuilder generates the options to decode a packet.
// some tags need to be updated
func decodeBuilder(key uint8) (keyLength, bool) {
	var decodeOption keyLength
	var skip bool
	lenField := (key >> 4)
	keyField := (key & 0b00001111)

	// smpte 336 decode methods
	switch lenField {
	case 0, 1:
		decodeOption.lengthLen = 8
		decodeOption.lengthFunc = klv.BerDecode
	case 4, 5:
		decodeOption.lengthLen = 2
		decodeOption.lengthFunc = twoLengthKL
	default:
		skip = true
	}

	switch lenField%2 + keyField {
	case 0, 1, 2, 0xB:
		decodeOption.keyFunc = fullNameKL
		decodeOption.keyLen = 16
	case 4:
		decodeOption.keyFunc = twoNameKL
		decodeOption.keyLen = 2
	case 3:
		decodeOption.keyFunc = oneNameKL
		decodeOption.keyLen = 1
	case 0xC:
		// 3 is 1 byte
		// 0xB is ASN1
		// 0xC is 4
	default:
		skip = true
	}

	return decodeOption, skip
}

// Ref is the type for identifying reference types
type Ref string

const (
	// StrongRef is the reference type for strong references
	StrongRef Ref = "StrongReference"
	// WeakRef is the reference type for weak references
	WeakRef Ref = "WeakReference"
)

// ReferenceExtract extracts all references of a given type from a field.
// It returns each reference as the bytes.
// It looks at the type of the field and checks its name for
// the <reftype>Reference fields,
// if the field is not a reference object, then no references are returned.
//
// This is for use with the github.com/metarex-media/mxf-to-go
// repository as it contains a large number of variables that are
// references or contain nested references. A switch statement would be
// unfeasible for every type and this is currently the next best thing.
func ReferenceExtract(field any, reftype Ref) [][]byte {

	switch v := field.(type) {
	case mxf2go.TStrongReference:
		return [][]byte{v}
	default:
		switch {
		case strings.Contains(reflect.TypeOf(field).Name(), string(reftype)+"Set") || strings.Contains(reflect.TypeOf(field).Name(), string(reftype)+"Vector"):
			arr := reflect.ValueOf(field)
			arrLen := arr.Len()
			referenced := make([][]byte, arrLen)

			for i := 0; i < arrLen; i++ {

				// id, _ := yaml.Marshal(arr.Index(i).Interface())

				arrField := arr.Index(i).Interface()
				//	_, ok := idmap[strid]
				// fmt.Println(strid, ok, []byte(strid))
				// the midmap ensures the preservation of the object order
				// result := StrongReference(idmap[string(id)].mapper, idmap)
				result := ReferenceExtract(arrField, reftype)
				referenced[i] = result[0]
			}

			return referenced
		case strings.Contains(reflect.TypeOf(field).Name(), string(reftype)):
			return [][]byte{getId(v)}
		default:

			return [][]byte{}
		}
	}

}

// this just makes all the ids on the same page for when the ids are being added / read
func getId(ref any) []byte {
	arr := reflect.ValueOf(ref)
	arrLen := arr.Len()

	UID := make([]byte, arrLen)
	for i := 0; i < arrLen; i++ {
		UID[i] = arr.Index(i).Interface().(uint8)
	}

	return UID
}

// DecodeGroupNode decodes a Node into a map[string]any,
// where the key of the map is the name of the field and the any is the decoded value.
// Any unknown fields will not be decoded and are skipped from the returned values.
//
// The primer is a map of map[shorthandKey]fullUL
func DecodeGroupNode(doc io.ReadSeeker, node *Node, primer map[string]string) (map[string]any, error) {
	groupKLV, err := NodeToKLV(doc, node)
	if err != nil {
		return nil, err
	}

	return DecodeGroup(groupKLV, primer)
}

// DecodeGroup decodes a group KLV into a map[string]any,
// where the key of the map is the name of the field and the any is the decoded value.
// Any unknown fields will not be decoded and are skipped from the returned values.
//
// The primer is a map of map[shorthandKey]fullUL
func DecodeGroup(group *klv.KLV, primer map[string]string) (map[string]any, error) {
	dec, skip := decodeBuilder(group.Key[5])

	if skip {
		return nil, fmt.Errorf("unable to decode essence, unknown decode method byte %0x", group.Key[5])
	}

	decoders, ok := mxf2go.Groups["urn:smpte:ul:"+fullName(group.Key)]

	if !ok {
		decoders, ok = mxf2go.Groups["urn:smpte:ul:"+FullNameMask(group.Key, 5)]
	}
	if !ok {
		decoders, ok = mxf2go.Groups["urn:smpte:ul:"+FullNameMask(group.Key, 5, 13)]
	}

	if !ok {
		return nil, fmt.Errorf("no group for the key %s was found", fullName(group.Key))
	}

	output := make(map[string]any)
	pos := 0

	for pos < len(group.Value) {
		key, klength := dec.keyFunc(group.Value[pos : pos+dec.keyLen])

		end := pos + dec.keyLen + dec.lengthLen
		if end > len(group.Value) {
			end = len(group.Value)
		}
		length, lenlength := dec.lengthFunc(group.Value[pos+dec.keyLen : end])
		if klength != 16 {
			key = primer[key]
		}

		decodeF, ok := decoders.Group["urn:smpte:ul:"+key]

		if ok {
			b, _ := decodeF.Decode(group.Value[pos+dec.keyLen+lenlength : pos+dec.keyLen+lenlength+length])
			output[decodeF.UL] = b
		}

		pos += klength + length + lenlength
	}

	return output, nil
}

// NodeToKLV converts a node to a KLV object
func NodeToKLV(stream io.ReadSeeker, node *Node) (*klv.KLV, error) {
	stream.Seek(int64(node.Key.Start), 0)
	key := make([]byte, node.Key.End-node.Key.Start)
	leng := make([]byte, node.Length.End-node.Length.Start)
	val := make([]byte, node.Value.End-node.Value.Start)

	_, err := stream.Read(key)
	if err != nil {
		return nil, err
	}

	_, err = stream.Read(leng)
	if err != nil {
		return nil, err
	}

	_, err = stream.Read(val)
	if err != nil {
		return nil, err
	}

	return &klv.KLV{Key: key, Length: leng, Value: val}, nil
}
