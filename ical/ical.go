// Implemented on top of https://github.com/laurent22/ical-go.

package ical

import (
	"strings"
	"regexp"
	"time"
	"strconv"
	"errors"
	"log"
)

const (
	VCALENDAR = "VCALENDAR"
	VEVENT  = "VEVENT"
	DTSTART = "DTSTART"
	DTEND   = "DTEND"
	DURATION   = "DURATION"
)

type Node struct {
	Name string
	Value string
	Type int // 1 = Object, 0 = Name/Value
	Parameters map[string]string
	Children []*Node
}

func (this *Node) ChildrenByName(name string) []*Node {
	var output []*Node
	for _, child := range this.Children {
		if child.Name == name {
			output = append(output, child)
		}
	}
	return output
}

func (this *Node) ChildByName(name string) *Node {
	for _, child := range this.Children {
		if child.Name == name {
			return child
		}
	}
	return nil
}

func (this *Node) PropString(name string, defaultValue string) string {
	for _, child := range this.Children {
		if child.Name == name {
			return child.Value
		}
	}
	return defaultValue
}

func (this *Node) PropDate(name string, defaultValue time.Time) time.Time {
	node := this.ChildByName(name)
	if node == nil { return defaultValue }
	tzid := node.Parameter("TZID", "")
	var output time.Time
	var err error
	if tzid != "" {
		loc, err := time.LoadLocation(tzid)
		if err != nil { panic(err) }
		output, err = time.ParseInLocation("20060102T150405", node.Value, loc)
	} else {
		output, err = time.Parse("20060102T150405Z", node.Value)
	}

	if err != nil { panic(err) }
	return output
}

func (this *Node) PropDuration(name string) time.Duration {
	durStr := this.PropString(name, "")

	if durStr == "" {
		return time.Duration(0)
	}

	durRgx := regexp.MustCompile("PT(?:([0-9]+)H)?(?:([0-9]+)M)?(?:([0-9]+)S)?")
	matches := durRgx.FindStringSubmatch(durStr)

	if len(matches) != 4 {
		return time.Duration(0)
	}

	strToDuration := func(value string) time.Duration {
		d := 0
		if value != "" {
			d, _ = strconv.Atoi(value)
		}
		return time.Duration(d)
	}

	hours := strToDuration(matches[1])
	min := strToDuration(matches[2])
	sec := strToDuration(matches[3])

	return hours * time.Hour +  min * time.Minute + sec * time.Second
}

func (this *Node) PropInt(name string, defaultValue int) int {
	n := this.PropString(name, "")
	if n == "" { return defaultValue }
	output, err := strconv.Atoi(n)
	if err != nil { panic(err) }
	return output
}

func (this *Node) DigProperty(propPath... string) (string, bool) {
	return this.dig("prop", propPath...)
}

func (this *Node) Parameter(name string, defaultValue string) string {
	if len(this.Parameters) <= 0 { return defaultValue }
	v, ok := this.Parameters[name]
	if !ok { return defaultValue }
	return v
}

func (this *Node) DigParameter(paramPath... string) (string, bool) {
	return this.dig("param", paramPath...)
}

// Digs a value based on a given value path.
// valueType: can be "param" or "prop".
// valuePath: the path to access the value.
// Returns ("", false) when not found or (value, true) when found.
//
// Example:
// dig("param", "VCALENDAR", "VEVENT", "DTEND", "TYPE") -> It will search for "VCALENDAR" node,
// then a "VEVENT" node, then a "DTEND" note, then finally the "TYPE" param.
func (this *Node) dig(valueType string, valuePath... string) (string, bool) {
	current := this
	lastIndex := len(valuePath) - 1
	for _, v := range valuePath[:lastIndex] {
		current = current.ChildByName(v)

		if current == nil {
			return "", false
		}
	}

	target := valuePath[lastIndex]

	value := ""
	if valueType == "param" {
		value = current.Parameter(target, "")
	} else if valueType == "prop" {
		value = current.PropString(target, "")
	}

	if value == "" {
		return "", false
	}

	return value, true
}

func (this *Node) String() string {
	s := ""
	if this.Type == 1 {
		s += "===== " + this.Name
		s += "\n"
	} else {
		s += this.Name
		s += ":" + this.Value
		s += "\n"
	}
	for _, child := range this.Children {
		s += child.String()
	}
	if this.Type == 1 {
		s += "===== /" + this.Name
		s += "\n"
	}
	return s
}

func UnescapeTextType(s string) string {
	s = strings.Replace(s, "\\;", ";", -1)
	s = strings.Replace(s, "\\,", ",", -1)
	s = strings.Replace(s, "\\n", "\n", -1)
	s = strings.Replace(s, "\\\\", "\\", -1)
	return s
}

func ParseTextType(lines []string, lineIndex int) (string, int) {
	line := lines[lineIndex]
	colonIndex := strings.Index(line, ":")
	output := strings.TrimSpace(line[colonIndex+1:len(line)])
	lineIndex++
	for {
		line := lines[lineIndex]
		if line == "" || line[0] != ' ' {
			return UnescapeTextType(output), lineIndex
		}
		output += line[1:len(line)]
		lineIndex++
	}
	return UnescapeTextType(output), lineIndex
}

func ParseCalendar(data string) (*Node, error) {
	r := regexp.MustCompile("([\r|\t| ]*\n[\r|\t| ]*)+")
	lines := r.Split(strings.TrimSpace(data), -1)
	node, _, err, _  := ParseCalendarNode(lines, 0)

	return node, err
}

func ParseCalendarNode(lines []string, lineIndex int) (*Node, bool, error, int) {
	line := strings.TrimSpace(lines[lineIndex])
	_ = log.Println
	colonIndex := strings.Index(line, ":")
	if colonIndex <= 0 {
		return nil, false, errors.New("Invalid value/pair: " + line), lineIndex + 1
	}
	name := line[0:colonIndex]
	splitted := strings.Split(name, ";")
	var parameters map[string]string
	if len(splitted) >= 2 {
		name = splitted[0]
		parameters = make(map[string]string)
		for i := 1; i < len(splitted); i++ {
			p := strings.Split(splitted[i], "=")
			if len(p) != 2 { panic("Invalid parameter format: " + name) }
			parameters[p[0]] = p[1]
		}
	}
	value := line[colonIndex+1:len(line)]

	if name == "BEGIN" {
		node := new(Node)
		node.Name = value
		node.Type = 1
		lineIndex = lineIndex + 1
		for {
			child, finished, _, newLineIndex := ParseCalendarNode(lines, lineIndex)
			if finished {
				return node, false, nil, newLineIndex
			} else {
				if child != nil {
					node.Children = append(node.Children, child)
				}
				lineIndex = newLineIndex
			}
		}
	} else if name == "END" {
		return nil, true, nil, lineIndex + 1
	} else {
		node := new(Node)
		node.Name = name
		if name == "DESCRIPTION" || name == "SUMMARY" {
			text, newLineIndex := ParseTextType(lines, lineIndex)
			node.Value = text
			node.Parameters = parameters
			return node, false, nil, newLineIndex
		} else {
			node.Value = value
			node.Parameters = parameters
			return node, false, nil, lineIndex + 1
		}
	}

	panic("Unreachable")
	return nil, false, nil, lineIndex + 1
}
