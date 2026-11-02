// Package version expõe a versão do daylist, preenchida em tempo de build.
package version

import (
	"fmt"
	"strings"
)

var (
	// Version é a versão do build; pode ser sobrescrita com -ldflags.
	Version = "dev"
	// Commit é o commit curto do build; pode ser sobrescrito com -ldflags.
	Commit = "unknown"
	// Date é a data do build; pode ser sobrescrita com -ldflags.
	Date = "unknown"
)

// String devolve a versão com os metadados disponíveis.
func String() string {
	details := make([]string, 0, 2)
	if Commit != "" && Commit != "unknown" {
		details = append(details, "commit "+Commit)
	}
	if Date != "" && Date != "unknown" {
		details = append(details, Date)
	}
	if len(details) == 0 {
		return Version
	}
	return fmt.Sprintf("%s (%s)", Version, strings.Join(details, ", "))
}
