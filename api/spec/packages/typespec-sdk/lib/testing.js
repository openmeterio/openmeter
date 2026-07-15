import { fileURLToPath } from 'node:url'
import { createTestLibrary } from '@typespec/compiler/testing'

export const TypeSpecSdkTestLibrary = createTestLibrary({
  name: '@openmeter/typespec-sdk',
  packageRoot: fileURLToPath(new URL('..', import.meta.url)),
  typespecFileFolder: 'lib',
  jsFileFolder: 'lib',
})
