package pdfnav

import "testing"

func TestDecodeString(t *testing.T) {
	cases := map[string]string{
		`(Hola \(mundo\))`:         "Hola (mundo)",
		`<FEFF00CD006E0064006900>`: "Índi",
		`(caf\351)`:                "café",
	}
	for in, want := range cases {
		if got := decodeString([]byte(in)); got != want {
			t.Errorf("decodeString(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPDFStringRoundTrip(t *testing.T) {
	for _, s := range []string{"Portada", "Introducción", "a(b)c"} {
		if got := decodeString([]byte(pdfString(s))); got != s {
			t.Errorf("round trip %q -> %q", s, got)
		}
	}
}

func TestOutlineRejectsNonPDF(t *testing.T) {
	if _, err := Outline([]byte("no es un pdf")); err == nil {
		t.Fatal("se esperaba error")
	}
}
