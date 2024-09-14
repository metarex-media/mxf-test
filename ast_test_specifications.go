package mxftest

import (
	"fmt"
	"io"

	mxf2go "github.com/metarex-media/mxf-to-go"
	. "github.com/onsi/gomega"
)

// this package is here to stop convey and gomega clashes

// make a node tests that will pass or fail
func makeNodeTest(pass bool) NodeTest {
	return NodeTest{UL: mxf2go.GContentStorageUL[13:], Test: func(_ io.ReadSeeker, _ *Node, primer map[string]string) func(t Test) {
		return func(t Test) {
			t.Test(fmt.Sprintf("A demo Node test with an expected outcome of pass:%v", pass), NewSpecificationDetails("A demo specification", "XX", "shall", 1),
				t.Expect(pass).Shall(BeTrue()),
			)
		}
	}}
}

func makePartitionTest(pass bool) PartitionTest {
	return PartitionTest{PartitionType: Header, Test: func(_ io.ReadSeeker, _ *PartitionNode) func(t Test) {
		return func(t Test) {
			t.Test(fmt.Sprintf("A demo partition test with an expected outcome of pass:%v", pass), NewSpecificationDetails("A demo specification", "XX", "shall", 1),
				t.Expect(pass).Shall(BeTrue()),
			)
		}
	}}
}

func makeStructureTest(pass bool) func(doc io.ReadSeeker, mxf *MXFNode) func(t Test) {
	return func(doc io.ReadSeeker, mxf *MXFNode) func(t Test) {
		return func(t Test) {
			t.Test(fmt.Sprintf("A demo structure test with an expected outcome of pass:%v", pass), NewSpecificationDetails("A demo specification", "XX", "shall", 1),
				t.Expect(pass).Shall(BeTrue()),
			)
		}
	}
}
