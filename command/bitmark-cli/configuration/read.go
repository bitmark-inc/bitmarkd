// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package configuration

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	libucl "github.com/bitmark-inc/go-libucl"
	"reflect"
)

// read a configuration file and parse using libucl
func readConfigurationFile(fileName string, config interface{}) error {

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
