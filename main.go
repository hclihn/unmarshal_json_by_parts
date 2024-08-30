package main

import (
  "fmt"
  "bytes"
	"encoding/json"
  "strings"
  "strconv"
)

var SimpleStringUnmarshalForVersionString = false

func WrapTraceableErrorf(err error, fs string, args ...interface{}) error {
  s := fmt.Sprintf(fs, args...)
  return fmt.Errorf("%s: %w", s, err)
}

type VersionField struct {
	IsStr    bool   // Is a string field? Determines NumValue or StrValue to be used
	NumValue uint64 // numerical value
	StrValue string // string value (ordered string)
}

// VersionString represents a version string
type VersionString struct {
	Version        string         // raw version
	Fields         []VersionField // version fields, nil if no field specified or not numerical
	OrderedVersion bool           // treat the Version as ordered string?
}

func (v *VersionString) FromString(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return WrapTraceableErrorf(nil, "empty version string specified")
	}
	v.Version = s
	v.Fields = nil
	v.OrderedVersion = false
  // numerical version, parse it
	fields := strings.Split(s, ".")
	ver := make([]VersionField, len(fields))
	for i, f := range fields {
		x, err := strconv.ParseUint(f, 10, 64)
		if err != nil { // with the numVerPtn check, this is very unlikely
			return WrapTraceableErrorf(err, "failed to convert field #%d (%s) to a number in version %q", i, f, s)
		}
		ver[i].NumValue = x
		ver[i].IsStr = false
	}
	v.Fields = ver
	return nil
}

func (v *VersionString) UnmarshalJSON(b []byte) error {
	if b[0] == '"' || string(b) == "null" { // backward compatibility support  for a pure string value
		s := strings.Trim(string(b), "\"") // covers "", "null" and null values
		if s == "null" || s == "" {        // null version
			v.Version = ""
			v.Fields = nil
			v.OrderedVersion = false
			return nil
		}
		// simple version
		return v.FromString(s)
	}
	// unmarshal its fields one at a time. If we call Unmarshal(v), we will get infinite loops
	dec := json.NewDecoder(bytes.NewReader(b))
	// read the open brace
	if t, err := dec.Token(); err != nil {
		return WrapTraceableErrorf(err, "failed to decode JSON token for VersionString")
	} else if d, ok := t.(json.Delim); !ok {
		return WrapTraceableErrorf(nil, "expected a JSON delimiter for VersionString, got %T (%s)", t, t)
	} else if d != '{' {
		return WrapTraceableErrorf(nil, "bad JSON delimiter '%s' for VersionString, expected '{'", t)
	}

	// while the object contains values
	for dec.More() {
		if t, err := dec.Token(); err != nil {
			return WrapTraceableErrorf(err, "failed to decode JSON token for VersionString")
		} else if ts, ok := t.(string); !ok {
			return WrapTraceableErrorf(nil, "bad JSON token %T (%s) for VersionString field name", t, t)
		} else {
			switch ts { // unmarshal individual field
			case "Version":
				var s string
				if err := dec.Decode(&s); err != nil {
					return err
				}
				v.Version = s
			case "Fields":
				var f []VersionField
				if err := dec.Decode(&f); err != nil {
					return err
				}
				v.Fields = f
			case "OrderedVersion":
				var b bool
				if err := dec.Decode(&b); err != nil {
					return err
				}
				v.OrderedVersion = b
			default:
				return WrapTraceableErrorf(nil, "unknown field name %q for VersionString type", ts)
			}
		}
	}

	// read the closing brace
	if t, err := dec.Token(); err != nil {
		return WrapTraceableErrorf(err, "failed to decode JSON token for VersionString")
	} else if d, ok := t.(json.Delim); !ok {
		return WrapTraceableErrorf(nil, "expected a JSON delimiter for VersionString, got %T (%s)", t, t)
	} else if d != '}' {
		return WrapTraceableErrorf(nil, "bad JSON delimiter '%s' for VersionString, expected '}'", t)
	}
	return nil
}

type VersionStrings []VersionString

func (vs *VersionStrings) FromString(s string) error {
  fields := strings.Split(s, ";")
    if len(fields) == 0 { // empty
      *vs = nil
      return nil
    }
    vs1 := make(VersionStrings, len(fields))
    for i := range vs1 {
      if err := vs1[i].FromString(fields[i]); err != nil {
        return WrapTraceableErrorf(err, "empty or malformed version string part[%d] %q in %q: %s", i, fields[i], s)
      }
    }
    *vs = vs1
    return nil
}

func (vs *VersionStrings) UnmarshalJSON(b []byte) error {
	if b[0] == '"' || string(b) == "null" { // backward compatibility support  for a pure string value
		s := strings.Trim(string(b), "\"") // covers "", "null" and null values
		if s == "null" || s == "" {        // null version
			*vs = nil
			return nil
		}
		// simple version
		return vs.FromString(s)
	}
	// unmarshal its fields one at a time. If we call Unmarshal(v), we will get infinite loops
	dec := json.NewDecoder(bytes.NewReader(b))
	// read the open brace
	if t, err := dec.Token(); err != nil {
		return WrapTraceableErrorf(err, "failed to decode JSON token for VersionString")
	} else if d, ok := t.(json.Delim); !ok {
		return WrapTraceableErrorf(nil, "expected a JSON delimiter for VersionString, got %T (%s)", t, t)
	} else if d != '[' {
		return WrapTraceableErrorf(nil, "bad JSON delimiter '%s' for VersionString, expected '['", t)
	}

	// while the object contains values
  vs1 := make(VersionStrings, 0)
	for dec.More() {
		var v VersionString
    if err := dec.Decode(&v); err != nil {
      return err
    }
    vs1 = append(vs1, v)
	}

	// read the closing brace
	if t, err := dec.Token(); err != nil {
		return WrapTraceableErrorf(err, "failed to decode JSON token for VersionString")
	} else if d, ok := t.(json.Delim); !ok {
		return WrapTraceableErrorf(nil, "expected a JSON delimiter for VersionString, got %T (%s)", t, t)
	} else if d != ']' {
		return WrapTraceableErrorf(nil, "bad JSON delimiter '%s' for VersionString, expected ']'", t)
	}
  if len(vs1) > 0 {
		*vs = vs1
	} else {
		*vs = nil
	}
	return nil
}

func main() {
  var vs VersionString
  vs.FromString("1.2.3.4")
  fmt.Printf("VersionString: %#v\n", vs)
  b, err := json.Marshal(vs)
  if err != nil {
    fmt.Printf("Error: %s\n", err)
    return
  }
  fmt.Printf("JSON: %s\n", string(b))
  var vs1 VersionString
  if err := json.Unmarshal([]byte(`"1.0.2.3"`), &vs1); err != nil {
    fmt.Printf("Error: %s\n", err)
    return
  }
  fmt.Printf("VersionString1 from simple string JSON: %#v\n", vs1)
  if err := json.Unmarshal(b, &vs1); err != nil {
    fmt.Printf("Error: %s\n", err)
    return
  }
  fmt.Printf("VersionString1 from full JSON: %#v\n", vs1)

  var vss VersionStrings
  vss.FromString("1.2.3.4;0.1.2.6")
  fmt.Printf("VersionStrings: %#v\n", vss)
  b, err = json.Marshal(vss)
  if err != nil {
    fmt.Printf("Error: %s\n", err)
    return
  }
  fmt.Printf("JSON: %s\n", string(b))
  var vss1 VersionStrings
  if err := json.Unmarshal([]byte(`"1.0.2.3"`), &vss1); err != nil {
    fmt.Printf("Error: %s\n", err)
    return
  }
  fmt.Printf("VersionStrings1 from simple string JSON: %#v\n", vss1)
  if err := json.Unmarshal(b, &vss1); err != nil {
    fmt.Printf("Error: %s\n", err)
    return
  }
  fmt.Printf("VersionStrings1 from full JSON: %#v\n", vss1)
}