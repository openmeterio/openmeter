package openmeter

import "testing"

func TestMeterListParams_Values(t *testing.T) {
	tests := []struct {
		name   string
		params MeterListParams
		want   string
	}{
		{
			name:   "empty produces no query",
			params: MeterListParams{},
			want:   "",
		},
		{
			name:   "page deepObject",
			params: MeterListParams{Page: &PageParams{Size: Int(10), Number: Int(2)}},
			want:   "page%5Bnumber%5D=2&page%5Bsize%5D=10",
		},
		{
			name:   "sort joined as comma-separated form value",
			params: MeterListParams{Sort: []string{"name", "created_at desc"}},
			want:   "sort=name%2Ccreated_at+desc",
		},
		{
			name:   "filter nested deepObject",
			params: MeterListParams{Filter: &MeterFilter{Key: &StringFilter{Eq: String("tokens")}}},
			want:   "filter%5Bkey%5D%5Beq%5D=tokens",
		},
		{
			name: "filter comparison operators",
			params: MeterListParams{Filter: &MeterFilter{Name: &StringFilter{
				Neq: String("a"),
				Gt:  String("b"),
				Gte: String("c"),
				Lt:  String("d"),
				Lte: String("e"),
			}}},
			want: "filter%5Bname%5D%5Bgt%5D=b&filter%5Bname%5D%5Bgte%5D=c&filter%5Bname%5D%5Blt%5D=d&filter%5Bname%5D%5Blte%5D=e&filter%5Bname%5D%5Bneq%5D=a",
		},
		{
			name: "combined styles",
			params: MeterListParams{
				Page:   &PageParams{Size: Int(25)},
				Sort:   []string{"key"},
				Filter: &MeterFilter{Name: &StringFilter{Contains: String("gpt"), Exists: Bool(true)}},
			},
			want: "filter%5Bname%5D%5Bcontains%5D=gpt&filter%5Bname%5D%5Bexists%5D=true&page%5Bsize%5D=25&sort=key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.params.values().Encode()

			if got != tt.want {
				t.Fatalf("values().Encode()\n got: %q\nwant: %q", got, tt.want)
			}
		})
	}
}
