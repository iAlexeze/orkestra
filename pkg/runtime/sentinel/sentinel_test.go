package sentinel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func obj(gen int64, labels, annotations map[string]string) metav1.Object {
	o := &metav1.ObjectMeta{
		Generation:  gen,
		Labels:      labels,
		Annotations: annotations,
	}
	return o
}

func TestCompute_Empty(t *testing.T) {
	result := Compute(nil, obj(1, nil, nil), obj(2, nil, nil))
	assert.Nil(t, result)
}

func TestCompute_GenerationChanged(t *testing.T) {
	result := Compute([]string{string(GenerationChanged)},
		obj(1, nil, nil), obj(2, nil, nil))
	assert.Equal(t, "true", result[string(GenerationChanged)])
}

func TestCompute_GenerationUnchanged(t *testing.T) {
	result := Compute([]string{string(GenerationChanged)},
		obj(5, nil, nil), obj(5, nil, nil))
	assert.Equal(t, "false", result[string(GenerationChanged)])
}

func TestCompute_LabelsChanged(t *testing.T) {
	result := Compute([]string{string(LabelsChanged)},
		obj(1, map[string]string{"a": "1"}, nil),
		obj(1, map[string]string{"a": "2"}, nil))
	assert.Equal(t, "true", result[string(LabelsChanged)])
}

func TestCompute_LabelsUnchanged(t *testing.T) {
	result := Compute([]string{string(LabelsChanged)},
		obj(1, map[string]string{"a": "1"}, nil),
		obj(1, map[string]string{"a": "1"}, nil))
	assert.Equal(t, "false", result[string(LabelsChanged)])
}

func TestCompute_AnnotationsChanged(t *testing.T) {
	result := Compute([]string{string(AnnotationsChanged)},
		obj(1, nil, map[string]string{"x": "old"}),
		obj(1, nil, map[string]string{"x": "new"}))
	assert.Equal(t, "true", result[string(AnnotationsChanged)])
}

func TestCompute_OnlyDeclaredAreComputed(t *testing.T) {
	result := Compute([]string{string(GenerationChanged)},
		obj(1, map[string]string{"a": "1"}, nil),
		obj(2, map[string]string{"b": "2"}, nil))
	assert.Equal(t, "true", result[string(GenerationChanged)])
	_, hasLabels := result[string(LabelsChanged)]
	assert.False(t, hasLabels)
}

func TestCompute_DeletionStarted(t *testing.T) {
	now := metav1.Now()
	old := &metav1.ObjectMeta{}
	new := &metav1.ObjectMeta{DeletionTimestamp: &now}
	result := Compute([]string{string(DeletionStarted)}, old, new)
	assert.Equal(t, "true", result[string(DeletionStarted)])
}

func TestCompute_DeletionStarted_AlreadyDeleting(t *testing.T) {
	now := metav1.Now()
	old := &metav1.ObjectMeta{DeletionTimestamp: &now}
	new := &metav1.ObjectMeta{DeletionTimestamp: &now}
	result := Compute([]string{string(DeletionStarted)}, old, new)
	assert.Equal(t, "false", result[string(DeletionStarted)])
}

func TestCompute_FinalizersChanged(t *testing.T) {
	old := &metav1.ObjectMeta{Finalizers: []string{"orkestra.io/protect"}}
	new := &metav1.ObjectMeta{}
	result := Compute([]string{string(FinalizersChanged)}, old, new)
	assert.Equal(t, "true", result[string(FinalizersChanged)])
}

func TestCompute_FinalizersUnchanged(t *testing.T) {
	old := &metav1.ObjectMeta{Finalizers: []string{"orkestra.io/protect"}}
	new := &metav1.ObjectMeta{Finalizers: []string{"orkestra.io/protect"}}
	result := Compute([]string{string(FinalizersChanged)}, old, new)
	assert.Equal(t, "false", result[string(FinalizersChanged)])
}

func TestCompute_UnknownSentinelReturnsEmpty(t *testing.T) {
	result := Compute([]string{"specChanged"},
		obj(1, nil, nil), obj(2, nil, nil))
	assert.Equal(t, "", result["specChanged"])
}
