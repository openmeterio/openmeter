import assert from 'node:assert/strict'
import { it } from 'node:test'
import { getFriendlyName, serializeValueAsJson } from '@typespec/compiler'
import { createTestHost, createTestRunner } from '@typespec/compiler/testing'

it('preserves distinct names for visibility-projected nested models', async () => {
  const host = await createTestHost()
  const runner = await createTestRunner(host)
  await runner.compile(`
    @friendlyName("ApiChild")
    model Child {
      @visibility(Lifecycle.Read) id: string;
      @visibility(Lifecycle.Create) value: string;
    }
    model Parent { child: Child; }
    model Request is FilterVisibility<
      Parent, #{ all: #[Lifecycle.Create] }, "Create{name}"
    >;
  `)

  const models = runner.program.checker.getGlobalNamespaceType().models
  const child = models.get('Child')
  const requestChild = models.get('Request').properties.get('child').type

  // The request clone must not inherit the response model's schema identity.
  assert.equal(getFriendlyName(runner.program, child), 'ApiChild')
  assert.equal(getFriendlyName(runner.program, requestChild), 'CreateChild')
  assert.deepEqual([...requestChild.properties.keys()], ['value'])
})

it('preserves discriminated-union object defaults during JSON serialization', async () => {
  const host = await createTestHost()
  const runner = await createTestRunner(host)
  await runner.compile(`
    enum Kind { subscription: "subscription", anchored: "anchored" }
    model Subscription { type: Kind.subscription; }
    model Anchored { type: Kind.anchored; interval: string; }
    @discriminated(#{ discriminatorPropertyName: "type", envelope: "none" })
    union Alignment { subscription: Subscription, anchored: Anchored }
    const defaultAlignment: Subscription = #{ type: Kind.subscription };
    model Settings { alignment?: Alignment = defaultAlignment; }
  `)

  const property = runner.program.checker
    .getGlobalNamespaceType()
    .models.get('Settings')
    .properties.get('alignment')
  assert.deepEqual(
    serializeValueAsJson(runner.program, property.defaultValue, property),
    { type: 'subscription' },
  )
})
