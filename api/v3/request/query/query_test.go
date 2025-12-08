package query

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name:  "simple key-value",
			input: "a=b",
			want: map[string]interface{}{
				"a": "b",
			},
		},
		{
			name:  "nested object",
			input: "a[b][c]=d",
			want: map[string]interface{}{
				"a": map[string]interface{}{
					"b": map[string]interface{}{
						"c": "d",
					},
				},
			},
		},
		{
			name:  "array",
			input: "a[]=b&a[]=c",
			want: map[string]interface{}{
				"a": []interface{}{"b", "c"},
			},
		},
		{
			name:  "nested array",
			input: "a[b][]=c&a[b][]=d",
			want: map[string]interface{}{
				"a": map[string]interface{}{
					"b": []interface{}{"c", "d"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(t.Context(), tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseComplex(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name:  "deep nested objects",
			input: "user[profile][settings][theme][colors][primary]=blue&user[profile][settings][theme][colors][secondary]=green",
			want: map[string]interface{}{
				"user": map[string]interface{}{
					"profile": map[string]interface{}{
						"settings": map[string]interface{}{
							"theme": map[string]interface{}{
								"colors": map[string]interface{}{
									"primary":   "blue",
									"secondary": "green",
								},
							},
						},
					},
				},
			},
		},
		{
			name:  "array of objects",
			input: "users[0][name]=John&users[0][age]=30&users[1][name]=Jane&users[1][age]=25",
			want: map[string]interface{}{
				"users": map[string]interface{}{
					"0": map[string]interface{}{
						"name": "John",
						"age":  "30",
					},
					"1": map[string]interface{}{
						"name": "Jane",
						"age":  "25",
					},
				},
			},
		},
		{
			name:  "mixed arrays and objects",
			input: "data[users][]=John&data[users][]=Jane&data[settings][theme]=dark&data[config][api][endpoints][]=auth&data[config][api][endpoints][]=users",
			want: map[string]interface{}{
				"data": map[string]interface{}{
					"users": []interface{}{"John", "Jane"},
					"settings": map[string]interface{}{
						"theme": "dark",
					},
					"config": map[string]interface{}{
						"api": map[string]interface{}{
							"endpoints": []interface{}{"auth", "users"},
						},
					},
				},
			},
		},
		{
			name:  "complex e-commerce query",
			input: "products[0][id]=123&products[0][name]=Laptop&products[0][price]=999&products[0][tags][]=electronics&products[0][tags][]=computers&products[0][variants][0][size]=15inch&products[0][variants][0][color]=black&products[1][id]=456&products[1][name]=Mouse&products[1][price]=25&filters[category]=electronics&filters[price][min]=0&filters[price][max]=1000&sort[field]=price&sort[order]=asc",
			want: map[string]interface{}{
				"products": map[string]interface{}{
					"0": map[string]interface{}{
						"id":    "123",
						"name":  "Laptop",
						"price": "999",
						"tags":  []interface{}{"electronics", "computers"},
						"variants": map[string]interface{}{
							"0": map[string]interface{}{
								"size":  "15inch",
								"color": "black",
							},
						},
					},
					"1": map[string]interface{}{
						"id":    "456",
						"name":  "Mouse",
						"price": "25",
					},
				},
				"filters": map[string]interface{}{
					"category": "electronics",
					"price": map[string]interface{}{
						"min": "0",
						"max": "1000",
					},
				},
				"sort": map[string]interface{}{
					"field": "price",
					"order": "asc",
				},
			},
		},
		{
			name:  "empty values and arrays",
			input: "empty=&arr[]=&arr[]=value&nested[empty]=&nested[arr][]=",
			want: map[string]interface{}{
				"empty": "",
				"arr":   []interface{}{"", "value"},
				"nested": map[string]interface{}{
					"empty": "",
					"arr":   []interface{}{""},
				},
			},
		},
		{
			name:  "url encoded values",
			input: "message=Hello%20World&symbols=%21%40%23%24%25&unicode=%D0%9F%D1%80%D0%B8%D0%B2%D0%B5%D1%82",
			want: map[string]interface{}{
				"message": "Hello World",
				"symbols": "!@#$%",
				"unicode": "Привет",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(t.Context(), tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseWithOptions(t *testing.T) {
	ctx := t.Context()
	// Test with custom delimiter
	result, err := Parse(ctx, "name=John;age=30;city=NYC", &ParseOptions{
		Delimiter: ";",
	})
	if err != nil {
		t.Errorf("Parse with custom delimiter failed: %v", err)
	}

	expected := map[string]interface{}{
		"name": "John",
		"age":  "30",
		"city": "NYC",
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Parse with custom delimiter = %v, want %v", result, expected)
	}

	// Test with parameter limit
	longQuery := strings.Repeat("param=value&", 1001)
	longQuery = strings.TrimSuffix(longQuery, "&")

	_, err = Parse(ctx, longQuery, &ParseOptions{
		ParameterLimit:       100,
		ThrowOnLimitExceeded: true,
	})
	if err == nil {
		t.Error("Expected error when exceeding parameter limit")
	}

	// Test without throwing on limit exceeded
	result2, err := Parse(ctx, longQuery, &ParseOptions{
		ParameterLimit:       100,
		ThrowOnLimitExceeded: false,
	})
	if err != nil {
		t.Errorf("Parse without throwing on limit exceeded failed: %v", err)
	}
	if len(result2) != 1 { // Should only have one key "param" with last value
		t.Errorf("Expected 1 key in result, got %d", len(result2))
	}
}

func TestStringify(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]interface{}
		want    string
		wantErr bool
	}{
		{
			name: "simple object",
			input: map[string]interface{}{
				"a": "b",
				"c": "d",
			},
			want: "a=b&c=d",
		},
		{
			name: "nested object",
			input: map[string]interface{}{
				"a": map[string]interface{}{
					"b": "c",
				},
			},
			want: "a[b]=c",
		},
		{
			name: "array",
			input: map[string]interface{}{
				"a": []interface{}{"b", "c"},
			},
			want: "a[0]=b&a[1]=c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Stringify(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Stringify() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Since order is not guaranteed in maps, we check if all expected parts are present
			if !tt.wantErr {
				parts := strings.Split(tt.want, "&")
				for _, part := range parts {
					if !strings.Contains(got, part) {
						t.Errorf("Stringify() = %v, should contain %v", got, part)
					}
				}
			}
		})
	}
}

func TestStringifyComplex(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]interface{}
		want    []string // Parts that should be present
		wantErr bool
	}{
		{
			name: "deep nested structure",
			input: map[string]interface{}{
				"user": map[string]interface{}{
					"profile": map[string]interface{}{
						"name": "John",
						"settings": map[string]interface{}{
							"theme": "dark",
						},
					},
				},
			},
			want: []string{"user[profile][name]=John", "user[profile][settings][theme]=dark"},
		},
		{
			name: "various data types",
			input: map[string]interface{}{
				"string": "hello",
				"number": 42,
				"bool":   true,
				"array":  []interface{}{"a", "b"},
			},
			want: []string{"string=hello", "number=42", "bool=true", "array[0]=a", "array[1]=b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Stringify(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Stringify() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				for _, want := range tt.want {
					if !strings.Contains(got, want) {
						t.Errorf("Stringify() = %v, should contain %v", got, want)
					}
				}
			}
		})
	}
}

func TestStringifyWithOptionsSimple(t *testing.T) {
	input := map[string]interface{}{
		"name": "John Doe",
		"age":  30,
	}

	// Test with query prefix
	result, err := Stringify(input, &StringifyOptions{
		AddQueryPrefix: true,
	})
	if err != nil {
		t.Errorf("Stringify with AddQueryPrefix failed: %v", err)
	}
	if !strings.HasPrefix(result, "?") {
		t.Errorf("Expected result to start with '?', got %s", result)
	}

	// Test with custom delimiter
	result2, err := Stringify(input, &StringifyOptions{
		Delimiter: ";",
	})
	if err != nil {
		t.Errorf("Stringify with custom delimiter failed: %v", err)
	}
	if !strings.Contains(result2, ";") {
		t.Errorf("Expected result to contain ';', got %s", result2)
	}
}

// Test structures for struct parsing
type User struct {
	Name     string  `query:"name"`
	Age      int     `query:"age"`
	Email    string  `query:"email"`
	IsActive bool    `query:"active"`
	Score    float64 `query:"score"`
}

type SearchFilter struct {
	Query    string   `query:"q"`
	Tags     []string `query:"tags"`
	Category string   `query:"category"`
	MinPrice int      `query:"min_price"`
	MaxPrice int      `query:"max_price"`
}

type NestedStruct struct {
	User     User              `query:"user"`
	Settings map[string]string `query:"settings"`
	Enabled  bool              `query:"enabled"`
}

type ProductVariant struct {
	Size  string `query:"size"`
	Color string `query:"color"`
}

type Product struct {
	ID       int              `query:"id"`
	Name     string           `query:"name"`
	Price    float64          `query:"price"`
	Tags     []string         `query:"tags"`
	Variants []ProductVariant `query:"variants"`
}

type TextUnmarshalStruct struct {
	Value string
}

func (t *TextUnmarshalStruct) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return errors.New("empty text")
	}

	t.Value = strings.ToUpper(string(text))
	return nil
}

type WrapperWithTextUnmarshaler struct {
	Custom TextUnmarshalStruct `query:"custom"`
}

type WrapperWithPointerTextUnmarshaler struct {
	Custom *TextUnmarshalStruct `query:"custom"`
}

func TestParseToStruct(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		dest     interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name:  "simple user struct",
			input: "name=John&age=30&email=john@example.com&active=true&score=95.5",
			dest:  &User{},
			expected: &User{
				Name:     "John",
				Age:      30,
				Email:    "john@example.com",
				IsActive: true,
				Score:    95.5,
			},
		},
		{
			name:  "search filter with arrays",
			input: "q=golang&tags[]=programming&tags[]=web&category=tech&min_price=10&max_price=100",
			dest:  &SearchFilter{},
			expected: &SearchFilter{
				Query:    "golang",
				Tags:     []string{"programming", "web"},
				Category: "tech",
				MinPrice: 10,
				MaxPrice: 100,
			},
		},
		{
			name:  "nested struct",
			input: "user[name]=Alice&user[age]=25&user[email]=alice@test.com&user[active]=false&user[score]=88.0&settings[theme]=dark&settings[lang]=en&enabled=true",
			dest:  &NestedStruct{},
			expected: &NestedStruct{
				User: User{
					Name:     "Alice",
					Age:      25,
					Email:    "alice@test.com",
					IsActive: false,
					Score:    88.0,
				},
				Settings: map[string]string{
					"theme": "dark",
					"lang":  "en",
				},
				Enabled: true,
			},
		},
		{
			name:  "struct field using UnmarshalText",
			input: "custom=hello world",
			dest:  &WrapperWithTextUnmarshaler{},
			expected: &WrapperWithTextUnmarshaler{
				Custom: TextUnmarshalStruct{Value: "HELLO WORLD"},
			},
		},
		{
			name:  "pointer field using UnmarshalText",
			input: "custom=go test",
			dest:  &WrapperWithPointerTextUnmarshaler{},
			expected: &WrapperWithPointerTextUnmarshaler{
				Custom: &TextUnmarshalStruct{Value: "GO TEST"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ParseToStruct(t.Context(), tt.input, tt.dest)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseToStruct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(tt.dest, tt.expected) {
				t.Errorf("ParseToStruct() = %+v, want %+v", tt.dest, tt.expected)
			}
		})
	}
}

func TestStructToQueryString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		contains []string // Check if result contains these substrings instead of exact match
		wantErr  bool
	}{
		{
			name: "simple user struct",
			input: &User{
				Name:     "John",
				Age:      30,
				Email:    "john@example.com",
				IsActive: true,
				Score:    95.5,
			},
			contains: []string{"active=true", "age=30", "email=john%40example.com", "name=John", "score=95.5"},
		},
		{
			name: "search filter with arrays",
			input: &SearchFilter{
				Query:    "golang programming",
				Tags:     []string{"web", "api"},
				Category: "tech",
				MinPrice: 10,
				MaxPrice: 100,
			},
			contains: []string{
				"category=tech", "max_price=100", "min_price=10",
				"q=golang%20programming", "tags[0]=web", "tags[1]=api",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StructToQueryString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("StructToQueryString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				for _, substr := range tt.contains {
					if !strings.Contains(got, substr) {
						t.Errorf("StructToQueryString() = %v, expected to contain %v", got, substr)
					}
				}
			}
		})
	}
}

func TestMapToStruct(t *testing.T) {
	data := map[string]interface{}{
		"name":   "Bob",
		"age":    "35",
		"email":  "bob@test.com",
		"active": "true",
		"score":  "92.3",
	}

	var user User
	err := MapToStruct(t.Context(), data, &user)
	if err != nil {
		t.Fatalf("MapToStruct() error = %v", err)
	}

	expected := User{
		Name:     "Bob",
		Age:      35,
		Email:    "bob@test.com",
		IsActive: true,
		Score:    92.3,
	}

	if !reflect.DeepEqual(user, expected) {
		t.Errorf("MapToStruct() = %+v, want %+v", user, expected)
	}
}

func TestStructToMap(t *testing.T) {
	user := &User{
		Name:     "Charlie",
		Age:      28,
		Email:    "charlie@example.com",
		IsActive: false,
		Score:    78.9,
	}

	got, err := StructToMap(user)
	if err != nil {
		t.Fatalf("StructToMap() error = %v", err)
	}

	expected := map[string]interface{}{
		"name":   "Charlie",
		"age":    28,
		"email":  "charlie@example.com",
		"active": false,
		"score":  78.9,
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("StructToMap() = %+v, want %+v", got, expected)
	}
}

func BenchmarkParseSimple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Parse(context.Background(), "a=b&c=d&e=f")
	}
}

func BenchmarkParseComplex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Parse(context.Background(), "user[profile][settings][theme][colors][primary]=blue&user[profile][settings][theme][colors][secondary]=green&data[users][]=John&data[users][]=Jane")
	}
}

func BenchmarkStringifySimple(b *testing.B) {
	obj := map[string]interface{}{
		"a": "b",
		"c": "d",
		"e": "f",
	}

	for i := 0; i < b.N; i++ {
		Stringify(obj)
	}
}

func BenchmarkStringifyComplex(b *testing.B) {
	obj := map[string]interface{}{
		"products": map[string]interface{}{
			"0": map[string]interface{}{
				"id":    "123",
				"name":  "Laptop",
				"price": "999",
				"tags":  []interface{}{"electronics", "computers"},
				"variants": map[string]interface{}{
					"0": map[string]interface{}{
						"size":  "15inch",
						"color": "black",
					},
				},
			},
			"1": map[string]interface{}{
				"id":    "456",
				"name":  "Mouse",
				"price": "25",
			},
		},
		"filters": map[string]interface{}{
			"category": "electronics",
			"price": map[string]interface{}{
				"min": "0",
				"max": "1000",
			},
		},
		"sort": map[string]interface{}{
			"field": "price",
			"order": "asc",
		},
	}

	for i := 0; i < b.N; i++ {
		_, _ = Stringify(obj)
	}
}

func BenchmarkParseToStruct(b *testing.B) {
	queryString := "name=John&age=30&email=john@example.com&active=true&score=95.5"

	for i := 0; i < b.N; i++ {
		var user User
		_ = ParseToStruct(context.Background(), queryString, &user)
	}
}

func BenchmarkStructToQueryString(b *testing.B) {
	user := &User{
		Name:     "John",
		Age:      30,
		Email:    "john@example.com",
		IsActive: true,
		Score:    95.5,
	}

	for i := 0; i < b.N; i++ {
		_, _ = StructToQueryString(user)
	}
}

// Tests for Marshal/Unmarshal functions
func TestUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		target   interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name:   "unmarshal to struct",
			query:  "name=John&age=30&email=john@example.com&active=true&score=95.5",
			target: &User{},
			expected: &User{
				Name:     "John",
				Age:      30,
				Email:    "john@example.com",
				IsActive: true,
				Score:    95.5,
			},
		},
		{
			name:   "unmarshal to map",
			query:  "name=John&age=30&city=NYC",
			target: &map[string]interface{}{},
			expected: &map[string]interface{}{
				"name": "John",
				"age":  "30",
				"city": "NYC",
			},
		},
		{
			name:   "unmarshal to struct with nested map",
			query:  "user[name]=Alice&user[age]=25&settings[theme]=dark&settings[lang]=en&enabled=true",
			target: &NestedStruct{},
			expected: &NestedStruct{
				User: User{
					Name: "Alice",
					Age:  25,
				},
				Settings: map[string]string{
					"theme": "dark",
					"lang":  "en",
				},
				Enabled: true,
			},
		},
		{
			name:   "unmarshal to struct with slice",
			query:  "q=golang&tags[]=programming&tags[]=web&category=tech&min_price=10&max_price=100",
			target: &SearchFilter{},
			expected: &SearchFilter{
				Query:    "golang",
				Tags:     []string{"programming", "web"},
				Category: "tech",
				MinPrice: 10,
				MaxPrice: 100,
			},
		},
		{
			name:   "unmarshal to map with complex structure",
			query:  "users[0][name]=John&users[0][age]=30&users[1][name]=Jane&users[1][age]=25&metadata[total]=2",
			target: &map[string]interface{}{},
			expected: &map[string]interface{}{
				"users": map[string]interface{}{
					"0": map[string]interface{}{
						"name": "John",
						"age":  "30",
					},
					"1": map[string]interface{}{
						"name": "Jane",
						"age":  "25",
					},
				},
				"metadata": map[string]interface{}{
					"total": "2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Unmarshal(t.Context(), tt.query, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(tt.target, tt.expected) {
				t.Errorf("Unmarshal() = %+v, want %+v", tt.target, tt.expected)
			}
		})
	}
}

func TestMarshal(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		contains []string // Check if result contains these substrings
		wantErr  bool
	}{
		{
			name: "marshal struct",
			input: User{
				Name:     "John",
				Age:      30,
				Email:    "john@example.com",
				IsActive: true,
				Score:    95.5,
			},
			contains: []string{"name=John", "age=30", "active=true", "score=95.5"},
		},
		{
			name: "marshal map",
			input: map[string]interface{}{
				"name": "Alice",
				"age":  25,
				"city": "NYC",
			},
			contains: []string{"name=Alice", "age=25", "city=NYC"},
		},
		{
			name: "marshal struct with slice",
			input: SearchFilter{
				Query:    "golang",
				Tags:     []string{"programming", "web"},
				Category: "tech",
				MinPrice: 10,
				MaxPrice: 100,
			},
			contains: []string{"q=golang", "tags[0]=programming", "tags[1]=web", "category=tech", "min_price=10", "max_price=100"},
		},
		{
			name: "marshal nested struct",
			input: NestedStruct{
				User: User{
					Name: "Bob",
					Age:  35,
				},
				Settings: map[string]string{
					"theme": "light",
					"lang":  "en",
				},
				Enabled: true,
			},
			contains: []string{"user[name]=Bob", "user[age]=35", "settings[theme]=light", "settings[lang]=en", "enabled=true"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Marshal(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				for _, substr := range tt.contains {
					if !strings.Contains(got, substr) {
						t.Errorf("Marshal() = %v, expected to contain %v", got, substr)
					}
				}
			}
		})
	}
}

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{
			name: "simple struct",
			input: User{
				Name:     "John",
				Age:      30,
				Email:    "john@example.com",
				IsActive: true,
				Score:    95.5,
			},
		},
		{
			name: "struct with slice",
			input: SearchFilter{
				Query:    "golang",
				Tags:     []string{"programming", "web", "backend"},
				Category: "tech",
				MinPrice: 10,
				MaxPrice: 100,
			},
		},
		{
			name: "nested struct",
			input: NestedStruct{
				User: User{
					Name:     "Alice",
					Age:      25,
					Email:    "alice@example.com",
					IsActive: false,
					Score:    88.0,
				},
				Settings: map[string]string{
					"theme": "dark",
					"lang":  "ru",
				},
				Enabled: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			// Marshal to query string
			queryString, err := Marshal(tt.input)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			// Create a new instance of the same type
			targetType := reflect.TypeOf(tt.input)
			target := reflect.New(targetType).Interface()

			// Unmarshal back
			if apiErr := Unmarshal(ctx, queryString, target); apiErr != nil {
				t.Fatalf("Unmarshal() error = %v (%T) underlying=%#v invalid=%#v", apiErr, apiErr, apiErr.Unwrap(), apiErr.InvalidParameters)
			}

			// Compare (dereference pointer)
			targetValue := reflect.ValueOf(target).Elem().Interface()
			if !reflect.DeepEqual(tt.input, targetValue) {
				t.Errorf("Round trip failed: original = %+v, result = %+v", tt.input, targetValue)
			}
		})
	}
}

func TestUnmarshalErrors(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		target  interface{}
		wantErr bool
	}{
		{
			name:    "nil target",
			query:   "name=John",
			target:  nil,
			wantErr: true,
		},
		{
			name:    "non-pointer target",
			query:   "name=John",
			target:  User{},
			wantErr: true,
		},
		{
			name:    "unsettable target",
			query:   "name=John",
			target:  (*User)(nil),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Unmarshal(t.Context(), tt.query, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func BenchmarkMarshal(b *testing.B) {
	user := User{
		Name:     "John",
		Age:      30,
		Email:    "john@example.com",
		IsActive: true,
		Score:    95.5,
	}

	for i := 0; i < b.N; i++ {
		_, _ = Marshal(user)
	}
}

func BenchmarkUnmarshal(b *testing.B) {
	queryString := "name=John&age=30&email=john@example.com&active=true&score=95.5"

	for i := 0; i < b.N; i++ {
		var user User
		_ = Unmarshal(context.Background(), queryString, &user)
	}
}

func BenchmarkMarshalComplex(b *testing.B) {
	nested := NestedStruct{
		User: User{
			Name:     "Alice",
			Age:      25,
			Email:    "alice@example.com",
			IsActive: false,
			Score:    88.0,
		},
		Settings: map[string]string{
			"theme": "dark",
			"lang":  "ru",
		},
		Enabled: true,
	}

	for i := 0; i < b.N; i++ {
		_, _ = Marshal(nested)
	}
}

func BenchmarkUnmarshalComplex(b *testing.B) {
	queryString := "user[name]=Alice&user[age]=25&user[email]=alice@example.com&user[active]=false&user[score]=88.0&settings[theme]=dark&settings[lang]=ru&enabled=true"

	for i := 0; i < b.N; i++ {
		var nested NestedStruct
		_ = Unmarshal(context.Background(), queryString, &nested)
	}
}
