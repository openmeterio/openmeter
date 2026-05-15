---

**Every agent also gets these GROUNDING RULES:**

> **GROUNDING RULES — every claim must come from code you READ, never from names or conventions.**
> This applies to ANY codebase — web, mobile, desktop, backend, CLI, library, any language.
>
> 1. **Paths**: Only reference file/directory paths that appear in scan.json file_tree. Do NOT infer conventional paths — every framework has different conventions (Rails uses `spec/`, Go uses `_test.go` files, Swift uses `Tests/`, etc.). Find the ACTUAL paths by reading the code.
> 2. **Component responsibility**: Read the actual source file. Describe what the code DOES, not what the filename or class name suggests. A file named `Publisher.swift` might only subscribe to events. A class named `UserManager` might only read, never write. Only the code tells you which.
> 3. **API/route/endpoint methods**: Read the actual handler code. List ONLY the methods/verbs/actions that are actually implemented. Do NOT assume CRUD from the resource name — a "presets" endpoint might support GET+PUT but not POST. This applies to HTTP routes, gRPC services, GraphQL resolvers, CLI commands, or any interface.
> 4. **File placement rules**: Search the file_tree for where test files, config files, build output, and generated code actually live. Do NOT assume conventional locations — the project might put tests alongside source, in a top-level `test/` dir, or anywhere else.
> 5. **Pitfalls**: Only describe problems grounded in actual code patterns you observed. Describe what the code DOES and the risks of THAT pattern. Do NOT recommend alternatives the code doesn't use as if it does.
> 6. **Build/output paths**: Read the build config, generation logic, or Makefile to find where output is actually written. Do NOT assume any framework's default output directory.
>
> **The rule is simple: if you didn't read it in a file, don't claim it.** Focus on cross-file relationships, architecture decisions, conventions, and integration patterns. Return ONLY valid JSON.

**Spawn ALL agents in parallel.**

```bash
python3 .archie/intent_layer.py deep-scan-state "$PROJECT_ROOT" complete-step 3
```

