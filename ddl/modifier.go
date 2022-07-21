package ddl

import (
	"bytes"
	"fmt"
	"strings"
)

// Modifiers is a slice of modifiers.
type Modifiers []Modifier

// Modifier represents a modifier in a ddl struct tag.
type Modifier struct {
	// Dialects is the slice of dialects that the modifier is applicable for.
	// If empty, the modifier is applicable for every dialect.
	Dialects []string

	// Name is the name of the modifier.
	Name string

	// RawValue is the raw value of the modifier.
	RawValue string

	// Value is the value of the modifier (parsed from the RawValue).
	Value string

	// Submodifiers are the submodifiers of the modifier (parsed from the
	// RawValue).
	Submodifiers Modifiers
}

// NewModifiers parses a string into a slice of modifiers.
func NewModifiers(s string) (Modifiers, error) {
	var err error
	var ms Modifiers
	var m Modifier
	remainder := s
	for remainder != "" {
		m, remainder, err = popModifier(remainder)
		if err != nil {
			return ms, err
		}
		if m.Name == "" {
			continue
		}
		ms = append(ms, m)
	}
	return ms, nil
}

// String converts a slice of modifiers into their string representation.
func (ms *Modifiers) String() string {
	buf := bufpool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufpool.Put(buf)
	for i := range *ms {
		(*ms)[i].writeBuf(buf)
	}
	return buf.String()
}

// ExcludesDialect checks if a dialect should be excluded from the effects of
// the modifier.
func (m *Modifier) ExcludesDialect(dialect string) bool {
	if len(m.Dialects) == 0 {
		return false
	}
	for _, modifierDialect := range m.Dialects {
		if modifierDialect == dialect {
			return false
		}
	}
	return true
}

// ParseRawValue parses a modifier's raw value and fills in the fields `Value`
// and `Submodifiers`.
func (m *Modifier) ParseRawValue() error {
	if m.RawValue == "" {
		return nil
	}
	var err error
	var remainder string
	m.Value, remainder, err = popValue(m.RawValue)
	if err != nil {
		return err
	}
	m.Submodifiers, err = NewModifiers(remainder)
	return err
}

// writeBuf writes the string representation of a modifier into a buffer.
func (m *Modifier) writeBuf(buf *bytes.Buffer) {
	if m.Name == "" {
		return
	}
	last := buf.Len() - 1
	if last > 0 && buf.Bytes()[last] != ' ' {
		buf.WriteString(" ")
	}
	buf.WriteString(m.Name)
	if m.RawValue != "" {
		if i := strings.IndexByte(m.RawValue, ' '); i >= 0 {
			buf.WriteString("={" + m.RawValue + "}")
		} else {
			buf.WriteString("=" + m.RawValue)
		}
		return
	}
	if m.Value == "" && len(m.Submodifiers) == 0 {
		return
	}
	buf.WriteString("=")
	valueHasSpace := strings.IndexByte(m.Value, ' ') >= 0
	if valueHasSpace || len(m.Submodifiers) > 0 {
		buf.WriteString("{")
	}
	if m.Value == "" {
		buf.WriteString(".")
	} else if valueHasSpace {
		buf.WriteString("{" + m.Value + "}")
	} else {
		buf.WriteString(m.Value)
	}
	for i := range m.Submodifiers {
		m.Submodifiers[i].writeBuf(buf)
	}
	if valueHasSpace || len(m.Submodifiers) > 0 {
		buf.WriteString("}")
	}
}

// popValue pops the first value from a string and returns the value and
// remainder.
func popValue(s string) (value, remainder string, err error) {
	s = strings.TrimLeft(s, " ")
	if s == "" {
		return "", "", nil
	}
	isBraceQuoted := s[0] == '{'
	whitespaceIndex := -1
	bracelevel := 0
	for i, char := range s {
		if char == ' ' && bracelevel == 0 {
			whitespaceIndex = i
			break
		}
		if isBraceQuoted {
			switch char {
			case '{':
				bracelevel++
			case '}':
				if bracelevel > 0 {
					bracelevel--
				}
			}
		}
	}
	if bracelevel > 0 {
		return s, "", fmt.Errorf("%q: unclosed brace", s)
	}
	if whitespaceIndex > 0 {
		value, remainder = s[:whitespaceIndex], s[whitespaceIndex:]
	} else {
		value = s
	}
	if isBraceQuoted {
		value = value[1 : len(value)-1]
	}
	return value, remainder, nil
}

// popModifier pops the first modifier from a string and returns the modifier and remainder.
func popModifier(s string) (m Modifier, remainder string, err error) {
	s = strings.TrimLeft(s, " ")
	if s == "" {
		return m, "", nil
	}
	whitespaceIndex := -1
	equalsIndex := -1
	for i, char := range s {
		if char == ' ' {
			whitespaceIndex = i
			break
		}
		if char == '=' {
			equalsIndex = i
			break
		}
	}
	if equalsIndex >= 0 {
		m.Name, remainder = s[:equalsIndex], s[equalsIndex+1:]
		if len(remainder) > 0 && remainder[0] == ' ' {
			return m, remainder, nil
		}
		m.RawValue, remainder, err = popValue(remainder)
		if err != nil {
			return m, remainder, err
		}
	} else if whitespaceIndex >= 0 {
		m.Name, remainder = s[:whitespaceIndex], s[whitespaceIndex:]
	} else {
		m.Name, remainder = s, ""
	}
	if i := strings.IndexByte(m.Name, ':'); i >= 0 {
		m.Dialects = strings.Split(m.Name[:i], ",")
		m.Name = m.Name[i+1:]
	}
	return m, remainder, nil
}
