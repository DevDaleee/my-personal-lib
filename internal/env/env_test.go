package env

import (
	"os"
	"testing"
)

func TestGetRequired(t *testing.T) {
	const key = "TEST_GET_REQUIRED"

	tests := []struct {
		name    string
		set     bool   // false = a variável nem existe no ambiente
		value   string // valor usado quando set = true
		want    string // valor esperado no retorno
		wantErr bool   // se esperamos um erro
	}{
		{name: "variável não existe devolve erro", set: false, want: "", wantErr: true},
		{name: "variável existe mas vazia devolve erro", set: true, value: "", want: "", wantErr: true},
		{name: "variável com valor devolve o valor", set: true, value: "host=localhost", want: "host=localhost", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.set {
				// t.Setenv define a variável só durante este teste e restaura depois.
				t.Setenv(key, tt.value)
			} else {
				// Garante que a variável não existe. O Setenv antes registra a
				// restauração automática; o Unsetenv remove de fato.
				t.Setenv(key, "")
				os.Unsetenv(key)
			}

			got, err := GetRequired(key)

			if (err != nil) != tt.wantErr {
				t.Fatalf("erro = %v, queria erro? %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("valor = %q, quero %q", got, tt.want)
			}
		})
	}
}
