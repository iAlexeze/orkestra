package main

import "github.com/ialexeze/orkestra/pkg/utils"

type Foo struct {
	Bar *bool
}

func NewFoo(bar *bool) *Foo {
	if bar == nil {
		bar = utils.BoolPtr(false)
	}
	return &Foo{Bar: bar}
}

func main() {
	// Test bool pointers
	foo := NewFoo(nil)
	println(*foo.Bar)
}