package v1alpha1

import (
	"fmt"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var _ admission.Validator = &Rule{}
var _ admission.Defaulter = &Rule{}

func (r *Rule) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(r).Complete()
}

func (r *Rule) ValidateCreate() (admission.Warnings, error) {
	klog.Infof("Start to validate create rule %v", r)
	return nil, r.validateSpec()
}

func (r *Rule) ValidateUpdate(runtime.Object) (admission.Warnings, error) {
	klog.Infof("Start to validate update rule %v", r)
	return nil, r.validateSpec()
}
func (r *Rule) ValidateDelete() (admission.Warnings, error) { return nil, nil }

func (r *Rule) validateSpec() error {
	if r.Spec.Direct != Egress && r.Spec.Direct != Ingress {
		return fmt.Errorf("direct must set ingress or egress")
	}
	if r.Spec.Match.DstMac == "" && r.Spec.Match.SrcMac == "" {
		return fmt.Errorf("must set rule match")
	}
	if r.Spec.Match.DstMac != "" {
		err := r.validateMac(r.Spec.Match.DstMac)
		if err != nil {
			return err
		}
	}
	if r.Spec.Match.SrcMac != "" {
		err := r.validateMac(r.Spec.Match.SrcMac)
		if err != nil {
			return err
		}
	}

	if r.Spec.TowerOption != nil {
		if r.Spec.TowerOption.VMID == "" || r.Spec.TowerOption.Nic == "" {
			return fmt.Errorf("must set tower option with vmid and nic")
		}
	}
	return nil
}

func (r *Rule) validateMac(m string) error {
	regex := `^([0-9a-f]{2}:){5}[0-9a-f]{2}$`
	matched, err := regexp.MatchString(regex, m)
	if err != nil {
		return fmt.Errorf("failed to exec verificate mac %s: %s", m, err)
	}
	if !matched {
		return fmt.Errorf("mac %s is invalid, doesn't match %s", m, regex)
	}
	return nil
}

func (r *Rule) Default() {
	klog.Infof("Start to modify rule %v", r)
	r.Spec.Match.SrcMac = strings.ToLower(r.Spec.Match.SrcMac)
	r.Spec.Match.DstMac = strings.ToLower(r.Spec.Match.DstMac)
}
