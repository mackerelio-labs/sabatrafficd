package mib

import (
	"fmt"
	"regexp"
)

func Oidmapping() map[string]string {
	return oidMapping
}

var oidMapping = map[string]string{
	"ifInOctets":    "1.3.6.1.2.1.2.2.1.10",
	"ifOutOctets":   "1.3.6.1.2.1.2.2.1.16",
	"ifHCInOctets":  "1.3.6.1.2.1.31.1.1.1.6",
	"ifHCOutOctets": "1.3.6.1.2.1.31.1.1.1.10",
	"ifInDiscards":  "1.3.6.1.2.1.2.2.1.13",
	"ifOutDiscards": "1.3.6.1.2.1.2.2.1.19",
	"ifInErrors":    "1.3.6.1.2.1.2.2.1.14",
	"ifOutErrors":   "1.3.6.1.2.1.2.2.1.20",
}

func Validate(rawMibs []string) ([]string, error) {
	var parseMibs []string
	if len(rawMibs) == 0 {
		for key := range Oidmapping() {
			// skipped 32 bit octets.
			if key == "ifInOctets" || key == "ifOutOctets" {
				continue
			}
			parseMibs = append(parseMibs, key)
		}
		return parseMibs, nil
	}

	for _, name := range rawMibs {
		if _, exists := Oidmapping()[name]; !exists {
			return nil, fmt.Errorf("mib %s is not supported", name)
		}
		parseMibs = append(parseMibs, name)
	}
	return parseMibs, nil
}

var re = regexp.MustCompile(`^([\d]+\.)+[\d]+$`)

// TODO smi support
func ValidateCustom(mib string) error {
	if valid := re.MatchString(mib); !valid {
		return fmt.Errorf("mib '%s' is not supported", mib)
	}
	return nil
}
