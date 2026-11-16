package books

import "testing"

func TestStatusForPage(t *testing.T) {
	// Os valores esperados são as strings literais (e não as constantes)
	// de propósito: é esse texto que vai para o banco e para o JSON.
	// Se alguém mudar o valor de uma constante por engano, o teste pega.
	tests := []struct {
		name        string
		currentPage int32
		totalPages  int32
		want        string
	}{
		{name: "página 0 ainda não começou", currentPage: 0, totalPages: 300, want: "wanna_read"},
		{name: "página 1 já está lendo", currentPage: 1, totalPages: 300, want: "reading"},
		{name: "meio do livro está lendo", currentPage: 150, totalPages: 300, want: "reading"},
		{name: "penúltima página ainda está lendo", currentPage: 299, totalPages: 300, want: "reading"},
		{name: "última página terminou", currentPage: 300, totalPages: 300, want: "read"},
		{name: "livro de 1 página na página 0", currentPage: 0, totalPages: 1, want: "wanna_read"},
		{name: "livro de 1 página na página 1", currentPage: 1, totalPages: 1, want: "read"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := statusForPage(tt.currentPage, tt.totalPages)
			if got != tt.want {
				t.Errorf("statusForPage(%d, %d) = %q, quero %q", tt.currentPage, tt.totalPages, got, tt.want)
			}
		})
	}
}
