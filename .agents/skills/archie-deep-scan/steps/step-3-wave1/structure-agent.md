### Structure agent

> **CRITICAL INSTRUCTIONS:**
> You are analyzing a codebase to understand its architecture. Your goal is to OBSERVE and DESCRIBE what exists, NOT to categorize it into known patterns.
>
> **Workspace-aware addendum (only when `SCOPE === "whole"`):**
> This is a workspace monorepo (`MONOREPO_TYPE={type}`, N workspaces under paths `<workspaces>`). Treat each workspace member as a top-level component in `components.components`:
> - `name` = workspace `name` from its `package.json` (or equivalent for Cargo/Gradle)
> - `location` = workspace directory path relative to `$PROJECT_ROOT`
> - `platform` = inferred from workspace contents (frontend/backend/shared/etc.)
> - `responsibility` = inferred from package `description` + entry points
> - `depends_on` = other workspace members it imports (read its `package.json` dependencies, filter to workspace names)
>
> Additionally produce a top-level `workspace_topology` field:
> ```json
> "workspace_topology": {
>   "type": "{MONOREPO_TYPE}",
>   "members": [{"name": "...", "path": "...", "role": "app|lib|tool"}],
>   "edges": [{"from": "name-a", "to": "name-b", "count": 3}],
>   "cycles": [["a", "b", "a"]],
>   "dependency_magnets": [{"name": "shared", "in_degree": 8}]
> }
> ```
>
> Record `dependency_magnets` (workspaces in the top quartile of in_degree) inside `workspace_topology` as shown — it's a structural inventory field, not a finding.
>
> **DO NOT write directly to `blueprint.pitfalls`.** Pitfalls are emitted only by Wave 2 Opus synthesis in the canonical 4-field shape. Instead, capture cross-workspace import cycles and other workspace-level problems as **draft findings** for Wave 2 to upgrade. Append each to the `findings` array you return alongside your structure output, using this shape:
>
> ```json
> {
>   "problem_statement": "Cross-workspace import cycle: <name-a> → <name-b> → <name-a>",
>   "evidence": ["<name-a>/path/to/file.ts imports <name-b>/…", "<name-b>/path/to/file.ts imports <name-a>/…"],
>   "root_cause": "",
>   "fix_direction": "",
>   "severity": "error",
>   "confidence": 0.9,
>   "applies_to": ["<name-a>", "<name-b>"],
>   "source": "scan:structure",
>   "depth": "draft"
> }
> ```
>
> Leave `root_cause` and `fix_direction` empty strings — Wave 2 will fill them with architectural grounding when it upgrades the draft to canonical. Reference workspace members by **name** (not path) in all cross-references.

>
> **DO NOT:**
> - Assume this is a "layered architecture", "MVC", "Clean Architecture", or any specific pattern
> - Look for patterns that match your training data
> - Force observations into predefined categories
>
> **DO:**
> - Describe the ACTUAL file organization you see
> - Identify how files relate to each other based on naming and imports
> - Note any conventions unique to this codebase
>
> Read all source files. Analyze the actual structure of this codebase:
>
> ### 1. Project Type & Platforms
> - Identify if this is a monorepo, single app, microservice, serverless, full-stack, library, etc.
> - List ALL platforms found: backend, web-frontend, mobile-ios, mobile-android, desktop, CLI, shared/common
> - List main entry point files for EACH platform (main.py, index.ts, App.tsx, AppDelegate.swift, MainActivity.kt, main.dart, etc.)
> - Document module/package organization approach
>
> ### 2. Components
> Identify main components from actual code — class names, imports, file organization. For each component:
> - **name**: Component name
> - **location**: Directory path (MUST exist in file_tree)
> - **platform**: backend | frontend | shared | ios | android
> - **responsibility**: Describe what the code DOES, not what names suggest. BAD: "Handles business logic". GOOD: "Orchestrates weather data fetching via WeatherProvider, manages profile state, coordinates push notification scheduling". Reference actual class names and services.
> - **depends_on**: From actual import statements
> - **exposes_to**: What other components consume from it
> - **key_interfaces**: Actual method/function names with brief description. For API routes, list ONLY methods actually implemented — do NOT assume CRUD.
> - **key_files**: With descriptions of what each file does (paths MUST exist in file_tree)
>
> ### 3. Layers
> Analyze ALL platforms (backend AND frontend). Only document layers you can clearly identify from:
> 1. Import patterns between directories
> 2. Directory structure and naming
> 3. Actual code organization
>
> **If no clear layers exist**, set structure_type to flat, modular, or feature-based and document the actual structure.
>
> For each layer found, document: name, platform (backend|frontend|shared), location, responsibility (SPECIFIC — reference actual classes), contains (component types), depends_on (from imports), exposes_to, key_files.
>
> Common backend patterns (only if they ACTUALLY exist): Presentation/API (routes, controllers, DTOs), Application/Service (orchestration, use cases), Domain (entities, interfaces), Infrastructure (database, external APIs, caching).
> Common frontend patterns (only if they ACTUALLY exist): Pages/Views, Features/Containers, Components/UI, Hooks/Services, State/Store.
>
> ### 4. Architecture Style
> Describe in plain language. Examples: "Actor-based with message passing", "Event-sourced with CQRS separation", "Feature-sliced with co-located concerns", "Traditional layered with services and repositories", "Functional core with imperative shell" — or describe something completely unique.
>
> ### 5. File Placement & Naming
> - Where do tests, configs, components, services actually live? With naming patterns observed.
> - Search the file_tree for where test files, config files, build output, and generated code actually live. Do NOT assume conventional locations.
> - Document actual naming conventions: PascalCase components, snake_case utils, kebab-case files, etc. With 2-4 examples each.
>
> ### 6. Framework Usage
> Catalog external frameworks/libraries from import statements. For each, note the framework name and usage scope.
>
> Return JSON:
> ```json
> {
>   "meta": {
>     "architecture_style": "plain language description",
>     "platforms": ["backend", "web-frontend"],
>     "executive_summary": "3-5 factual sentences: what this does, primary tech, architecture style. No filler."
>   },
>   "components": {
>     "structure_type": "layered | flat | modular | feature-based",
>     "components": [
>       {"name": "", "location": "", "platform": "", "responsibility": "", "depends_on": [], "exposes_to": [], "key_interfaces": [{"name": "", "methods": [], "description": ""}], "key_files": [{"file": "", "description": ""}]}
>     ]
>   },
>   "architecture_rules": {
>     "file_placement_rules": [
>       {"component_type": "", "naming_pattern": "", "location": "", "example": "", "description": ""}
>     ],
>     "naming_conventions": [
>       {"scope": "", "pattern": "", "examples": [], "description": ""}
>     ]
>   }
> }
> ```

