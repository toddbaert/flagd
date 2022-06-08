package model

// TODO: use some composition/interfaces here to reduce duplication
type Flags struct {
	BooleanFlags map[string]BooleanFlag `json:"booleanFlags"`
	StringFlags map[string]StringFlag `json:"stringFlags"`
	NumericFlags map[string]NumberFlag `json:"numericFlags"`
	ObjectFlags map[string]ObjectFlag `json:"objectFlags"`
}

type BooleanFlag struct {
	State bool `json:"state"`
	Variants map[string]bool `json:"variants"`
	DefaultVariant string	`json:"defaultVariant"`
	Rules []interface{} `json:"rules"`
}

type StringFlag struct {
	State bool `json:"state"`
	Variants map[string]string `json:"variants"`
	DefaultVariant string	`json:"defaultVariant"`
	Rules []interface{} `json:"rules"`
}

type NumberFlag struct {
	State bool `json:"state"`
	Variants map[string]float32 `json:"variants"`
	DefaultVariant string	`json:"defaultVariant"`
	Rules []interface{} `json:"rules"`
}

type ObjectFlag struct {
	State bool `json:"state"`
	Variants map[string]map[string]interface{} `json:"variants"`
	DefaultVariant string	`json:"defaultVariant"`
	Rules []interface{} `json:"rules"`
}