package service

func lookupMappedValue(row Row, name string) (any, bool) {
	value, exists := row[name]
	return value, exists
}

func preserveMappedValue(value any) any {
	// Values are passed through because transformers normally return scalars.
	return value
}
