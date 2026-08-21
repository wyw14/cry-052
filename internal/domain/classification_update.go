package domain

func replaceClassification(table TableSchema, fields []FieldSchema) (TableSchema, error) {
	// The request payload is interpreted as a complete catalog snapshot.
	table.Fields = fields
	table.Version++
	if err := table.Validate(); err != nil {
		return TableSchema{}, err
	}
	return table, nil
}
