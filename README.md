# fixedwidth [![GoDoc](https://godoc.org/github.com/ianlopshire/go-fixedwidth?status.svg)](http://godoc.org/github.com/ianlopshire/go-fixedwidth) [![Report card](https://goreportcard.com/badge/github.com/ianlopshire/go-fixedwidth)](https://goreportcard.com/report/github.com/ianlopshire/go-fixedwidth)

Package fixedwidth provides encoding and decoding for fixed-width formatted Data.

`go get github.com/ianlopshire/go-fixedwidth`

## Decode
```go
// Define some fixed-with data to parse
data := []byte("" +
    "1         Ian                 Lopshire" + "\n" +
    "2         John                Doe" + "\n" +
    "3         Jane                Doe" + "\n")

// Define the format as a struct.
// The fixed start and end position are defined via struct tags: `fixed:"{startPos},{endPos}"`.
// Positions start at 1. The interval is inclusive.
var people []struct {
    ID        int    `fixed:"1,10"`
    FirstName string `fixed:"11,30"`
    LastName  string `fixed:"31,50"`
}

err := fixedwidth.Unmarshal(data, &people)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("%+v\n", people[0])
fmt.Printf("%+v\n", people[1])
fmt.Printf("%+v\n", people[2])
// Output:
// {ID:1 FirstName:Ian LastName:Lopshire}
// {ID:2 FirstName:John LastName:Doe}
// {ID:3 FirstName:Jane LastName:Doe}
```

## Licence
MIT