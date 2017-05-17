// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package configuration

import (
	"fmt"
	"os"
	"strings"
	"text/template"
)

func Save(filename string, configuration *Configuration) error {

	tempFile := filename + ".new"
	previousFile := filename + ".bk"

	os.Remove(tempFile)

	file, err := os.Create(tempFile)
	if nil != err {
		fmt.Printf("Create file fail: %s\n", err)
		return err
	}

	configurationTemplate := template.Must(template.New("config").Parse(configurationTemplate))
	err = configurationTemplate.Execute(file, configuration)
	if nil != err {
		fmt.Printf("write config file error: %s\n", err)
		return err
	}

	err = os.Remove(previousFile)
	if nil != err && !strings.Contains(err.Error(), "no such file") {
		return err
	}
	err = os.Rename(filename, previousFile)
	if nil != err && !strings.Contains(err.Error(), "no such file") {
		return err
	}
	err = os.Rename(tempFile, filename)
	if nil != err {
		return err
	}

	return nil
}
