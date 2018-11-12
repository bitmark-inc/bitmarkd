package configuration

import (
	"github.com/yuin/gluamapper"
	lua "github.com/yuin/gopher-lua"
)

func ParseConfigurationFile(fileName string, config interface{}) error {
	L := lua.NewState()
	defer L.Close()

	L.OpenLibs()

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
