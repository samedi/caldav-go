package caldav

import (
  "git.samedi.cc/ferraz/caldav/data"
  "git.samedi.cc/ferraz/caldav/global"
)

func SetupStorage(stg data.Storage) {
  global.Storage = stg
}