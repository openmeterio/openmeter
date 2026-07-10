import * as go from '@alloy-js/go'

export const context = go.createModule(
  'context',
  {
    kind: 'package',
    members: {
      Context: { kind: 'interface', members: {} },
    },
  } as const,
  true,
)

export const url = go.createModule(
  'url',
  {
    kind: 'package',
    path: 'net/url',
    members: {
      URL: { kind: 'struct', members: {} },
      Values: {
        kind: 'type',
        members: {
          Add: { kind: 'method' },
          Set: { kind: 'method' },
        },
      },
      Parse: { kind: 'function' },
      PathEscape: { kind: 'function' },
      PathUnescape: { kind: 'function' },
    },
  } as const,
  true,
)

export const iter = go.createModule(
  'iter',
  {
    kind: 'package',
    path: 'iter',
    members: {
      Seq2: { kind: 'type', members: {} },
    },
  } as const,
  true,
)

export const strings = go.createModule(
  'strings',
  {
    kind: 'package',
    path: 'strings',
    members: {
      HasSuffix: { kind: 'function' },
      Join: { kind: 'function' },
      ReplaceAll: { kind: 'function' },
      TrimPrefix: { kind: 'function' },
    },
  } as const,
  true,
)

export const json = go.createModule(
  'json',
  {
    kind: 'package',
    path: 'encoding/json',
    members: {
      Marshal: { kind: 'function' },
      RawMessage: { kind: 'type', members: {} },
      Unmarshal: { kind: 'function' },
    },
  } as const,
  true,
)

export const strconv = go.createModule(
  'strconv',
  {
    kind: 'package',
    members: {
      FormatBool: { kind: 'function' },
      FormatFloat: { kind: 'function' },
      FormatInt: { kind: 'function' },
      FormatUint: { kind: 'function' },
    },
  } as const,
  true,
)
