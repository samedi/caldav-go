package test

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/yosssi/gohtml"
)

func AssertStr(target string, expectation string, t *testing.T) {
	if target != expectation {
		t.Error("Expected:", expectation, "| Got:", target, "\n ->", logFailedLine())
	}
}

func AssertInt(target int, expectation int, t *testing.T) bool {
	if target != expectation {
		t.Error("Expected:", expectation, "| Got:", target, "\n ->", logFailedLine())
		return false
	}

	return true
}

func AssertResourceDoesNotExist(rpath string, t *testing.T) {
	pwd, _ := os.Getwd()
	if _, err := os.Stat(pwd + rpath); !os.IsNotExist(err) {
		t.Error("Resource", rpath, "exists", "\n ->", logFailedLine())
	}
}

func AssertResourceExists(rpath string, t *testing.T) {
	pwd, _ := os.Getwd()
	_, err := os.Stat(pwd + rpath)
	if os.IsNotExist(err) {
		t.Error("Resource", rpath, "does not exist", "\n ->", logFailedLine())
	} else {
		panicerr(err)
	}
}

func AssertResourceData(rpath, expectation string, t *testing.T) {
	pwd, _ := os.Getwd()
	data, err := ioutil.ReadFile(pwd + rpath)
	dataStr := string(data)
	panicerr(err)
	if dataStr != expectation {
		t.Error("Expected:", expectation, "| Got:", dataStr, "\n ->", logFailedLine())
	}
}

func AssertMultistatusXML(target, expectation string, t *testing.T) {
	cleanXML := func(xml string) string {
		cleanupMap := map[string]string{
			`\r?\n`:                          "",
			`>[\s|\t]+<`:                     "><",
			`<D:getetag>[^<]+</D:getetag>`:   `<D:getetag>?</D:getetag>`,
			`<CS:getctag>[^<]+</CS:getctag>`: `<CS:getctag>?</CS:getctag>`,
			`<D:getlastmodified>[^<]+</D:getlastmodified>`: `<D:getlastmodified>?</D:getlastmodified>`,
		}

		for k, v := range cleanupMap {
			re := regexp.MustCompile(k)
			xml = re.ReplaceAllString(xml, v)
		}

		return strings.TrimSpace(xml)
	}

	target2 := cleanXML(target)
	expectation2 := cleanXML(expectation)

	if target2 != expectation2 {
		target3 := gohtml.Format(target2)
		expectation3 := gohtml.Format(expectation2)

		t.Error("\n== Expected XML ==\n\n", expectation3, "\n\n== Got XML ==\n\n", target3, "\n\n ->", logFailedLine())
	}
}

func logFailedLine() string {
	pc, fn, line, _ := runtime.Caller(2)
	return fmt.Sprintf("Failed in %s[%s:%d]", runtime.FuncForPC(pc).Name(), fn, line)
}
