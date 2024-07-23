//go:generate quicktype  --src-lang schema  --lang go -o events.gen.go --package events ./schema/
//go:generate quicktype  --src-lang schema  --lang schema -o events-schema.gen.json ./schema/

package events
