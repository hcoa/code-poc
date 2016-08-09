package main

import (
	"bufio"
	"bytes"
	"io"
)

type skipTillReader struct {
	rdr   *bufio.Reader
	delim []byte
	found bool
}

func newSkipTillReader(reader io.Reader, delim []byte) *skipTillReader {
	return &skipTillReader{
		rdr:   bufio.NewReader(reader),
		delim: delim,
		found: false,
	}
}

func (str *skipTillReader) Read(p []byte) (n int, err error) {
	if str.found {
		return str.rdr.Read(p)
	} else {
		// search byte by byte for the delimiter
	outer:
		for {
			for i := range str.delim {
				var c byte
				c, err = str.rdr.ReadByte()
				if err != nil {
					return 0, err
				}
				// doens't match so start over
				if str.delim[i] != c {
					continue outer
				}
			}
			str.found = true
			// we read the delimiter so add it back
			str.rdr = bufio.NewReader(io.MultiReader(bytes.NewReader(str.delim), str.rdr))
			return str.Read(p)
		}
	}
}

type readTillReader struct {
	rdr   *bufio.Reader
	delim []byte
	found bool
}

func newReadTillReader(reader io.Reader, delim []byte) *readTillReader {
	return &readTillReader{
		rdr:   bufio.NewReader(reader),
		delim: delim,
		found: false,
	}
}

func (rtr *readTillReader) Read(p []byte) (n int, err error) {
	if rtr.found {
		return 0, io.EOF
	} else {
	outer:
		for n < len(p) {
			for i := range rtr.delim {
				var c byte
				c, err = rtr.rdr.ReadByte()
				if err != nil && n > 0 {
					return n, nil
				} else if err != nil {
					return n, err
				}
				p[n] = c
				n++
				if rtr.delim[i] != c {
					continue outer
				}
			}
			rtr.found = true
			break
		}
		if n == 0 {
			err = io.EOF
		}
		return n, err
	}
}
