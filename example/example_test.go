package example

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/mm-eng/mxftest"
	"gopkg.in/yaml.v3"
)

func TestAST(t *testing.T) {

	mxfToTest := []string{"../testdata/demoReports/goodISXD.mxf",
		"../testdata/demoReports/veryBadISXD.mxf", "../testdata/demoReports/badISXD.mxf"}
	for _, mxf := range mxfToTest {
		sc := mxftest.NewSniffContext()
		doc, docErr := os.Open(mxf)
		var buf bytes.Buffer
		mxftest.MRXTest(doc, &buf, ISXDSpecifications(sc))
		// @TODO do hash comparisons
		f, cre := os.Create(mxf + ".yaml")
		f.Write(buf.Bytes())
		//	fmt.Println(buf.String())

		Convey("Checking AST maps are consistent and are in the expected form", t, func() {
			Convey(fmt.Sprintf("generating an AST of %s, which is saved as a yaml", mxf), func() {
				Convey("The generated yaml matches the expected yaml", func() {

					So(docErr, ShouldBeNil)
					So(cre, ShouldBeNil)
					//			So(yamErr, ShouldBeNil)
					//			So(expecErr, ShouldBeNil)
					//			So(fmt.Sprintf("%x", htest.Sum(nil)), ShouldResemble, fmt.Sprintf("%x", hnormal.Sum(nil)))
				})
			})
		})
	}

}

func TestGPS(t *testing.T) {

	mxfToTest := []string{"./testdata/gpsdemo.mxf"}
	for _, mxf := range mxfToTest {
		sc := mxftest.NewSniffContext()
		doc, docErr := os.Open(mxf)
		var buf bytes.Buffer
		testErr := mxftest.MRXTest(doc, &buf, GPSSpecifications(sc))
		//	fmt.Println(buf.String())

		var report mxftest.Report
		repErr := yaml.Unmarshal(buf.Bytes(), &report)
		fmt.Println(buf.String())

		Convey("Checking AST maps are consistent and are in the expected form", t, func() {
			Convey(fmt.Sprintf("generating an AST of %s, which is saved as a yaml", mxf), func() {
				Convey("The generated yaml matches the expected yaml", func() {

					So(docErr, ShouldBeNil)
					So(testErr, ShouldBeNil)
					So(repErr, ShouldBeNil)
					So(report.TestPass, ShouldBeTrue)
					//			So(expecErr, ShouldBeNil)
					//			So(fmt.Sprintf("%x", htest.Sum(nil)), ShouldResemble, fmt.Sprintf("%x", hnormal.Sum(nil)))
				})
			})
		})
	}

}
