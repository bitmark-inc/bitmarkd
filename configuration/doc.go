// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// parse a ucl configuration file
//
// The configuration supports variable expansion:
//
//   ${VAR}             variable provided bu optional map
//   ${ENV_VAR}         all environment variables are prefix with ENV_
//
// However, if the VAR is undefined then the text is not expanded and
// remains as "${VAR}" which is normally not wanted.  To cope with
// this several macros are defined:
//
// Set a variable to the first non-blank variable; its existing value
// ia overwritten.  If all variables are undefined then set it to the
// empty string.  Include the variable itself at an apropriate
// position if you wish to retain its existing value
//
//   .set(var=NAME) "var1:var2:...:varN"
//   .set(var=NAME) "var1:var2:...:varN:NAME"
//   .set(var=NAME) "NAME:var1:var2:...:varN"
//
// Prepend text to an non-blank variable.  If the variable was
// undefined it will be set to blank.
//
//   .prepend(var-NAME) "some text"
//
// Append text to an non-blank variable.  If the variable was
// undefined it will be set to blank.
//
//   .append(var-NAME) "some text"
//
// Set an undefined or blank variable to a default value.  The text can be empty just to ensure that undefined variables are converted to empty string.
//
//   .default(var=NAME) "some text"
//
package configuration
