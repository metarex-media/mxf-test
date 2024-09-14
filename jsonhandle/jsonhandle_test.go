package jsonhandle

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/mm-eng/mxftest"
)

var (
	goodFiles = []string{"./testdata/goodjson/array.json", "./testdata/goodjson/good.json", "./testdata/goodjson/gps.json"}
	badFiles  = []string{"./testdata/badjson/bad.json", "./testdata/badjson/note.xml", "./testdata/badjson/simple.yaml"}
)

func TestIdentifier(t *testing.T) {

	for _, f := range goodFiles {

		path, _ := filepath.Abs(f)
		jsBytes, readErr := os.ReadFile(path)

		pass := jSONIdentifier(jsBytes)

		Convey("Checking the json identifies json files correctly", t, func() {
			Convey(fmt.Sprintf("Checking %s bytes", path), func() {
				Convey("The json is identified as json", func() {
					So(readErr, ShouldBeNil)
					So(pass, ShouldBeTrue)
				})
			})
		})
	}

	for _, f := range badFiles {
		path, _ := filepath.Abs(f)
		jsonBytes, readErr := os.ReadFile(path)

		pass := jSONIdentifier(jsonBytes)

		Convey("Checking the json identifies json files correctly", t, func() {
			Convey(fmt.Sprintf("Checking %s bytes", path), func() {
				Convey("The bad file is not identified as json", func() {
					So(readErr, ShouldBeNil)
					So(pass, ShouldBeFalse)
				})
			})
		})
	}

}

func TestSchemas(t *testing.T) {
	files := []string{"./testdata/goodjson/array.json", "./testdata/goodjson/gps.json"}
	schemas := []string{"./testdata/schemas/gpsArraySchema.json", "./testdata/schemas/gpsSchema.json"}

	for i, f := range files {
		sc := mxftest.NewSniffContext()
		path, _ := filepath.Abs(f)
		jsonBytes, readErr := os.ReadFile(path)

		schema, schemaErr := os.ReadFile(schemas[i])

		sch, schErr := SchemaCheck(sc, schema, "test")

		// this is to prevent panics later on when the object
		// is referenced
		if sch == nil {
			Convey("Checking the json schema parser runs", t, func() {
				Convey(fmt.Sprintf("Checking %s bytes", path), func() {
					Convey("A schema check function is returned", func() {
						So(readErr, ShouldBeNil)
						So(schemaErr, ShouldBeNil)
						So(schErr, ShouldBeNil)
						So(sch, ShouldNotBeNil)
					})
				})
			})
			continue
		}

		valid := *sch

		res := valid(jsonBytes)

		Convey("Checking the json schema parser runs", t, func() {
			Convey(fmt.Sprintf("Checking %s bytes", path), func() {
				Convey("A schema check function is returned, which validates the json bytes", func() {
					So(readErr, ShouldBeNil)
					So(schemaErr, ShouldBeNil)
					So(res, ShouldResemble, mxftest.SniffResult{Key: "test", Field: "pass", Data: mxftest.CType(""), Certainty: 100})
				})
			})
		})
	}
}
