import { type Children, type ComponentContext, createContext, type Refkey, useContext } from "@alloy-js/core";
import { FunctionCallExpression, MemberExpression } from "@alloy-js/typescript";
import type { Program, Type } from "@typespec/compiler";
import { $ } from "@typespec/compiler/typekit";
import { getEmitOptionsForType, type ZodOptionsContext } from "./context/zod-options.js";
import { zod } from "./external-packages/zod.js";

export const refkeySym = Symbol.for("typespec-zod.refkey");

/**
 * Tracks the type currently being declared, so that properties whose subtree
 * loops back to it can be emitted as object getters (`get name() { ... }`).
 * Wrapping recursive members in a getter defers evaluation, which sidesteps
 * TS 2448 ("used before its declaration") and TS 7022/7024 ("implicit any in
 * own initializer") that pure `z.lazy(() => x)` would still trigger.
 */
export const DeclaringTypeContext: ComponentContext<Type | undefined> = createContext<Type | undefined>(undefined);

export function useDeclaringType(): Type | undefined {
  return useContext(DeclaringTypeContext);
}

/**
 * Returns true if the given type is a declaration or an instantiation of a
 * declaration.
 */
export function isDeclaration(program: Program, type: Type): boolean {
  switch (type.kind) {
    case "Namespace":
    case "Interface":
    case "Operation":
    case "EnumMember":
    case "UnionVariant":
      return false;

    case "Model":
      if (($(program).array.is(type) || $(program).record.is(type)) && isBuiltIn(program, type)) {
        return false;
      }

      return Boolean(type.name);
    case "Union":
      return Boolean(type.name);
    case "Enum":
      return true;
    case "Scalar":
      return true;
    default:
      return false;
  }
}

export function isRecord(program: Program, type: Type): boolean {
  return type.kind === "Model" && !!type.indexer && type.indexer.key === $(program).builtin.string;
}

export function shouldReference(program: Program, type: Type, options?: ZodOptionsContext) {
  return (
    isDeclaration(program, type) &&
    !isBuiltIn(program, type) &&
    (!options || !getEmitOptionsForType(program, type, options?.customEmit)?.noDeclaration)
  );
}

export function isBuiltIn(program: Program, type: Type) {
  let resolved: Type = type;
  if (resolved.kind === "ModelProperty" && resolved.model) {
    resolved = resolved.model;
  }

  if (!("namespace" in resolved) || resolved.namespace === undefined) {
    return false;
  }

  const globalNs = program.getGlobalNamespaceType();
  let tln = resolved.namespace;
  if (tln === globalNs) {
    return false;
  }

  while (tln.namespace !== globalNs) {
    tln = tln.namespace!;
  }

  return tln === globalNs.namespaces.get("TypeSpec");
}

/**
 * Returns true if the given type's inlined subtree contains a reference back
 * to the declaring type. We descend through types that get inlined into the
 * current declaration (arrays, records, unions, tuples, anonymous models),
 * and stop at any other declaration boundary because that subtree would emit
 * as its own refkey rather than text inside the current schema.
 */
export function subtreeReachesType(program: Program, root: Type, target: Type): boolean {
  const visited = new Set<Type>();

  function walk(type: Type): boolean {
    if (type === target) {
      return true;
    }
    if (visited.has(type)) {
      return false;
    }
    visited.add(type);

    if (type !== root && shouldReference(program, type)) {
      return false;
    }

    switch (type.kind) {
      case "Model": {
        if (type.baseModel && walk(type.baseModel)) {
          return true;
        }
        if (type.indexer) {
          if (walk(type.indexer.key) || walk(type.indexer.value)) {
            return true;
          }
        }
        for (const prop of type.properties.values()) {
          if (walk(prop.type)) {
            return true;
          }
        }
        return false;
      }
      case "ModelProperty":
        return walk(type.type);
      case "Union":
        for (const variant of type.variants.values()) {
          const variantType = variant.kind === "UnionVariant" ? variant.type : variant;
          if (walk(variantType)) {
            return true;
          }
        }
        return false;
      case "UnionVariant":
        return walk(type.type);
      case "Tuple":
        for (const value of type.values) {
          if (walk(value)) {
            return true;
          }
        }
        return false;
      case "Scalar":
        return type.baseScalar ? walk(type.baseScalar) : false;
      default:
        return false;
    }
  }

  return walk(root);
}

interface TypeCollector {
  collectType: (type: Type) => void;
  get types(): Type[];
}

/**
 * Recursively collect all declaration types that the given type depends on.
 * This handles nested types like arrays, records, unions, etc.
 */
function collectAllReferencedTypes(program: Program, type: Type, visited: Set<Type>, result: Set<Type>): void {
  if (visited.has(type)) {
    return;
  }
  visited.add(type);

  if (shouldReference(program, type)) {
    result.add(type);
  }

  switch (type.kind) {
    case "Model": {
      if (type.baseModel) {
        collectAllReferencedTypes(program, type.baseModel, visited, result);
      }
      if (type.indexer) {
        collectAllReferencedTypes(program, type.indexer.key, visited, result);
        collectAllReferencedTypes(program, type.indexer.value, visited, result);
      }
      for (const prop of type.properties.values()) {
        collectAllReferencedTypes(program, prop.type, visited, result);
      }
      break;
    }
    case "Union": {
      for (const variant of type.variants.values()) {
        const variantType = variant.kind === "UnionVariant" ? variant.type : variant;
        collectAllReferencedTypes(program, variantType, visited, result);
      }
      break;
    }
    case "UnionVariant": {
      collectAllReferencedTypes(program, type.type, visited, result);
      break;
    }
    case "Scalar": {
      if (type.baseScalar) {
        collectAllReferencedTypes(program, type.baseScalar, visited, result);
      }
      break;
    }
    case "Tuple": {
      for (const value of type.values) {
        collectAllReferencedTypes(program, value, visited, result);
      }
      break;
    }
    case "Enum":
      break;
    default:
      break;
  }
}

/**
 * Get all declaration types that the given type directly depends on.
 */
function getReferencedDeclarations(program: Program, type: Type): Type[] {
  const visited = new Set<Type>();
  const result = new Set<Type>();

  visited.add(type);

  switch (type.kind) {
    case "Model": {
      if (type.baseModel) {
        collectAllReferencedTypes(program, type.baseModel, visited, result);
      }
      if (type.indexer) {
        collectAllReferencedTypes(program, type.indexer.key, visited, result);
        collectAllReferencedTypes(program, type.indexer.value, visited, result);
      }
      for (const prop of type.properties.values()) {
        collectAllReferencedTypes(program, prop.type, visited, result);
      }
      break;
    }
    case "Union": {
      for (const variant of type.variants.values()) {
        const variantType = variant.kind === "UnionVariant" ? variant.type : variant;
        collectAllReferencedTypes(program, variantType, visited, result);
      }
      break;
    }
    case "Scalar": {
      if (type.baseScalar) {
        collectAllReferencedTypes(program, type.baseScalar, visited, result);
      }
      break;
    }
    case "Enum":
      break;
    default:
      break;
  }

  return [...result];
}

/**
 * Performs a topological sort using Kahn's algorithm.
 * Returns types in dependency-first order.
 */
function topologicalSort(program: Program, typeSet: Set<Type>): Type[] {
  const inDegree = new Map<Type, number>();
  const dependents = new Map<Type, Set<Type>>();

  for (const type of typeSet) {
    inDegree.set(type, 0);
    dependents.set(type, new Set());
  }

  for (const type of typeSet) {
    const refs = getReferencedDeclarations(program, type);
    for (const ref of refs) {
      if (typeSet.has(ref) && ref !== type) {
        inDegree.set(type, (inDegree.get(type) ?? 0) + 1);
        dependents.get(ref)?.add(type);
      }
    }
  }

  const result: Type[] = [];
  const placed = new Set<Type>();
  const queue: Type[] = [];

  for (const [type, degree] of inDegree) {
    if (degree === 0) {
      queue.push(type);
    }
  }

  while (queue.length > 0) {
    const current = queue.shift()!;
    result.push(current);
    placed.add(current);

    for (const dependent of dependents.get(current) ?? []) {
      const newDegree = (inDegree.get(dependent) ?? 1) - 1;
      inDegree.set(dependent, newDegree);
      if (newDegree === 0) {
        queue.push(dependent);
      }
    }
  }

  for (const type of typeSet) {
    if (!placed.has(type)) {
      result.push(type);
    }
  }

  return result;
}

export function newTopologicalTypeCollector(program: Program): TypeCollector {
  const typeSet = new Set<Type>();

  return {
    collectType(type: Type) {
      if (shouldReference(program, type)) {
        typeSet.add(type);
      }
    },
    get types() {
      return topologicalSort(program, typeSet);
    },
  };
}

export function call(target: string, ...args: Children[]) {
  return <FunctionCallExpression target={target} args={args} />;
}

export function memberExpr(...parts: Children[]) {
  return <MemberExpression children={parts} />;
}

export function zodMemberExpr(...parts: Children[]) {
  return memberExpr(refkeyPart(zod.z), ...parts);
}

export function idPart(id: string) {
  return <MemberExpression.Part id={id} />;
}

export function refkeyPart(refkey: Refkey) {
  return <MemberExpression.Part refkey={refkey} />;
}

export function callPart(target: string | Refkey, ...args: Children[]) {
  return (
    <MemberExpression>
      {typeof target === "string" ? idPart(target) : refkeyPart(target)}
      <MemberExpression.Part args={args} />
    </MemberExpression>
  );
}
