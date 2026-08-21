package domain

import (
	"fmt"
	"sort"
	"strings"
)

type FieldMapping struct {
	SourceField string     `json:"source_field"`
	TargetField string     `json:"target_field"`
	Strategies  []Strategy `json:"strategies"`
}

type MappingConflict struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func DetectMappingConflicts(source, target TableSchema, mappings []FieldMapping) []MappingConflict {
	conflicts := make([]MappingConflict, 0)
	seenTargets := map[string]string{}
	for _, mapping := range mappings {
		sourceField, sourceOK := source.Field(mapping.SourceField)
		targetField, targetOK := target.Field(mapping.TargetField)
		if !sourceOK {
			conflicts = append(conflicts, MappingConflict{Field: mapping.SourceField, Code: "SOURCE_FIELD_MISSING", Message: "source field is not present in the snapshot"})
			continue
		}
		if !targetOK {
			conflicts = append(conflicts, MappingConflict{Field: mapping.TargetField, Code: "TARGET_FIELD_MISSING", Message: "target field does not exist"})
			continue
		}
		key := strings.ToLower(mapping.TargetField)
		if previous, exists := seenTargets[key]; exists {
			conflicts = append(conflicts, MappingConflict{Field: mapping.TargetField, Code: "TARGET_COLLISION", Message: fmt.Sprintf("also mapped from %s", previous)})
		}
		seenTargets[key] = mapping.SourceField
		if sourceField.DataType != targetField.DataType {
			conflicts = append(conflicts, MappingConflict{Field: mapping.TargetField, Code: "TYPE_MISMATCH", Message: sourceField.DataType + " cannot be written to " + targetField.DataType})
		}
		if len(mapping.Strategies) == 0 && sourceField.Sensitivity >= SensitivityConfidential {
			conflicts = append(conflicts, MappingConflict{Field: mapping.SourceField, Code: "SENSITIVE_UNPROTECTED", Message: "sensitive field has no strategy"})
		}
	}
	sort.Slice(conflicts, func(i, j int) bool {
		if conflicts[i].Field == conflicts[j].Field {
			return conflicts[i].Code < conflicts[j].Code
		}
		return conflicts[i].Field < conflicts[j].Field
	})
	return conflicts
}
