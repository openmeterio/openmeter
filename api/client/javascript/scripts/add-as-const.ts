import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { type Expression, Project, SyntaxKind } from 'ts-morph'
import type * as ts from 'typescript'

/**
 * Post-generation TypeScript transformer to add "as const" assertions to make sure the generated Zod schemas are valid
 * This fixes TypeScript errors with Zod schemas that expect literal types
 */

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const ZOD_FILE_PATH = path.join(__dirname, '../src/zod/index.ts')

function addAsConstAssertions() {
  const project = new Project({
    tsConfigFilePath: path.join(__dirname, '../tsconfig.json'),
  })

  const sourceFile = project.addSourceFileAtPath(ZOD_FILE_PATH)
  let transformationCount = 0

  // Find all variable declarations
  const variableStatements = sourceFile.getVariableStatements()

  for (const statement of variableStatements) {
    // Only process exported const declarations
    if (!statement.isExported()) {
      continue
    }

    const declarations = statement.getDeclarations()

    for (const declaration of declarations) {
      if (declaration.getKind() !== SyntaxKind.VariableDeclaration) {
        continue
      }

      const name = declaration.getName()
      const initializer = declaration.getInitializer()

      if (!initializer) {
        continue
      }

      // Check if this is a Zod default value (contains "Default" in name)
      const isZodDefault = name.includes('Default')

      // Handle array literals
      if (initializer.getKind() === SyntaxKind.ArrayLiteralExpression) {
        const arrayLiteral = initializer.asKindOrThrow(
          SyntaxKind.ArrayLiteralExpression,
        )

        if (isZodDefault) {
          // For Zod defaults, convert to function that returns mutable array
          const elements = arrayLiteral
            .getElements()
            .map((el) => el.getText())
            .join(', ')
          const returnType = inferZodArrayType(name, elements)

          declaration.setInitializer(`(): ${returnType} => [${elements}]`)
          transformationCount++
        } else if (!hasAsConstAssertion(initializer)) {
          // For regular arrays, add "as const"
          declaration.setInitializer(`${initializer.getText()} as const`)
          transformationCount++
        }
      }

      // Handle boolean literals
      else if (
        initializer.getKind() === SyntaxKind.TrueKeyword ||
        initializer.getKind() === SyntaxKind.FalseKeyword
      ) {
        if (!hasAsConstAssertion(initializer)) {
          declaration.setInitializer(`${initializer.getText()} as const`)
          transformationCount++
        }
      }

      // Handle string literals
      else if (initializer.getKind() === SyntaxKind.StringLiteral) {
        if (!hasAsConstAssertion(initializer)) {
          declaration.setInitializer(`${initializer.getText()} as const`)
          transformationCount++
        }
      }

      // Handle numeric literals
      else if (initializer.getKind() === SyntaxKind.NumericLiteral) {
        if (!hasAsConstAssertion(initializer)) {
          declaration.setInitializer(`${initializer.getText()} as const`)
          transformationCount++
        }
      }

      // Handle object literals (for cases like { type: "subscription" })
      else if (initializer.getKind() === SyntaxKind.ObjectLiteralExpression) {
        if (!hasAsConstAssertion(initializer)) {
          declaration.setInitializer(`${initializer.getText()} as const`)
          transformationCount++
        }
      }
    }
  }

  if (transformationCount > 0) {
    sourceFile.saveSync()
  }
}

function hasAsConstAssertion(node: Expression<ts.Expression>): boolean {
  const parent = node.getParent()
  if (parent?.getKind() === SyntaxKind.AsExpression) {
    return (
      parent.asKindOrThrow(SyntaxKind.AsExpression).getTypeNode()?.getText() ===
      'const'
    )
  }
  return false
}

function inferZodArrayType(variableName: string, elements: string): string {
  // Try to infer the correct type based on common Zod patterns
  if (variableName.includes('Expand')) {
    return "('lines' | 'preceding' | 'workflow.apps')[]"
  }

  // For other cases, try to parse the elements
  const elementTypes = elements.split(',').map((el) => {
    const trimmed = el.trim()
    if (trimmed.startsWith("'") || trimmed.startsWith('"')) {
      return trimmed
    }
    return `'${trimmed}'`
  })

  return `(${elementTypes.join(' | ')})[]`
}

// Run the transformer
try {
  addAsConstAssertions()
} catch (error) {
  console.error('Error during transformation:', error)
  process.exit(1)
}
