package entity

import "testing"

func TestNormalizeRankForStorage(t *testing.T) {
	cases := map[string]string{
		"A+": "A_P", "B+": "B_P", "C+": "C_P", "D+": "D_P",
		"A_P": "A_P", "B_P": "B_P",
		"A": "A", "S": "S", "SS": "SS", "SSS": "SSS", "F": "F",
		"": "", "Z+": "Z+",
	}
	for in, want := range cases {
		if got := NormalizeRankForStorage(in); got != want {
			t.Errorf("Storage(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeRankForCoefficient(t *testing.T) {
	cases := map[string]string{
		"A_P": "A+", "B_P": "B+", "C_P": "C+", "D_P": "D+",
		"A+": "A+", "B+": "B+",
		"A": "A", "S": "S", "SS": "SS", "SSS": "SSS", "F": "F",
		"": "", "Z_P": "Z_P",
	}
	for in, want := range cases {
		if got := NormalizeRankForCoefficient(in); got != want {
			t.Errorf("Coefficient(%q) = %q, want %q", in, got, want)
		}
	}
}

// 두 표기 어느 쪽으로 들어와도 저장·EXP 양쪽이 각자 올바른 표기를 얻는지.
func TestBothNotationsConverge(t *testing.T) {
	for _, in := range []string{"A+", "A_P"} {
		if got := NormalizeRankForStorage(in); got != "A_P" {
			t.Errorf("Storage(%q) = %q, want A_P", in, got)
		}
		if got := NormalizeRankForCoefficient(in); got != "A+" {
			t.Errorf("Coefficient(%q) = %q, want A+", in, got)
		}
	}
}
