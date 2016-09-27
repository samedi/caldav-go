package caldav

// This file defines accessible variables used to setup the caldav server.

import (
  "git.samedi.cc/ferraz/caldav/data"
)

// default storage is the `FileStorage`, that deals with the resources as files from the File System.
var Storage data.Storage = new(data.FileStorage)
