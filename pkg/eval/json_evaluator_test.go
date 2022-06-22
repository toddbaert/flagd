package eval

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	gen "github.com/open-feature/flagd/pkg/generated"
	"github.com/open-feature/flagd/pkg/model"
)

const InvalidFlags = `{
  "flags": {
    "invalidFlag": {
      "notState": "ENABLED",
      "notVariants": {
        "on": true,
        "off": false
      },
      "notDefaultVariant": "on"
    }
  }
}`

const ValidFlags = `{
  "flags": {
    "validFlag": {
      "state": "ENABLED",
      "variants": {
        "on": true,
        "off": false
      },
      "defaultVariant": "on"
    }
  }
}`

const StaticBoolFlag = "staticBoolFlag"
const StaticBoolValue = true
const StaticStringFlag = "staticStringFlag"
const StaticStringValue = "#CC0000"
const StaticNumberFlag = "staticNumberFlag"
const StaticNumberValue float32 = 1
const StaticObjectFlag = "staticObjectFlag"
const StaticObjectValue = `{"abc": 123}`
var StaticFlags = fmt.Sprintf(`{
  "flags": {
    "%s": {
      "state": "ENABLED",
      "variants": {
        "on": %t,
        "off": false
      },
      "defaultVariant": "on"
    },
		"%s": {
      "state": "ENABLED",
      "variants": {
        "red": "%s",
        "blue": "#0000CC"
      },
      "defaultVariant": "red"
    },
		"%s": {
      "state": "ENABLED",
      "variants": {
        "one": %f,
        "two": 2
      },
      "defaultVariant": "one"
    },
		"%s": {
      "state": "ENABLED",
      "variants": {
        "obj1": %s,
        "obj2": {
					"xyz": true
				}
      },
      "defaultVariant": "obj1"
    }
  }
}`,
StaticBoolFlag,
StaticBoolValue,
StaticStringFlag,
StaticStringValue,
StaticNumberFlag,
StaticNumberValue,
StaticObjectFlag,
StaticObjectValue)

const DynamicFlag = "ruleFlag";
const ColorProp = "color";
const ColorValue = "yellow";
var DynamicFlags = fmt.Sprintf( `{
  "flags": {
    "%s": {
      "state": "ENABLED",
      "variants": {
        "on": true,
        "off": false
      },
      "defaultVariant": "off",
			"rules": {
        "if": [
          {
            "==": [
              {
                "var": [
                  "%s"
                ]
              },
              "%s"
            ]
          },
          "on",
          null
        ]
      }
    }
  }
}`, DynamicFlag, ColorProp, ColorValue)

func TestGetState_Valid_ContainsFlag(t *testing.T) {
	evaluator := JsonEvaluator{}
	// set the state internally
	json.Unmarshal([]byte(ValidFlags), &evaluator.state)
	
	// get the state
	state, err := evaluator.GetState()
	if err != nil {
		t.Fatalf("Expected no error")
	}

	// validate it contains the flag
	wants := "validFlag"
	if !strings.Contains(state, wants) {
		t.Fatalf("Expected %s to contain %s", state, wants)
	}
}

func TestSetState_Invalid_Error(t *testing.T) {
	evaluator := JsonEvaluator{}

	// set state with an invalid flag definition
	err := evaluator.SetState(InvalidFlags)
	if err == nil {
		t.Fatalf("Expected error")
	}
}

func TestSetState_Valid_NoError(t *testing.T) {
	evaluator := JsonEvaluator{}

	// set state with a valid flag definition
	err := evaluator.SetState(ValidFlags)
	if err != nil {
		t.Fatalf("Expected no error")
	}
}

func TestResolveBooleanValue_FlagExistsStatic_ReturnsValue(t *testing.T) {
	evaluator := JsonEvaluator{}

	// set state with an static flag definition
	err := evaluator.SetState(StaticFlags)
	if err != nil {
		t.Fatalf("Expected no error")
	}

	// evaluate
	wantedVal := StaticBoolValue
	val, reason, err := evaluator.ResolveBooleanValue(StaticBoolFlag, false, gen.Context{})
	if assert.NoError(t, err) {
		assert.Equal(t, val, wantedVal)
		assert.Equal(t, reason, model.StaticReason)
	}
}

func TestResolveBooleanValue_NotBoolean_Error(t *testing.T) {
	evaluator := JsonEvaluator{}

	// set state with an static flag definition
	err := evaluator.SetState(StaticFlags)
	if err != nil {
		t.Fatalf("Expected no error")
	}

	// evaluate a non-boolean flag
	_, _, err = evaluator.ResolveBooleanValue(StaticObjectFlag, false, gen.Context{})
	assert.EqualError(t, err, model.TypeMismatchErrorCode)
}

func TestResolveStringValue_FlagExistsStatic_ReturnsValue(t *testing.T) {
	evaluator := JsonEvaluator{}

	// set state with an static flag definition
	err := evaluator.SetState(StaticFlags)
	if err != nil {
		t.Fatalf("Expected no error")
	}

	// evaluate
	wantedVal := StaticStringValue
	val, reason, err := evaluator.ResolveStringValue(StaticStringFlag, "other", gen.Context{})
	if assert.NoError(t, err) {
		assert.Equal(t, val, wantedVal)
		assert.Equal(t, reason, model.StaticReason)
	}
}

func TestResolveStringValue_NotString_Error(t *testing.T) {
	evaluator := JsonEvaluator{}

	// set state with an static flag definition
	err := evaluator.SetState(StaticFlags)
	if err != nil {
		t.Fatalf("Expected no error")
	}

	// evaluate a non-string flag
	_, _, err = evaluator.ResolveStringValue(StaticObjectFlag, "other", gen.Context{})
	assert.EqualError(t, err, model.TypeMismatchErrorCode)
}

func TestResolveNumberValue_FlagExistsStatic_ReturnsValue(t *testing.T) {
	evaluator := JsonEvaluator{}

	// set state with an static flag definition
	err := evaluator.SetState(StaticFlags)
	if err != nil {
		t.Fatalf("Expected no error")
	}

	// evaluate
	wantedVal := StaticNumberValue
	val, reason, err := evaluator.ResolveNumberValue(StaticNumberFlag, 0, gen.Context{})
	if assert.NoError(t, err) {
		assert.Equal(t, val, wantedVal)
		assert.Equal(t, reason, model.StaticReason)
	}
}

func TestResolveNumberValue_NotNumber_Error(t *testing.T) {
	evaluator := JsonEvaluator{}

	// set state with an static flag definition
	err := evaluator.SetState(StaticFlags)
	if err != nil {
		t.Fatalf("Expected no error")
	}

	// evaluate a non-number flag
	_, _, err = evaluator.ResolveNumberValue(StaticObjectFlag, 0, gen.Context{})
	assert.EqualError(t, err, model.TypeMismatchErrorCode)
}

func TestResolveObjectValue_FlagExistsStatic_ReturnsValue(t *testing.T) {
	evaluator := JsonEvaluator{}

	// set state with an static flag definition
	err := evaluator.SetState(StaticFlags)
	if err != nil {
		t.Fatalf("Expected no error")
	}

	// evaluate
	wantedVal := StaticObjectValue
	val, reason, err := evaluator.ResolveObjectValue(StaticObjectFlag, map[string]interface{}{ "def": 123 }, gen.Context{})
	if assert.NoError(t, err) {
		marshalled, err := json.Marshal(val)
		if assert.NoError(t, err) {
			assert.JSONEq(t, string(marshalled), wantedVal)
			assert.Equal(t, reason, model.StaticReason)
		}
	}

}

func TestResolveObjectValue_NotObject_Error(t *testing.T) {
	evaluator := JsonEvaluator{}

	// set state with an static flag definition
	err := evaluator.SetState(StaticFlags)
	if err != nil {
		t.Fatalf("Expected no error")
	}

	// evaluate a non-object flag
	_, _, err = evaluator.ResolveObjectValue(StaticBoolFlag, map[string]interface{}{ "def": 123 }, gen.Context{})
	assert.EqualError(t, err, model.TypeMismatchErrorCode)
}

func TestResolveXxxValue_RuleResolvesVariant_DynamicValue(t *testing.T) {
	evaluator := JsonEvaluator{}

	// set state with an static flag definition
	err := evaluator.SetState(DynamicFlags)
	if err != nil {
		t.Fatalf("Expected no error")
	}

	// evaluate the dynamic flag, this should return a variant, and therefor reasons should be TARGET_MATCH
	val, reason, err := evaluator.ResolveBooleanValue(DynamicFlag, false, gen.Context{ AdditionalProperties: map[string]interface{}{
		ColorProp: ColorValue,
		} })
	if assert.NoError(t, err) {
		assert.True(t, val)
		assert.Equal(t, reason, model.TargetingMatchReason)
	}
}

func TestResolveXxxValue_RuleResolvesNonVariant_StaticValue(t *testing.T) {
	evaluator := JsonEvaluator{}

	// set state with an static flag definition
	err := evaluator.SetState(DynamicFlags)
	if err != nil {
		t.Fatalf("Expected no error")
	}

	// evaluate the dynamic flag, this should return null, and therefor reasons should be STATIC
	val, reason, err := evaluator.ResolveBooleanValue(DynamicFlag, false, gen.Context{ AdditionalProperties: map[string]interface{}{
		ColorProp: "red", // not the expected value for the rule to match
		} })
	if assert.NoError(t, err) {
		assert.False(t, val)
		assert.Equal(t, reason, model.StaticReason)
	}
}