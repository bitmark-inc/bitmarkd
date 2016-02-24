// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// command-line options processing
//
// Parses options of the forms:
//   -option               - increment otion count
//   -option=value         - set value
//   --option              - increment option count
//   --option=value        - set value
//   --                    - stop option parsing
//
// Note:
//   Does not support combined single letter options e.g. -vvv is the same as --vvv
//   and is the separate option "vvv".
//   Repeated options cause the value string to be appended to the options map item.
//   Options with no value append the empty string, e.g. "-v -v -v" would make len(options["v"]) == 3.
//
// Alias table:
//   This allows the option to be aliased e.g. -v -> --verbose
//
// Returns:
//   program_name          - string
//   options               - map["option"]=[]string{"value1","value2"}
//   arguments             - []string  (all options not starting with "-" and everything after --)
package getoptions
