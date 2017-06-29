// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// +build !freebsd

package configuration

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/hashicorp/hcl"
	"reflect"
)

// read a configuration file and parse using libucl
func ParseConfigurationFile(fileName string, config interface{}) error {

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

	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	if err = hcl.Unmarshal(b, config); nil != err {
		return err
	}
	return nil
}
