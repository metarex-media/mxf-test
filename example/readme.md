# MXF Test Demo Replit

This repl is a demo repl for giving an interactive playground to run
demos from the [mxftest][git] repo. Giving examples for the different types of MXF testing available.

It runs the following demos:

- [Data Validation Tests](./datavalidate.go)
- [File Structure tests](./structure.go)
- [File Metadata Validation](./filemetadata.go)
- [A complete ISXD specification test](./isxd.go)

## Running the demo

Please make sure you have read the [mxftest][git] repo and are comfortable with MXF.

Before running any code make sure to clone this Repo on replit using the fork, you will then be able to change the code to your whims.

Make sure you have the latest version of Go from the [official golang
source][g1] installed.

Run the demo with the following commands

```cmd
go run .
```

You should get the following output:

```cmd
successfully generated  exampleFiles/goodISXD.mxf-struct.yml
successfully generated  exampleFiles/goodISXD.mxf-node.yml
successfully generated  ./exampleFiles/gpsdemo.mxf.yml
successfully generated  ./exampleFiles/badISXD.mxf.yml
successfully generated  ./exampleFiles/goodISXD.mxf.yml
successfully generated  ./exampleFiles/veryBadISXD.mxf.yml
```

Check out the yamls generated to see the test results and how they worked.

Then you can start changing the tests and seeing how the results alter, some suggestions are listed below:

- Try changing the [gps schema](./exampleFiles/gpsSchema.json) and running it again, does the gps demo still pass?
- Update the gps Specification with the tests from other specifications, with the following lines

```go
mxftest.WithNodeTests(mxftest.NodeTest{UL:mxf2go.GISXDUL[13:],Test: nodeISXDDescriptor}),
mxftest.WithStructureTests(checkGPStructure),
```

Does the gps demo file still pass.

- Write any extra custom tests for any that deliberately pass or fail, then check the expected outcomes in the report.
- if you have any mxf files, try plugging them into the tests in this repl see if they pass.

[g1]:   https://go.dev/doc/install      "Golang Installation"
[git]: https://github.com/metarex-media/mxf-test "The git repo that this repl is testing"
