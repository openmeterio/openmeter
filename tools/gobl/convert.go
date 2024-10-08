package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/Masterminds/sprig/v3"
	"github.com/invopop/gobl/data"
	"github.com/openmeterio/openmeter/tools/gobl/pkg/schematype"
	"github.com/openmeterio/openmeter/tools/gobl/pkg/workqueue"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type schemaLoader struct {
	files map[string][]byte
}

func loadGoblSchema() (*schemaLoader, error) {
	loader := &schemaLoader{
		files: make(map[string][]byte),
	}
	return loader, loader.registerAllSchemas(data.Content, "schemas")
}

func (sl *schemaLoader) registerAllSchemas(fs embed.FS, prefix string) error {
	dirents, err := fs.ReadDir(prefix)
	if err != nil {
		return err
	}

	for _, dirent := range dirents {
		if dirent.IsDir() {
			if err := sl.registerAllSchemas(fs, prefix+"/"+dirent.Name()); err != nil {
				return err
			}
		} else {
			if !strings.HasSuffix(dirent.Name(), ".json") {
				continue
			}
			if err := sl.registerSchema(fs, prefix+"/"+dirent.Name()); err != nil {
				return err
			}
		}
	}

	return nil
}

type JSONSchemaHeader struct {
	ID string `json:"$id"`
}

func (sl *schemaLoader) registerSchema(fs embed.FS, path string) error {
	file, err := fs.Open(path)
	if err != nil {
		return err
	}

	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	// let's extract the $id field
	var header JSONSchemaHeader
	if err := json.Unmarshal(content, &header); err != nil {
		return err
	}

	if header.ID == "" {
		return fmt.Errorf("missing $id field in %s", path)
	}

	fmt.Printf("loading %s=>%s\n", path, header.ID)

	sl.files[header.ID] = content
	return nil
}

func (sl *schemaLoader) Load(url string) (any, error) {
	content, ok := sl.files[url]
	if !ok {
		return nil, fmt.Errorf("schema %s not found", url)
	}

	log.Printf("loading %s\n", url)

	return jsonschema.UnmarshalJSON(strings.NewReader(string(content)))
}

type TemplateIn struct {
	Properties []Property
	Type       schematype.TypeRef
}

type Property struct {
	Name       string
	Type       schematype.TypeRef
	TypeString string
	Required   bool
}

func ToProperties(schema *jsonschema.Schema) []Property {
	var properties []Property
	for name, prop := range schema.Properties {

		entType, err := schematype.NewFromSchema(name, prop)
		if err != nil {
			panic(err)
		}

		ts, err := entType.String()
		if err != nil {
			log.Fatal(err)
		}

		properties = append(properties, Property{
			Name:       name,
			Type:       *entType,
			TypeString: ts,
			Required:   slices.Contains(schema.Required, name),
		})
	}
	return properties
}

func main() {
	schemaURL := "https://gobl.org/draft-0/bill/invoice"

	schema, err := loadGoblSchema()
	if err != nil {
		log.Fatal(err)
	}

	loader := jsonschema.SchemeURLLoader{
		"file":  schema,
		"http":  schema,
		"https": schema,
	}

	c := jsonschema.NewCompiler()
	c.UseLoader(loader)
	sch, err := c.Compile(schemaURL)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(sch.Properties)

	fmt.Println(sch.Ref.Properties)

	tplFile, err := os.Open("template/main.tpl")
	if err != nil {
		panic(err)
	}

	tplContent, err := io.ReadAll(tplFile)
	if err != nil {
		panic(err)
	}

	tpl := template.Must(
		template.New("template/main").Funcs(sprig.FuncMap()).Parse(string(tplContent)),
	)

	f, err := os.Create("out.txt")
	if err != nil {
		panic(err)
	}

	defer f.Close()

	queue := workqueue.New()
	initialType, err := schematype.NewFromSchema("invoice", sch)
	queue.Add(initialType)

	nIter := 0

	for next := queue.Next(); next != nil; next = queue.Next() {
		nIter++
		if nIter > 10000 {
			panic("too many iterations")
		}

		fmt.Printf("iteration %d: %s\n", nIter, next.Location)

		props := ToProperties(next.GetTargetRef())

		in := TemplateIn{
			Properties: props,
			Type:       *next,
		}

		err = tpl.Execute(f, in)
		if err != nil {
			fmt.Printf("%s=>%s\n", next.Location, err.Error())
			log.Fatal(err)
		}

		for _, prop := range props {
			if targetRef := prop.Type.GetTargetRef(); targetRef != nil {
				queue.Add(&prop.Type)
			}
		}

		queue.Emitted(next)

	}

	/*
		sl := gojsonschema.NewSchemaLoader()

		mainSchemaLoader := gojsonschema.NewReferenceLoader("file:///Users/turip/src/invopop-gobl/data/schemas/bill/invoice.json")

		err := sl.AddSchema("invoice", mainSchemaLoader)
		if err != nil {
			panic(err.Error())
		}

		if err := registerGoblSchema(sl); err != nil {
			panic(err.Error())
		}

		schema, err := sl.Compile(mainSchemaLoader)
		if err != nil {
			panic(err.Error())
		}
		fmt.Println(schema)
	*/
	/*documentLoader := gojsonschema.NewReferenceLoader("file:///home/me/document.json")

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		panic(err.Error())
	}

	if result.Valid() {
		fmt.Printf("The document is valid\n")
	} else {
		fmt.Printf("The document is not valid. see errors :\n")
		for _, desc := range result.Errors() {
			fmt.Printf("- %s\n", desc)
		}
	}*/
}
