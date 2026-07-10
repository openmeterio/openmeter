import type { Model, Program, Type, Union } from '@typespec/compiler'
import { createTestHost, createTestRunner } from '@typespec/compiler/testing'
import { HttpTestLibrary } from '@typespec/http/testing'
import { OpenAPITestLibrary } from '@typespec/openapi/testing'
import { describe, expect, it } from 'vitest'
import {
  configureGoProjections,
  configureGoTypeNames,
  goFields,
  goType,
  optionalTypeName,
  resolveGoTypeNames,
  setSyntheticTypeNames,
  type GoProjections,
} from '../dist/go-types.js'
import { collectHttpOperations, jsonBodyOverrides } from '../dist/operations.js'
import { groupOperations } from '../dist/grouping.js'
import {
  computeDivergentTypes,
  computeReachability,
  computeStructuralAliases,
  planDeclarations,
  promoteAnonymousModels,
} from '../dist/projections.js'
import { renderUnion } from '../dist/components/GoModels.js'

interface PlannedProgram {
  program: Program
  projections: GoProjections
  modelTypes: Set<Type>
  namedType: (name: string) => Type
}

/** Mirrors the emitter's planning phase: reachability, name resolution,
 * anonymous-model promotion, divergence, declaration planning, and structural
 * dedupe, then configures the live registry the render components consult. */
async function planFixture(code: string): Promise<PlannedProgram> {
  const host = await createTestHost({
    libraries: [HttpTestLibrary, OpenAPITestLibrary],
  })
  const runner = await createTestRunner(host)
  await runner.compile(`
    import "@typespec/http";
    import "@typespec/openapi";
    using TypeSpec.Http;
    using TypeSpec.OpenAPI;
    ${code}
  `)
  const program = runner.program
  configureGoTypeNames(program, [])
  const operations = collectHttpOperations(program)
  const groups = groupOperations(operations)
  const bodyOverrides = jsonBodyOverrides(program)
  const reachability = computeReachability(program, groups, bodyOverrides)
  const modelTypes = new Set(
    [...reachability.byResource.values()].flatMap((types) => [...types]),
  )
  resolveGoTypeNames(program, modelTypes)
  setSyntheticTypeNames(program, promoteAnonymousModels(program, modelTypes))
  const divergent = computeDivergentTypes(
    program,
    reachability.readReachable,
    reachability.inputReachable,
  )
  const declarations = planDeclarations(
    program,
    modelTypes,
    reachability.readReachable,
    reachability.inputReachable,
    divergent,
  )
  const aliases = computeStructuralAliases(
    program,
    declarations,
    reachability.readReachable,
    divergent,
  )
  const projections: GoProjections = {
    readReachable: reachability.readReachable,
    inputReachable: reachability.inputReachable,
    divergent,
    aliases,
    declarations,
  }
  configureGoProjections(program, projections)

  const namedType = (name: string): Type => {
    const match = [...modelTypes].find(
      (type) => optionalTypeName(program, type) === name,
    )
    if (!match) {
      throw new Error(`fixture type ${name} not reachable`)
    }
    return match
  }

  return { program, projections, modelTypes, namedType }
}

const declarationNames = (projections: GoProjections): string[] =>
  [...projections.declarations.values()]
    .flatMap((declarations) => declarations.map(({ name }) => name))
    .sort()

describe('payload-context visibility', () => {
  it('drops create/update-only properties from response-reachable models', async () => {
    const { program, projections, namedType } = await planFixture(`
      model App {
        name: string;

        @visibility(Lifecycle.Read)
        masked_api_key: string;

        @visibility(Lifecycle.Create, Lifecycle.Update)
        @secret
        secret_api_key?: string;
      }

      namespace Apps {
        interface Operations {
          @get op get(): App;
        }
      }

      @service
      namespace Test {
        @route("/apps")
        interface Endpoints extends Apps.Operations {}
      }
    `)

    const app = namedType('App') as Model
    expect(projections.declarations.get(app)).toEqual([
      { name: 'App', mode: 'read' },
    ])
    const fieldNames = goFields(program, app).map((field) => field.name)
    expect(fieldNames).toEqual(['Name', 'MaskedAPIKey'])
    // The input rendering of the same model keeps the spec-projected fields.
    const inputNames = goFields(program, app, { mode: 'input' }).map(
      (field) => field.name,
    )
    expect(inputNames).toContain('SecretAPIKey')
  })
})

describe('request-type unification', () => {
  it('emits request-only models once, as the input projection under the natural name', async () => {
    const { program, projections, namedType } = await planFixture(`
      model CreateMeterRequest {
        name: string;
        labels?: Record<string>;
      }

      model Meter {
        id: string;
        name: string;
      }

      namespace Meters {
        interface Operations {
          @post op create(@body body: CreateMeterRequest): Meter;
        }
      }

      @service
      namespace Test {
        @route("/meters")
        interface Endpoints extends Meters.Operations {}
      }
    `)

    const request = namedType('CreateMeterRequest') as Model
    expect(projections.declarations.get(request)).toEqual([
      { name: 'CreateMeterRequest', mode: 'input' },
    ])
    expect(declarationNames(projections)).not.toContain(
      'CreateMeterRequestInput',
    )
    // The single emission uses input semantics: the optional collection is
    // pointered so an explicitly empty map survives omitempty.
    const labels = goFields(program, request, { mode: 'input' }).find(
      (field) => field.name === 'Labels',
    )
    expect(labels?.typeText).toBe('*map[string]string')
  })

  it('keeps distinct read and input declarations for dual-reachable divergent models', async () => {
    const { program, projections, namedType } = await planFixture(`
      model Event {
        id: string;
        specversion: string = "1.0";
      }

      model IngestedEvent {
        event: Event;
      }

      namespace Events {
        interface Operations {
          @post op ingest(@body body: Event): void;
          @get op get(): IngestedEvent;
        }
      }

      @service
      namespace Test {
        @route("/events")
        interface Endpoints extends Events.Operations {}
      }
    `)

    const event = namedType('Event')
    expect(projections.divergent.has(event)).toBe(true)
    expect(projections.declarations.get(event)).toEqual([
      { name: 'Event', mode: 'read' },
      { name: 'EventInput', mode: 'input' },
    ])
    // Request-side references resolve to the input twin, read side to Event.
    expect(goType(program, event, { mode: 'input' }).type).toBe('EventInput')
    expect(goType(program, event).type).toBe('Event')
  })

  it('does not diverge dual-reachable models whose only default sits on an optional property', async () => {
    // fieldShape keeps an already-optional property optional in both modes,
    // so an optional-with-default property renders byte-identically and must
    // not spawn a spurious *Input twin.
    const { program, projections, namedType } = await planFixture(`
      model Profile {
        id: string;
        interval?: string = "PT1H";
      }

      namespace Profiles {
        interface Operations {
          @post op create(@body body: Profile): Profile;
        }
      }

      @service
      namespace Test {
        @route("/profiles")
        interface Endpoints extends Profiles.Operations {}
      }
    `)

    const profile = namedType('Profile')
    expect(projections.divergent.has(profile)).toBe(false)
    expect(declarationNames(projections)).not.toContain('ProfileInput')
    expect(goType(program, profile, { mode: 'input' }).type).toBe('Profile')
  })
})

describe('structural dedupe of visibility projections', () => {
  it('collapses identical projection twins onto the canonical types', async () => {
    const { program, projections, namedType } = await planFixture(`
      model Address {
        // Visible in both lifecycles: the Update copy filters nothing away and
        // stays byte-identical to the canonical model.
        @visibility(Lifecycle.Read, Lifecycle.Update)
        country?: string;
      }

      model Doc {
        address: Address;
      }

      model UpdateDoc
        is FilterVisibility<Doc, #{ all: #[Lifecycle.Update] }, "Update{name}">;

      namespace Docs {
        interface Operations {
          @get op get(): Doc;
          @patch(#{ implicitOptionality: false }) op update(@body body: UpdateDoc): Doc;
        }
      }

      @service
      namespace Test {
        @route("/docs")
        interface Endpoints extends Docs.Operations {}
      }
    `)

    // The Update copies are byte-identical to the canonical read models, so
    // both the root and the nested reference pair collapse.
    expect(projections.aliases.get('UpdateDoc')).toBe('Doc')
    expect(projections.aliases.get('UpdateAddress')).toBe('Address')
    const names = declarationNames(projections)
    expect(names).not.toContain('UpdateDoc')
    expect(names).not.toContain('UpdateAddress')

    const updateDoc = namedType('UpdateDoc')
    expect(goType(program, updateDoc, { mode: 'input' }).type).toBe('Doc')
  })

  it('keeps projection twins whose filtered shape genuinely differs', async () => {
    const { projections } = await planFixture(`
      model Doc {
        name: string;

        @visibility(Lifecycle.Read)
        etag: string;
      }

      model UpdateDoc
        is FilterVisibility<Doc, #{ all: #[Lifecycle.Update] }, "Update{name}">;

      namespace Docs {
        interface Operations {
          @get op get(): Doc;
          @patch(#{ implicitOptionality: false }) op update(@body body: UpdateDoc): Doc;
        }
      }

      @service
      namespace Test {
        @route("/docs")
        interface Endpoints extends Docs.Operations {}
      }
    `)

    expect(projections.aliases.size).toBe(0)
    expect(declarationNames(projections)).toContain('UpdateDoc')
  })
})

describe('anonymous model promotion', () => {
  it('promotes anonymous inline models to enclosing-type-plus-field names', async () => {
    const { program, namedType, projections } = await planFixture(`
      model SubscriptionCreate {
        customer: {
          id?: string;
          key?: string;
        };
      }

      namespace Subscriptions {
        interface Operations {
          @post op create(@body body: SubscriptionCreate): void;
        }
      }

      @service
      namespace Test {
        @route("/subscriptions")
        interface Endpoints extends Subscriptions.Operations {}
      }
    `)

    expect(declarationNames(projections)).toContain(
      'SubscriptionCreateCustomer',
    )
    const create = namedType('SubscriptionCreate') as Model
    const customer = goFields(program, create, { mode: 'input' }).find(
      (field) => field.name === 'Customer',
    )
    expect(customer?.typeText).toBe('SubscriptionCreateCustomer')
  })

  it('fails loudly when a promoted name collides with an existing type', async () => {
    const host = await createTestHost({
      libraries: [HttpTestLibrary, OpenAPITestLibrary],
    })
    const runner = await createTestRunner(host)
    await runner.compile(`
      import "@typespec/http";
      import "@typespec/openapi";
      using TypeSpec.Http;
      using TypeSpec.OpenAPI;

      model WidgetOwner {
        name: string;
      }

      model Widget {
        owner: {
          id: string;
        };
        fallback: WidgetOwner;
      }

      namespace Widgets {
        interface Operations {
          @get op get(): Widget;
        }
      }

      @service
      namespace Test {
        @route("/widgets")
        interface Endpoints extends Widgets.Operations {}
      }
    `)
    const program = runner.program
    configureGoTypeNames(program, [])
    const operations = collectHttpOperations(program)
    const reachability = computeReachability(
      program,
      groupOperations(operations),
      jsonBodyOverrides(program),
    )
    const modelTypes = new Set(
      [...reachability.byResource.values()].flatMap((types) => [...types]),
    )
    resolveGoTypeNames(program, modelTypes)

    expect(() => promoteAnonymousModels(program, modelTypes)).toThrow(
      'promoted anonymous model name WidgetOwner collides',
    )
  })
})

describe('scalar alias pruning', () => {
  it('plans no declarations for spec scalars and keeps fields on Go primitives', async () => {
    const { program, projections, namedType } = await planFixture(`
      scalar ResourceKey extends string;

      model Meter {
        key: ResourceKey;
      }

      namespace Meters {
        interface Operations {
          @get op get(): Meter;
        }
      }

      @service
      namespace Test {
        @route("/meters")
        interface Endpoints extends Meters.Operations {}
      }
    `)

    expect(declarationNames(projections)).toEqual(['Meter'])
    const meter = namedType('Meter') as Model
    const key = goFields(program, meter).find((field) => field.name === 'Key')
    expect(key?.typeText).toBe('string')
  })
})

describe('union rendering guards', () => {
  it('throws for named unions with no concrete variants instead of emitting any', async () => {
    const host = await createTestHost({
      libraries: [HttpTestLibrary, OpenAPITestLibrary],
    })
    const runner = await createTestRunner(host)
    await runner.compile(`
      union Broken {
        nothing: null,
      }
    `)
    const program = runner.program
    configureGoTypeNames(program, [])
    const union = program.getGlobalNamespaceType().unions.get('Broken')!
    resolveGoTypeNames(program, [union])

    expect(() =>
      renderUnion(program, union as Union, { name: 'Broken', mode: 'read' }),
    ).toThrow('union Broken has no concrete variants representable in Go')
  })
})

describe('filter reachability', () => {
  it('does not promote object variants of runtime-backed filter unions', async () => {
    // Runtime-backed filter unions render as static runtime types
    // (StringFilter, ...); their anonymous object variants must not leak into
    // the reachable set and get emitted as dead *Object declarations.
    const { projections } = await planFixture(`
      union StringFieldFilter {
        equals: string,
        object: {
          eq?: string,
        },
      }

      model ItemFilter {
        name?: StringFieldFilter;
      }

      model Item {
        id: string;
      }

      namespace Items {
        interface Operations {
          @get op list(
            @query(#{ style: "deepObject", explode: true }) filter?: ItemFilter,
          ): Item[];
        }
      }

      @service
      namespace Test {
        @route("/items")
        interface Endpoints extends Items.Operations {}
      }
    `)

    const names = declarationNames(projections)
    expect(names).toContain('Item')
    expect(names.some((name) => name.endsWith('FieldFilterObject'))).toBe(false)
  })
})
