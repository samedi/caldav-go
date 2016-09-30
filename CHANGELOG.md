# CHANGELOG

v0.1.0
-----------
2016-09-23 Daniel Ferraz <d.ferrazm@gmail.com>

This version implements:

* Allow: "GET, HEAD, PUT, DELETE, OPTIONS, PROPFIND, REPORT"
* DAV: "1, 3, calendar-access"
* Also only handles the following components: `VCALENDAR`, `VEVENT`

Currently unsupported:

* Components `VTODO`, `VJOURNAL`, `VFREEBUSY`
* `VEVENT` recurrences
* Resource locking
* User authentication
