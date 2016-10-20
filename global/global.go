package global

// This file defines accessible variables used to setup the caldav server.

import (
  "git.samedi.cc/ferraz/caldav/data"
)

// The global storage used in the CRUD operations of resources. Default storage is the `FileStorage`.
var Storage data.Storage = new(data.FileStorage)
// Current caldav user. It is used to keep the info of the current user that is interacting with the calendar.
var User *data.CalUser
