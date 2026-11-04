package week

import "testing"

func TestParseHabitLine(t *testing.T) {
	cases := []struct {
		name    string
		line    string
		checked bool
		always  bool
		body    string
		desc    string
	}{
		{
			name: "completo",
			line: "- [ ] 5 min (U) - AWS",
			body: "5 min (U) - AWS",
			desc: "AWS",
		},
		{
			name:    "completo concluído",
			line:    "- [x] 5 min (U) - AWS",
			checked: true,
			body:    "5 min (U) - AWS",
			desc:    "AWS",
		},
		{
			name:    "X maiúsculo",
			line:    "- [X] 5 min (U) - AWS",
			checked: true,
			body:    "5 min (U) - AWS",
			desc:    "AWS",
		},
		{
			name: "só tempo",
			line: "- [ ] 5 min - AWS",
			body: "5 min - AWS",
			desc: "AWS",
		},
		{
			name: "só prioridade",
			line: "- [ ] (U) - AWS",
			body: "(U) - AWS",
			desc: "AWS",
		},
		{
			name: "livre",
			line: "- [ ] AWS",
			body: "AWS",
			desc: "AWS",
		},
		{
			name: "xST",
			line: "- [ ] xST () - dormir",
			body: "xST () - dormir",
			desc: "dormir",
		},
		{
			name: "tempo composto",
			line: "- [ ] 5/15 min () - Tempo na área",
			body: "5/15 min () - Tempo na área",
			desc: "Tempo na área",
		},
		{
			name:   "always completo",
			line:   "- [ ] 5 min (U) - AWS #always",
			always: true,
			body:   "5 min (U) - AWS #always",
			desc:   "AWS",
		},
		{
			name:   "always maiúsculo",
			line:   "- [ ] Ler #Always",
			always: true,
			body:   "Ler #Always",
			desc:   "Ler",
		},
		{
			name: "always colado",
			line: "- [ ] Ler#always",
			body: "Ler#always",
			desc: "Ler#always",
		},
		{
			name: "always no meio",
			line: "- [ ] Ler #always depois",
			body: "Ler #always depois",
			desc: "Ler #always depois",
		},
		{
			name: "CRLF",
			line: "- [ ] 5 min (U) - AWS\r",
			body: "5 min (U) - AWS",
			desc: "AWS",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ParseHabitLine(tc.line)
			if !ok {
				t.Fatalf("ParseHabitLine(%q) não reconheceu", tc.line)
			}
			if got.Checked != tc.checked {
				t.Errorf("Checked = %v, want %v", got.Checked, tc.checked)
			}
			if got.Always != tc.always {
				t.Errorf("Always = %v, want %v", got.Always, tc.always)
			}
			if got.Body != tc.body {
				t.Errorf("Body = %q, want %q", got.Body, tc.body)
			}
			if got.Desc != tc.desc {
				t.Errorf("Desc = %q, want %q", got.Desc, tc.desc)
			}
		})
	}
}

func TestParseHabitLineInvalid(t *testing.T) {
	for _, line := range []string{
		"",
		"## Categoria",
		"texto solto",
		"- [ ]",
		"- [y] algo",
		"- [] algo",
	} {
		if got, ok := ParseHabitLine(line); ok {
			t.Errorf("ParseHabitLine(%q) deveria falhar, veio %+v", line, got)
		}
	}
}
