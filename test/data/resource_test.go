package test

import (
  "testing"
  "time"
  "caldav/data"
)

func TestNewResource(t *testing.T) {
  res := data.NewResource("/foo///bar/123.ics//", FakeResourceAdapter{})

  if res.Name != "123.ics" {
    t.Error("Expected name to be 123.ics, got", res.Name)
  }

  // it cleans (sanitize) the path
  if res.Path != "/foo/bar/123.ics" {
    t.Error("Expected name to be /foo/bar/123.ics, got", res.Path)
  }
}

func TestIsCollection(t *testing.T) {
  adp := new(FakeResourceAdapter)
  res := data.NewResource("/foo/bar/", adp)

  adp.collection = false
  if res.IsCollection() {
    t.Error("Resource should not be a collection")
  }

  adp.collection = true
  if !res.IsCollection() {
    t.Error("Resource should be a collection")
  }
}

func TestIsPrincipal(t *testing.T) {
  res := data.NewResource("/foo", FakeResourceAdapter{})
  if !res.IsPrincipal() {
    t.Error("Resource should be principal")
  }

  res = data.NewResource("/foo/bar", FakeResourceAdapter{})
  if res.IsPrincipal() {
    t.Error("Resource should not be principal")
  }
}

func TestComponentName(t *testing.T) {
  adp := new(FakeResourceAdapter)
  res := data.NewResource("/foo", adp)

  adp.collection = false
  if res.ComponentName() != "VEVENT" {
    t.Error("Resource should be a VEVENT")
  }

  adp.collection = true
  if res.ComponentName() != "VCALENDAR" {
    t.Error("Resource should be a VCALENDAR")
  }
}

func TestEtag(t *testing.T) {
  adp := new(FakeResourceAdapter)
  res := data.NewResource("/foo", adp)

  adp.collection = false
  adp.etag = "1111"
  etag, found := res.GetEtag()
  if etag != "1111" || !found {
    t.Error("Etag should be 1111")
  }

  adp.etag = "2222"
  etag, found = res.GetEtag()
  if etag != "2222" || !found {
    t.Error("Etag should be 2222")
  }

  adp.collection = true
  etag, found = res.GetEtag()
  if etag != "" || found {
    t.Error("Collections should not have etags associated")
  }
}

func TestContentType(t *testing.T) {
  adp := new(FakeResourceAdapter)
  res := data.NewResource("/foo", adp)

  adp.collection = false
  ctype, found := res.GetContentType()
  if ctype != "text/calendar; component=vcalendar" || !found {
    t.Error("Content Type should be `text/calendar; component=vcalendar`")
  }

  adp.collection = true
  ctype, found = res.GetContentType()
  if ctype != "text/calendar" || !found {
    t.Error("Content Type should be `text/calendar`")
  }
}

func TestDisplayName(t *testing.T) {
  res := data.NewResource("foo/bar", FakeResourceAdapter{})

  // it just returns the resource Name
  name, found := res.GetDisplayName()
  if name != res.Name || !found {
    t.Error("Display name should be", res.Name)
  }
}

func TestContentData(t *testing.T) {
  adp := new(FakeResourceAdapter)
  res := data.NewResource("/foo", adp)

  adp.contentData = "EVENT;"

  adp.collection = true
  data, found := res.GetContentData()
  if data != "" || found {
    t.Error("Content data should be empty for collections")
  }

  adp.collection = false
  data, found = res.GetContentData()
  if data != "EVENT;" || !found {
    t.Error("Content data should be EVENT;")
  }
}

func TestContentLength(t *testing.T) {
  adp := new(FakeResourceAdapter)
  res := data.NewResource("foo", adp)

  adp.contentSize = 42

  adp.collection = false
  clength, found := res.GetContentLength()
  if clength != "42" || !found {
    t.Error("Content length should be 42")
  }

  adp.collection = true
  clength, found = res.GetContentLength()
  if clength != "" || found {
    t.Error("Content length should be marked as not found for collections")
  }
}

func TestLastModified(t *testing.T) {
  adp := new(FakeResourceAdapter)
  res := data.NewResource("foo", adp)

  adp.modtime = time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
  timeFormat := "2006-01-02 15:04:05"
  lastmod, found := res.GetLastModified(timeFormat)

  if lastmod != "2009-11-10 23:00:00" || !found {
    t.Error("Last modified should be equal `2009-11-10 23:00:00`")
  }
}

func TestOwnerPath(t *testing.T) {
  res := data.NewResource("/foo", FakeResourceAdapter{})
  owner, found := res.GetOwnerPath()
  if owner != "" || found {
    t.Error("Path owner should have been empty")
  }

  res = data.NewResource("/foo/bar", FakeResourceAdapter{})
  owner, found = res.GetOwnerPath()
  if owner != "/foo/" || !found {
    t.Error("Path owner should have been `/foo/`")
  }
}

func TestCollectionChildPaths(t *testing.T) {
  adp := new(FakeResourceAdapter)
  res := data.NewResource("/foo", adp)

  adp.collectionChildPaths = []string{"/foo/bar", "/foo/baz"}

  adp.collection = false
  paths, found := res.GetCollectionChildPaths()
  if paths != nil || found {
    t.Error("Collection child paths should not exist for non-collection resources")
  }

  adp.collection = true
  paths, found = res.GetCollectionChildPaths()
  if len(paths) != 2 || paths[0] != "/foo/bar" || paths[1] != "/foo/baz" || !found {
    t.Error("Collection child paths should be [/foo/bar /foo/baz] and it was", paths)
  }
}

type FakeResourceAdapter struct {
  collection bool
  etag string
  contentData string
  contentSize int64
  modtime time.Time
  collectionChildPaths []string
}

func (adp FakeResourceAdapter) IsCollection() bool {
  return adp.collection
}

func (adp FakeResourceAdapter) GetContent() string {
  return adp.contentData
}

func (adp FakeResourceAdapter) GetContentSize() int64 {
  return adp.contentSize
}

func (adp FakeResourceAdapter) CalculateEtag() string {
  return adp.etag
}

func (adp FakeResourceAdapter) GetModTime() time.Time {
  return adp.modtime
}

func (adp FakeResourceAdapter) GetCollectionChildPaths() []string {
  return adp.collectionChildPaths
}
