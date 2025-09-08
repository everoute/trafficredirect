package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRule_validateSpec(t *testing.T) {
	tests := []struct {
		name      string
		rule      Rule
		wantErr   bool
		errorText string
	}{
		{
			name: "direct is invalid",
			rule: Rule{Spec: RuleSpec{
				Direct: "test",
			}},
			wantErr:   true,
			errorText: "must set ingress or egress",
		},
		{
			name: "missing rule match",
			rule: Rule{
				Spec: RuleSpec{
					Direct: Ingress,
				},
			},
			wantErr:   true,
			errorText: "must set rule match",
		},
		{
			name: "invalid dst mac",
			rule: Rule{
				Spec: RuleSpec{
					Direct: Ingress,
					Match:  RuleMatch{DstMac: "invalid"},
				},
			},
			wantErr:   true,
			errorText: "mac invalid is invalid",
		},
		{
			name: "invalid src mac",
			rule: Rule{
				Spec: RuleSpec{
					Direct: Ingress,
					Match:  RuleMatch{SrcMac: "zz:zz", DstMac: "12:13:13:13:13:5a"},
				},
			},
			wantErr:   true,
			errorText: "mac zz:zz is invalid",
		},
		{
			name: "incomplete tower option",
			rule: Rule{
				Spec: RuleSpec{
					Direct: Egress,
					Match:  RuleMatch{DstMac: "00:11:22:33:44:55"},
					Option: &Option{
						TowerVM: "vm1",
					},
				},
			},
			wantErr:   true,
			errorText: "must set tower option with vmid and nic",
		},
		{
			name: "valid rule with dst mac",
			rule: Rule{
				Spec: RuleSpec{
					Direct: Ingress,
					Match:  RuleMatch{DstMac: "00:11:22:33:44:55"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid dst mac with -",
			rule: Rule{
				Spec: RuleSpec{
					Direct: Ingress,
					Match:  RuleMatch{DstMac: "00-11-22-33-44-55"},
				},
			},
			wantErr:   true,
			errorText: "mac 00-11-22-33-44-55 is invalid",
		},
		{
			name: "invalid dst mac with A-F",
			rule: Rule{
				Spec: RuleSpec{
					Direct: Ingress,
					Match:  RuleMatch{DstMac: "00:F5:22:33:44:55"},
				},
			},
			wantErr:   true,
			errorText: "mac 00:F5:22:33:44:55 is invalid",
		},
		{
			name: "valid rule with src mac and tower option",
			rule: Rule{
				Spec: RuleSpec{
					Direct: Egress,
					Match:  RuleMatch{SrcMac: "aa:bb:cc:dd:ee:ff"},
					Option: &Option{
						TowerVM: "vm2",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.validateSpec()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorText)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRuleDefault(t *testing.T) {
	r := &Rule{
		Spec: RuleSpec{
			Match: RuleMatch{
				SrcMac: "AA:BB:CC:D5:Ee:FF",
				DstMac: "11:22:33:44:55:ee",
			},
		},
	}

	r.Default()

	assert.Equal(t, "aa:bb:cc:d5:ee:ff", r.Spec.Match.SrcMac)
	assert.Equal(t, "11:22:33:44:55:ee", r.Spec.Match.DstMac) // already lowercase
}
