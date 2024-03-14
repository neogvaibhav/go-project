package utils

import (
	"os"
	"strings"
)

type FileReader interface {
	Readfile() (*string, error)
}

type processorImpl struct{ filePath string }

// NewRepository ...
func NewFileReader(arg string) FileReader {
	return &processorImpl{filePath: arg}
}

func (p *processorImpl) Readfile() (*string, error) {
	txt, err := os.ReadFile(p.filePath)
	if err != nil {
		return nil, err
	}

	var a []string
	for i, t := range txt {

		if i == (len(txt) - 1) {
			break
		}
		r := string(int(t) - 128)
		a = append(a, r)
	}
	csv := strings.Join(a, "")
	return &csv, nil
}
