package build

import (
	classyversion "go.szostok.io/version"
)

func Version() string {
	v := classyversion.Get()

	jsonVer, err := v.MarshalJSON()
	if err != nil {
		panic(err)
	}

	return v.Version + " " + string(jsonVer)
}
