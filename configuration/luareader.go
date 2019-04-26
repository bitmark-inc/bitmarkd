package configuration

import (
	"github.com/yuin/gluamapper"
	lua "github.com/yuin/gopher-lua"
)

// ParseConfigurationFile - read and execute a Lua files and assign
// the results to a configuration structure
func ParseConfigurationFile(fileName string, config interface{}) error {
	L := lua.NewState()
	defer L.Close()

	L.OpenLibs()

	// create the global "arg" table
	// arg[0] = config file
	arg := &lua.LTable{}
	arg.Insert(0, lua.LString(fileName))
	L.SetGlobal("arg", arg)

	// execute configuration
	if err := L.DoFile(fileName); err != nil {
		return err
	}

	mapperOption := gluamapper.Option{
		NameFunc: func(s string) string {
			return s
		},
		TagName: "gluamapper",
	}
	mapper := gluamapper.Mapper{Option: mapperOption}
	if err := mapper.Map(L.Get(L.GetTop()).(*lua.LTable), config); err != nil {
		return err
	}

	return nil
}
