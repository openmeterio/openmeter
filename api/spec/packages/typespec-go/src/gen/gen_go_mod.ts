export function emitGoMod(module: string): { path: string; content: string } {
  // Matches the reference SDK's go.mod exactly (only testify, only for tests).
  const content = `module ${module}

go 1.22

require github.com/stretchr/testify v1.11.1

require (
\tgithub.com/davecgh/go-spew v1.1.1 // indirect
\tgithub.com/pmezard/go-difflib v1.0.0 // indirect
\tgopkg.in/yaml.v3 v3.0.1 // indirect
)
`;
  return { path: "go.mod", content };
}
