package marker

import "testing"

func TestStripAlways(t *testing.T) {
	tests := map[string]struct {
		input  string
		want   string
		always bool
	}{
		"no fim":                {"Fio dental #always", "Fio dental", true},
		"maiúsculo":             {"Fio dental #Always", "Fio dental", true},
		"espaços no fim":        {"Fio dental #always   ", "Fio dental", true},
		"no meio":               {"Ler #always depois", "Ler #always depois", false},
		"colado":                {"Ler#always", "Ler#always", false},
		"sem descrição":         {"#always", "#always", false},
		"sem marcador":          {"Fio dental", "Fio dental", false},
		"marcador repetido":     {"Ler #always #always", "Ler #always", true},
		"espaço antes e depois": {"  Ler #always  ", "Ler", true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, always := StripAlways(tt.input)
			if got != tt.want || always != tt.always {
				t.Errorf("StripAlways(%q) = %q, %v; want %q, %v",
					tt.input, got, always, tt.want, tt.always)
			}
		})
	}
}
