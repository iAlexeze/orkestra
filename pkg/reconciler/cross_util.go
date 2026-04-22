// pkg/reconciler/cross_util.go
package reconciler

import "github.com/orkspace/orkestra/pkg/utils"

// rawToMap and metaField are thin package-local aliases over the shared
// utils.RawToMap / utils.MetaField so call sites in this package stay terse.

func rawToMap(raw interface{}) (map[string]interface{}, error) {
	return utils.RawToMap(raw)
}

func metaField(objMap map[string]interface{}, field string) string {
	return utils.MetaField(objMap, field)
}
