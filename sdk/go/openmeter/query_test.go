package openmeter

import "testing"

func TestMeterListParams_Values(t *testing.T) {
	tests := []struct {
		name    string
		params  MeterListParams
		want    string
		wantErr bool
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
			name: "filter comma-list operators",
			params: MeterListParams{Filter: &MeterFilter{Key: &StringFilter{
				Oeq:       []string{"a", "b"},
				Ocontains: []string{"c", "d"},
			}}},
			want: "filter%5Bkey%5D%5Bocontains%5D=c%2Cd&filter%5Bkey%5D%5Boeq%5D=a%2Cb",
		},
		{
			name: "combined styles",
			params: MeterListParams{
				Page:   &PageParams{Size: Int(25)},
				Sort:   []string{"key"},
				Filter: &MeterFilter{Name: &StringFilter{Contains: String("gpt"), Exists: Bool(true)}},
			},
			want: "filter%5Bname%5D%5B%24exists%5D=true&filter%5Bname%5D%5Bcontains%5D=gpt&page%5Bsize%5D=25&sort=key",
		},
		{
			name: "comma in one-of value is rejected",
			params: MeterListParams{Filter: &MeterFilter{Key: &StringFilter{
				Oeq: []string{"a,b", "c"},
			}}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.params.values()

			if tt.wantErr {
				if err == nil {
					t.Fatalf("values() error = nil, want an error")
				}
				return
			}

			if err != nil {
				t.Fatalf("values() unexpected error: %v", err)
			}

			if got := got.Encode(); got != tt.want {
				t.Fatalf("values().Encode()\n got: %q\nwant: %q", got, tt.want)
			}
		})
	}
}
