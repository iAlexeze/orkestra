package utils

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
)

// MatchesFieldRequirements checks that u satisfies all requirements using
// simple string equality on unstructured field values. Used as a post-filter
// after an index lookup to handle residual requirements the index did not cover.
func MatchesFieldRequirements(u *unstructured.Unstructured, reqs fields.Requirements) bool {
	for _, req := range reqs {
		parts := SplitField(req.Field)
		if len(parts) == 0 {
			continue
		}
		val, _, _ := unstructured.NestedString(u.Object, parts...)
		if val != req.Value {
			return false
		}
	}
	return true
}
