package domain

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Object interface {
	metav1.Object
	runtime.Object
}

type ObjectList interface {
	metav1.ListInterface
	runtime.Object
}
