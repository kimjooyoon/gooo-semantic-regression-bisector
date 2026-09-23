package contract

import (
	"crypto/sha256"
	"encoding/hex"
	_ "embed"
	"fmt"
	"strings"
)

//go:embed semantic_bisect.gooo
var source string

type Program struct {
	Name           string
	Version        string
	SourceDigest   string
	Ownership      map[string]string
	Rules          map[string]string
	Activities     []string
	CanonicalCases map[string]string
}

func Source() string {
	return source
}

func Load() (Program, error) {
	return parse(source)
}

func parse(source string) (Program, error) {
	p := Program{
		Ownership:      map[string]string{},
		Rules:          map[string]string{},
		CanonicalCases: map[string]string{},
	}
	hash := sha256.Sum256([]byte(source))
	p.SourceDigest = "sha256:" + hex.EncodeToString(hash[:])
	seenProgram := false
	for lineNumber, raw := range strings.Split(source, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 4 && fields[0] == "program" && fields[2] == "version" {
			if seenProgram {
				return Program{}, fmt.Errorf("contract line %d duplicates program declaration", lineNumber+1)
			}
			seenProgram = true
			p.Name = fields[1]
			p.Version = strings.Trim(fields[3], "\"")
			continue
		}
		if len(fields) == 4 && fields[0] == "owns" && fields[2] == "=" {
			if _, exists := p.Ownership[fields[1]]; exists {
				return Program{}, fmt.Errorf("contract line %d duplicates ownership declaration %q", lineNumber+1, fields[1])
			}
			p.Ownership[fields[1]] = fields[3]
			continue
		}
		if len(fields) >= 4 && fields[0] == "rule" && fields[2] == "=" {
			if _, exists := p.Rules[fields[1]]; exists {
				return Program{}, fmt.Errorf("contract line %d duplicates rule declaration %q", lineNumber+1, fields[1])
			}
			p.Rules[fields[1]] = strings.Join(fields[3:], " ")
			continue
		}
		if len(fields) == 3 && fields[0] == "activity" && fields[2] == "exactly_once" {
			p.Activities = append(p.Activities, fields[1])
			continue
		}
		if len(fields) == 4 && fields[0] == "canonical_case" && fields[2] == "=" {
			if _, exists := p.CanonicalCases[fields[1]]; exists {
				return Program{}, fmt.Errorf("contract line %d duplicates canonical case %q", lineNumber+1, fields[1])
			}
			p.CanonicalCases[fields[1]] = fields[3]
			continue
		}
		return Program{}, fmt.Errorf("contract line %d is not valid Gooo: %q", lineNumber+1, line)
	}
	if err := p.Validate(); err != nil {
		return Program{}, err
	}
	return p, nil
}

func (p Program) Validate() error {
	if p.Name != "semantic_regression_bisector" || p.Version != "0.1.0" {
		return fmt.Errorf("unexpected Gooo program identity %q@%q", p.Name, p.Version)
	}
	requiredOwnership := []string{
		"candidate_ordering",
		"admissible_observation",
		"interval_narrowing",
		"stop_conditions",
		"decision_precedence",
	}
	for _, key := range requiredOwnership {
		if p.Ownership[key] != "gooo" {
			return fmt.Errorf("Gooo must own %s", key)
		}
	}
	if len(p.Activities) != 9 {
		return fmt.Errorf("Gooo activity count is %d, want exactly 9", len(p.Activities))
	}
	seen := map[string]bool{}
	for _, activity := range p.Activities {
		if activity == "" || seen[activity] {
			return fmt.Errorf("Gooo activities must be unique: %q", activity)
		}
		seen[activity] = true
	}
	if len(p.CanonicalCases) != 9 {
		return fmt.Errorf("canonical case count is %d, want exactly 9", len(p.CanonicalCases))
	}
	return nil
}
