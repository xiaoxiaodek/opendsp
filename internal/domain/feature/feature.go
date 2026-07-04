package feature

// FeatureSet is a key-value map of feature names to their float64 values.
// Used by the scoring stage to provide model inputs.
type FeatureSet struct {
	Values map[string]float64
}

// NewFeatureSet creates an empty FeatureSet.
func NewFeatureSet() FeatureSet {
	return FeatureSet{Values: make(map[string]float64)}
}

// Get returns the feature value for a key, or 0 if not present.
func (fs FeatureSet) Get(key string) float64 {
	return fs.Values[key]
}

// Set adds or updates a feature value.
func (fs FeatureSet) Set(key string, value float64) {
	fs.Values[key] = value
}

// IsEmpty returns true if no features are set.
func (fs FeatureSet) IsEmpty() bool {
	return len(fs.Values) == 0
}
