import typescript from '@rollup/plugin-typescript'
import packageJson from './package.json' assert { type: 'json' }

export default [
  {
    input: 'index.ts',
    external: (id) => !/^[./]/.test(id),
    output: [
      {
        file: packageJson.exports.require,
        format: 'cjs',
        sourcemap: true,
      },
      {
        file: packageJson.exports.import,
        format: 'es',
        sourcemap: true,
      },
    ],
    plugins: [typescript({ outputToFilesystem: true })],
  },
]
