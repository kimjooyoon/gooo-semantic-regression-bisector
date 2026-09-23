package contract

import "testing"

func TestValidateRequiresAllOwnedRules(t *testing.T) {
	program, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	delete(program.Rules, "stop_conditions")
	if err := program.Validate(); err == nil {
		t.Fatal("Validate accepted a contract with a missing owned rule")
	}
}
