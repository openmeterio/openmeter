import * as ay from '@alloy-js/core'
import * as go from '@alloy-js/go'
import type { Refkey } from '@alloy-js/core'
import { strings, url } from '../stdlib.js'

export interface GoClientResource {
  name: string
  root: string
  nestPath: string[]
  serviceRefkey: Refkey
}

export interface GoClientProps {
  clientRefkey: Refkey
  resources: GoClientResource[]
}

export function GoClient({ clientRefkey, resources }: GoClientProps) {
  const roots = resources.filter((resource) => resource.nestPath.length === 0)
  const wiring = [...resources]
    .sort((left, right) => left.nestPath.length - right.nestPath.length)
    .map((resource) => {
      const target = ['c', resource.root, ...resource.nestPath].join('.')
      return `${target} = &${resource.name}Service{client: c}`
    })
    .join('\n')

  const receiver = () => (
    <go.FunctionReceiver
      name="c"
      type={
        <go.Pointer>
          <go.Reference refkey={clientRefkey} />
        </go.Pointer>
      }
    />
  )

  return (
    <ay.List joiner={'\n\n'}>
      <go.StructTypeDeclaration name="Client" refkey={clientRefkey}>
        <ay.List hardline>
          <go.StructMember
            name="baseURL"
            type={<go.Pointer>{url.URL}</go.Pointer>}
          />
          <go.StructMember
            name="httpClient"
            type={<go.Pointer>{go.std.net.http.Client}</go.Pointer>}
          />
          <go.StructMember name="token" type="string" />
          <go.StructMember name="userAgent" type="string" />
          <go.StructMember name="requestEditors" type="[]RequestEditorFn" />
          <ay.List hardline>
            {roots.map((resource) => (
              <go.StructMember
                name={resource.root}
                type={
                  <go.Pointer>
                    <go.Reference refkey={resource.serviceRefkey} />
                  </go.Pointer>
                }
              />
            ))}
          </ay.List>
        </ay.List>
      </go.StructTypeDeclaration>
      <go.FunctionDeclaration
        name="New"
        parameters={[
          { name: 'baseURL', type: 'string' },
          { name: 'opts', type: 'Option', variadic: true },
        ]}
        returns={['*Client', 'error']}
      >
        {ay.code`
          if baseURL == "" {
            return nil, ${go.std.fmt.Errorf}("openmeter: baseURL is required")
          }

          u, err := ${url.Parse}(baseURL)
          if err != nil {
            return nil, ${go.std.fmt.Errorf}("openmeter: invalid baseURL %q: %w", baseURL, err)
          }

          if u.Scheme == "" || u.Host == "" {
            return nil, ${go.std.fmt.Errorf}("openmeter: baseURL %q must be absolute (scheme and host)", baseURL)
          }

          c := &Client{
            baseURL: u,
            userAgent: defaultUserAgent,
          }

          for _, opt := range opts {
            opt(c)
          }

          if c.httpClient == nil {
            c.httpClient = defaultHTTPClient()
          }

          ${wiring}

          return c, nil
        `}
      </go.FunctionDeclaration>
      <ay.List joiner={'\n'}>
        {`// resolve joins the client base URL with an API path, preserving any base path
// prefix and base query present on the base URL. apiPath is parsed so percent
// escapes already present in a segment, such as an ID escaped by
// replacePathParam, are carried through on RawPath instead of being
// re-escaped.`}
        <go.FunctionDeclaration
          name="resolve"
          receiver={receiver()}
          parameters={[{ name: 'apiPath', type: 'string' }]}
          returns={<go.Pointer>{url.URL}</go.Pointer>}
        >
          {ay.code`
          base := *c.baseURL

          if !${strings.HasSuffix}(base.Path, "/") {
            base.Path += "/"
          }

          trimmed := ${strings.TrimPrefix}(apiPath, "/")
          ref, err := ${url.Parse}(trimmed)
          if err != nil {
            ref = &${url.URL}{Path: trimmed}
          }

          resolved := base.ResolveReference(ref)
          resolved.RawQuery = base.RawQuery
          return resolved
        `}
        </go.FunctionDeclaration>
      </ay.List>
      {ay.code`
        func replacePathParam(path, name, value string) string {
          return ${strings.ReplaceAll}(path, "{" + name + "}", ${url.PathEscape}(value))
        }
      `}
    </ay.List>
  )
}
