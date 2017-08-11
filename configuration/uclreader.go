// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package configuration

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	libucl "github.com/bitmark-inc/go-libucl"
	"os"
	"reflect"
	"strings"
)

// read a configuration file and parse using libucl
//
// note:
//   all environment variables are added as ENV_xxx to the lib ucl variables table
//
func ParseConfigurationFile(fileName string, config interface{}, variables map[string]string) error {

	// since interface{} is untyped, have to verify type compatibility at run-time
	rv := reflect.ValueOf(config)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fault.ErrInvalidStructPointer
	}

	// now sure item is a pointer, make sure it points to some kind of struct
	s := rv.Elem()
	if s.Kind() != reflect.Struct {
		return fault.ErrInvalidStructPointer
	}

	// create a libucl parser
	p := libucl.NewParser(0)
	defer p.Close()

	// keep all variable for checkdefined
	keep := make(map[string]string)

	// macro
	//   .set(var=NAME) "var1:var2:..."
	p.RegisterMacro("set", func(args libucl.Object, body string) bool {
		v := args.Get("var")
		if nil == v {
			return false
		}
		variable := v.ToString()
		if "" == variable {
			return false
		}
		if 0 == len(body) {
			return false
		}
		value := ""
	loop:
		for _, r := range strings.Split(body, ":") {
			v := keep[r]
			if "" != v {
				value = v
				break loop
			}
		}
		keep[variable] = value
		p.RegisterVariable(variable, value)
		return true
	})

	// macro
	//   .prepend(var=NAME) "value"
	p.RegisterMacro("prepend", func(args libucl.Object, body string) bool {
		v := args.Get("var")
		if nil == v {
			return false
		}
		variable := v.ToString()
		if "" == variable {
			return false
		}
		if 0 == len(body) {
			return false
		}
		value := keep[variable]
		if "" != value {
			value = body + value
		}
		keep[variable] = value
		p.RegisterVariable(variable, value)
		return true
	})

	// macro
	//   .prepend(var=NAME) "value"
	p.RegisterMacro("append", func(args libucl.Object, body string) bool {
		v := args.Get("var")
		if nil == v {
			return false
		}
		variable := v.ToString()
		if "" == variable {
			return false
		}
		if 0 == len(body) {
			return false
		}
		value := keep[variable]
		if "" != value {
			value = value + body
		}
		keep[variable] = value
		p.RegisterVariable(variable, value)
		return true
	})

	// macro
	//   .default(var=name) "<default-if-not-set>"
	p.RegisterMacro("default", func(args libucl.Object, body string) bool {
		v := args.Get("var")
		if nil == v {
			return false
		}
		variable := v.ToString()
		if "" == variable {
			return false
		}
		if v := keep[variable]; "" == v {
			keep[variable] = body
			p.RegisterVariable(variable, body)
		}
		return true
	})

	// set ENV_xxx as environment
scan_environment:
	for _, item := range os.Environ() {
		s := strings.SplitN(item, "=", 2) // expect key=value
		if 2 != len(s) {                  // value may contain more "="
			continue scan_environment
		}
		variable := strings.TrimSpace(s[0])
		if "" == variable {
			continue scan_environment
		}
		variable = "ENV_" + variable
		value := strings.TrimSpace(s[1])
		keep[variable] = value
		p.RegisterVariable(variable, value)
	}

	// add all the variables, can override ENV_xxx if required
	for variable, value := range variables {
		p.RegisterVariable(variable, value)
		keep[variable] = value
	}

	// add the master configuration file
	if err := p.AddFile(fileName); err != nil {
		return err
	}

	// fetch the root object
	rootObject := p.Object()
	defer rootObject.Close()

	// decode it into the callers struct
	if err := rootObject.Decode(config); err != nil {
		return err
	}

	return nil
}
