package provider

type IProvider interface {
	Initialize() error
	ResolveBooleanValue(flagKey string, defaultValue bool) (bool, error)
	ResolveStringValue(flagKey string, defaultValue string) (string, error)
	ResolveNumberValue(flagKey string, defaultValue float32) (float32, error)
	ResolveObjectValue(flagKey string, defaultValue map[string]interface{}) (map[string]interface{}, error)
}