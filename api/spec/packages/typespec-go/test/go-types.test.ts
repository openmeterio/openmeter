import { createTestHost, createTestRunner } from '@typespec/compiler/testing'
import { describe, expect, it } from 'vitest'
import { goType } from '../dist/go-types.js'
import type { Program } from '@typespec/compiler'

async function compileTypes(code: string): Promise<Program> {
  const host = await createTestHost()
  const runner = await createTestRunner(host)
  await runner.compile(code)
  return runner.program
}

describe('Go type mapping', () => {
  it('maps unbounded integer scalars to int64 and keeps sized scalars exact', async () => {
    const program = await compileTypes(`
      model Sample {
        version: integer;
        quantity: safeint;
        priority: int16;
        octet: int8;
      }
    `)

    const sample = program.getGlobalNamespaceType().models.get('Sample')!
    const fieldType = (name: string) =>
      goType(program, sample.properties.get(name)!.type).type

    expect(fieldType('version')).toBe('int64')
    expect(fieldType('quantity')).toBe('int64')
    expect(fieldType('priority')).toBe('int16')
    expect(fieldType('octet')).toBe('int8')
  })

  it('maps every known field filter union to its runtime filter type', async () => {
    const program = await compileTypes(`
      union StringFieldFilter { equals: string }
      union StringFieldFilterExact { equals: string }
      union ULIDFieldFilter { equals: string }
      union DateTimeFieldFilter { equals: utcDateTime }
      union NumericFieldFilter { equals: float64 }
      union BooleanFieldFilter { equals: boolean }
    `)

    const unions = program.getGlobalNamespaceType().unions
    const unionType = (name: string) => goType(program, unions.get(name)!).type

    expect(unionType('StringFieldFilter')).toBe('StringFilter')
    expect(unionType('StringFieldFilterExact')).toBe('StringExactFilter')
    expect(unionType('ULIDFieldFilter')).toBe('StringExactFilter')
    expect(unionType('DateTimeFieldFilter')).toBe('DateTimeFilter')
    expect(unionType('NumericFieldFilter')).toBe('NumericFilter')
    expect(unionType('BooleanFieldFilter')).toBe('BooleanFilter')
  })

  it('rejects field filter unions without a runtime filter mapping', async () => {
    const program = await compileTypes(`
      union DurationFieldFilter { equals: string }
    `)

    const union = program
      .getGlobalNamespaceType()
      .unions.get('DurationFieldFilter')!
    expect(() => goType(program, union)).toThrow(
      'field filter union DurationFieldFilter has no runtime filter type',
    )
  })

  it('rejects anonymous unions that mix non-string variants', async () => {
    const program = await compileTypes(`
      model Sample {
        value: int32 | boolean;
      }
    `)

    const value = program
      .getGlobalNamespaceType()
      .models.get('Sample')!
      .properties.get('value')!
    expect(() => goType(program, value.type)).toThrow(
      'anonymous union of [Scalar, Scalar] is not representable in Go',
    )
  })

  it('maps explicit unknown to any but rejects other intrinsics', async () => {
    const program = await compileTypes(`
      model Sample {
        config?: unknown;
        forbidden?: null;
      }
    `)

    const sample = program.getGlobalNamespaceType().models.get('Sample')!
    expect(goType(program, sample.properties.get('config')!.type).type).toBe(
      'any',
    )
    expect(() =>
      goType(program, sample.properties.get('forbidden')!.type),
    ).toThrow('intrinsic type null is not representable in Go')
  })
})
