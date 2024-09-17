package xmlhandle

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	mxftest "github.com/metarex-media/mxf-test"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	goodFiles = []string{"./testdata/goodxml/cd_catalog.xml", "./testdata/goodxml/simple.xml", "./testdata/goodxml/ttml.xml"}
	badFiles  = []string{"./testdata/badxml/bad.json", "./testdata/badxml/note_error.xml", "./testdata/badxml/simple.xml"}
)

func TestIdentifier(t *testing.T) {

	for _, f := range goodFiles {

		path, _ := filepath.Abs(f)
		xmlBytes, readErr := os.ReadFile(path)

		pass := xMLIdentifier(xmlBytes)

		Convey("Checking the xml identifies xml files correctly", t, func() {
			Convey(fmt.Sprintf("Checking %s bytes", path), func() {
				Convey("The xml is identified as xml", func() {
					So(readErr, ShouldBeNil)
					So(pass, ShouldBeTrue)
				})
			})
		})
	}

	for _, f := range badFiles {
		path, _ := filepath.Abs(f)
		xmlBytes, readErr := os.ReadFile(path)

		pass := xMLIdentifier(xmlBytes)

		Convey("Checking the xml identifies xml files correctly", t, func() {
			Convey(fmt.Sprintf("Checking %s bytes", path), func() {
				Convey("The bad file is not identified as xml", func() {
					So(readErr, ShouldBeNil)
					So(pass, ShouldBeFalse)
				})
			})
		})
	}

}

func TestSniffPath(t *testing.T) {

	// check the duplication
	sc := mxftest.NewSniffContext()

	expected := PathSniffer(sc, "/")
	repeat := PathSniffer(sc, "/")
	Convey("Checking the xml sniffer returns a pointer to the same function instead of making a new one", t, func() {
		Convey("creating two functions with the exact same input parameters", func() {
			Convey("The functions returned are identical", func() {
				So(expected, ShouldResemble, repeat)
			})
		})
	})

	nameSpaceFinder := PathSniffer(sc, "namespace-uri(/*)")
	nsf := *nameSpaceFinder

	expectedNameSpace := []string{"", "example.com", "http://www.w3.org/ns/ttml"}

	expectedRoot := []string{"CATALOG", "breakfast_menu", "tt"}

	for i, f := range goodFiles {
		path, _ := filepath.Abs(f)
		xmlBytes, readErr := os.ReadFile(path)

		res := nsf(xmlBytes)

		Convey("Checking the xml name space sniffer works", t, func() {
			Convey(fmt.Sprintf("searching with the key of namespace-uri(/*) within %s", f), func() {
				Convey(fmt.Sprintf("A namespace of \"%s\" is found", expectedNameSpace[i]), func() {
					So(readErr, ShouldBeNil)
					So(res.Field, ShouldResemble, expectedNameSpace[i])
				})
			})
		})

		rootFinder := PathSniffer(sc, "/*")
		rf := *rootFinder

		// check the root finder
		// any other xpath is on the user
		rootRes := rf(xmlBytes)

		Convey("Checking the xml xpath sniffer works", t, func() {
			Convey(fmt.Sprintf("searching with the key of /* within %s", f), func() {
				Convey(fmt.Sprintf("A root of \"%s\" is found", expectedRoot[i]), func() {
					So(rootRes.Field, ShouldResemble, expectedRoot[i])
				})
			})
		})

	}

	testF := "./testdata/goodxml/ttml.xml"
	xmlBytes, _ := os.ReadFile(testF)
	commands := []string{"/tt/@xml:lang", "/*", "/tt/head/metadata/ttm:title"}
	outs := []string{"en", "tt", "title"}

	for i, c := range commands {

		sniff := PathSniffer(sc, c)
		sf := *sniff

		// check the root finder
		// any other xpath is on the user
		rootRes := sf(xmlBytes)

		Convey("Checking the xml name xpath sniffer works", t, func() {
			Convey(fmt.Sprintf("searching with the key of %s within %s", c, testF), func() {
				Convey(fmt.Sprintf("A result of \"%s\" is found", outs[i]), func() {
					So(rootRes.Field, ShouldResemble, outs[i])
				})
			})
		})

	}

}
