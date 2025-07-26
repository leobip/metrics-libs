// metrics-common.go
package metricslibs

import (
	"os"
)

// Extrae el namespace del entorno o pone 'default'
func getNamespace() string {
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns
	}
	return "default"
}

// Extrae el cluster desde KUBECONTEXT o usa 'local'
func getClusterFromContext() string {
	if ctx := os.Getenv("KUBECONTEXT"); ctx != "" {
		return ctx
	}
	return "local"
}
