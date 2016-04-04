package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
)

const gzipstart = 0x4043

func assert(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	// Skip the header
	io.CopyN(ioutil.Discard, os.Stdin, gzipstart)

	reader, err := gzip.NewReader(os.Stdin)
	assert(err)

	data, err := ioutil.ReadAll(reader)
	assert(err)

	csdata := extractCS(data)

	var keys []int
	for k := range csdata {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, k := range keys {
		fmt.Printf("%d = %s\n", k, csdata[k])
	}
}

func extractCS(data []byte) map[int]string {
	out := make(map[int]string)
	// "..cs " (with first 2 bytes being non-ascii)
	cssep := []byte{0x00, 0x0B, 0x63, 0x73, 0x20}

	csdata := data[:]
	isend := false
	for {
		// Get next delimiter
		idx := bytes.Index(csdata, cssep)
		if idx < 0 {
			break
		}
		newidx := bytes.Index(csdata[idx+1:], cssep)
		if newidx < 0 {
			newidx = bytes.IndexByte(csdata[idx+1:], 0x00)
			isend = true
		}

		// Get current pair
		curdata := csdata[idx+5 : idx+1+newidx]

		// Split, separator is 0x20 (whitespace)
		curdatasep := bytes.IndexByte(curdata, 0x20)

		// Get pair
		curkey := curdata[:curdatasep]
		curval := curdata[curdatasep+1:]

		// Parse key as int
		intkey, err := strconv.Atoi(string(curkey))
		assert(err)

		// Put pair in map (trimming the "" around the value)
		out[intkey] = strings.Trim(string(curval), "\"")

		if isend {
			break
		}
		csdata = csdata[idx+1:]
	}

	return out
}
