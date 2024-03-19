package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToSQL(t *testing.T) {
	tests := []struct {
		name   string
		field  string
		filter Filter
		want   string
	}{
		// Equality
		{
			name:  "equal number",
			field: "subject",
			filter: toFilter(`{
				"$eq": 1
			}`),
			want: `subject = 1`,
		},
		{
			name:  "equal string",
			field: "subject",
			filter: toFilter(`{
				"$eq": "a"
			}`),
			want: `subject = 'a'`,
		},
		{
			name:  "not equal number",
			field: "subject",
			filter: toFilter(`{
				"$ne": 1
			}`),
			want: `subject != 1`,
		},
		{
			name:  "gt",
			field: "subject",
			filter: toFilter(`{
				"$gt": 2
			}`),
			want: `subject > 2`,
		},
		{
			name:  "gte",
			field: "subject",
			filter: toFilter(`{
				"$gte": 2
			}`),
			want: `subject >= 2`,
		},
		{
			name:  "lt",
			field: "subject",
			filter: toFilter(`{
				"$lt": 2
			}`),
			want: `subject < 2`,
		},
		{
			name:  "lte",
			field: "subject",
			filter: toFilter(`{
				"$lte": 2
			}`),
			want: `subject <= 2`,
		},
		{
			name:  "match",
			field: "subject",
			filter: toFilter(`{
				"$match": "[0-9]+"
			}`),
			want: `match(subject, /[0-9]+/)`,
		},
		{
			name:  "like",
			field: "subject",
			filter: toFilter(`{
				"$like": "%abc%"
			}`),
			want: `subject LIKE '%abc%'`,
		},
		{
			name:  "notLike",
			field: "subject",
			filter: toFilter(`{
				"$notLike": "%abc%"
			}`),
			want: `subject NOT LIKE '%abc%'`,
		},
		{
			name:  "in",
			field: "subject",
			filter: toFilter(`{
						"$in": [1, 2]
					}`),
			want: `subject IN (1, 2)`,
		},
		{
			name:  "nin",
			field: "subject",
			filter: toFilter(`{
						"$nin": [1, 2]
					}`),
			want: `subject NOT IN (1, 2)`,
		},
		// Controls
		{
			name:  "not",
			field: "subject",
			filter: toFilter(`{
				"$not": {
                    "$eq": 1
                }
			}`),
			want: "NOT (subject = 1)",
		},
		{
			name:  "and",
			field: "subject",
			filter: toFilter(`{
                "$and": [
                  {
                    "$eq": 1
                  },
                  {
                    "$eq": 2
                  }
                ]
              }`),
			want: "(subject = 1 AND subject = 2)",
		},
		{
			name:  "or",
			field: "subject",
			filter: toFilter(`{
                "$or": [
                  {
                    "$eq": 1
                  },
                  {
                    "$eq": 2
                  }
                ]
              }`),
			want: "(subject = 1 OR subject = 2)",
		},
		// Complex
		{
			name:  "complex",
			field: "subject",
			filter: toFilter(`{
                "$and": [
                  {
                    "$or": [
                      {
                        "$in": [1, 2, 3]
                      },
                      {
                        "$nin": [4, 5, 6]
                      }
                    ]
                  },
                  {
                    "$eq": 2
                  }
                ]
              }`),
			want: "((subject IN (1, 2, 3) OR subject NOT IN (4, 5, 6)) AND subject = 2)",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToSQL(tt.field, tt.filter)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func toFilter(s string) Filter {
	filter, err := ToFilter(s)
	if err != nil {
		panic(err)
	}
	return filter
}
