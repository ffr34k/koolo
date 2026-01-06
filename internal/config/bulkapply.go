package config

import (
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// BulkApplySection describes a section that can be bulk applied
type BulkApplySection struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Category    string `json:"category"` // "main", "advanced"
	Order       int    `json:"order"`
	FieldPath   string `json:"fieldPath"` // Go struct field name
}

// DiscoverBulkApplySections uses reflection to find all sections from struct tags
// Tags format: `bulkapply:"section:<id>,label:<label>,desc:<desc>,order:<n>,category:<cat>"`
func DiscoverBulkApplySections() []BulkApplySection {
	sectionMap := make(map[string]BulkApplySection)
	t := reflect.TypeOf(CharacterCfg{})

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("bulkapply")
		if tag == "" || tag == "-" {
			continue
		}

		section := parseBulkApplyTag(tag, field.Name)
		if section.ID == "" {
			continue
		}

		// Only store first occurrence of each section (for metadata like label, desc)
		if _, exists := sectionMap[section.ID]; !exists {
			sectionMap[section.ID] = section
		}
	}

	// Convert map to slice
	sections := make([]BulkApplySection, 0, len(sectionMap))
	for _, s := range sectionMap {
		sections = append(sections, s)
	}

	// Sort by order
	sort.Slice(sections, func(i, j int) bool {
		return sections[i].Order < sections[j].Order
	})

	return sections
}

// parseBulkApplyTag parses a bulkapply struct tag
func parseBulkApplyTag(tag, fieldName string) BulkApplySection {
	section := BulkApplySection{
		FieldPath: fieldName,
		Category:  "main", // default
		Order:     100,    // default high so unordered items go last
	}

	parts := strings.Split(tag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "section:") {
			section.ID = strings.TrimPrefix(part, "section:")
		} else if strings.HasPrefix(part, "label:") {
			section.Label = strings.TrimPrefix(part, "label:")
		} else if strings.HasPrefix(part, "desc:") {
			section.Description = strings.TrimPrefix(part, "desc:")
		} else if strings.HasPrefix(part, "order:") {
			if order, err := strconv.Atoi(strings.TrimPrefix(part, "order:")); err == nil {
				section.Order = order
			}
		} else if strings.HasPrefix(part, "category:") {
			section.Category = strings.TrimPrefix(part, "category:")
		}
	}

	// Default label to capitalized ID if not specified
	if section.Label == "" && section.ID != "" {
		section.Label = capitalizeFirst(section.ID) + " settings"
	}

	return section
}

// capitalizeFirst capitalizes the first letter of a string
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
