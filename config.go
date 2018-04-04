package caldav

import (
	"github.com/samedi/caldav-go/data"
	"github.com/samedi/caldav-go/global"
)

func SetupStorage(stg data.Storage) {
	global.Storage = stg
}

func SetupUser(username string) {
	global.User = &data.CalUser{username}
}
