package domain

import "time"

type ValueChange struct {
	Field  string `json:"field"`
	Before string `json:"before"`
	After  string `json:"after"`
	Rule   string `json:"rule"`
}

type PreviewRow struct {
	RowKey  string        `json:"row_key"`
	Changes []ValueChange `json:"changes"`
}

type Preview struct {
	ID               string            `json:"id"`
	SourceTableID    string            `json:"source_table_id"`
	TargetTableID    string            `json:"target_table_id"`
	PolicyVersionID  string            `json:"policy_version_id"`
	InputFingerprint string            `json:"input_fingerprint"`
	Mappings         []FieldMapping    `json:"mappings"`
	Rows             []PreviewRow      `json:"rows"`
	StrategyCounters map[string]int    `json:"strategy_counters"`
	Conflicts        []MappingConflict `json:"conflicts"`
	ConfirmedBy      string            `json:"confirmed_by,omitempty"`
	ConfirmedAt      *time.Time        `json:"confirmed_at,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
}

func (p *Preview) Confirm(actor string, now time.Time) error {
	if actor == "" {
		return NewValidationError("actor is required")
	}
	if len(p.Conflicts) > 0 {
		return ErrConflict
	}
	if p.ConfirmedAt != nil {
		if p.ConfirmedBy == actor {
			return nil
		}
		return ErrConflict
	}
	p.ConfirmedBy = actor
	p.ConfirmedAt = &now
	return nil
}

func (p Preview) IsConfirmed() bool { return p.ConfirmedAt != nil && p.ConfirmedBy != "" }
