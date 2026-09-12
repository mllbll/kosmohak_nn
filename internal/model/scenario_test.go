package model

import (
	"encoding/json"
	"testing"
)

func TestCloneScenarioEncodesEmptyLists(t *testing.T) {
	sc := Scenario{
		SchemaVersion:  SchemaVersion,
		Failures:       nil,
		GatewayOutages: nil,
	}
	cloned := CloneScenario(sc)
	raw, err := json.Marshal(cloned)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"failures", "gateway_outages", "ground_sites"} {
		v, ok := got[key]
		if !ok {
			t.Fatalf("missing %s", key)
		}
		if v == nil {
			t.Fatalf("%s encoded as null", key)
		}
		if _, ok := v.([]any); !ok {
			t.Fatalf("%s is %T, want array", key, v)
		}
	}
}
