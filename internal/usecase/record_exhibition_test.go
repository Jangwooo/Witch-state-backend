package usecase

import (
	"testing"

	"github.com/witchs-lounge_backend/internal/domain/entity"
)

func TestIsExhibitionRequest(t *testing.T) {
	cases := []struct {
		name string
		req  *entity.CreateRecordRequest
		want bool
	}{
		{"nil request", nil, false},
		{"nil additional", &entity.CreateRecordRequest{}, false},
		{
			"empty additional",
			&entity.CreateRecordRequest{Additional: map[string]interface{}{}},
			false,
		},
		{
			"marker true",
			&entity.CreateRecordRequest{Additional: map[string]interface{}{"is_exhibition": true}},
			true,
		},
		{
			"marker false",
			&entity.CreateRecordRequest{Additional: map[string]interface{}{"is_exhibition": false}},
			false,
		},
		{
			"marker non-bool (string) is not exhibition",
			&entity.CreateRecordRequest{Additional: map[string]interface{}{"is_exhibition": "true"}},
			false,
		},
		{
			"other keys only",
			&entity.CreateRecordRequest{Additional: map[string]interface{}{"exhibition_name": "alice"}},
			false,
		},
		{
			"marker true with name",
			&entity.CreateRecordRequest{Additional: map[string]interface{}{"is_exhibition": true, "exhibition_name": "alice"}},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isExhibitionRequest(tc.req); got != tc.want {
				t.Fatalf("isExhibitionRequest = %v, want %v", got, tc.want)
			}
		})
	}
}
