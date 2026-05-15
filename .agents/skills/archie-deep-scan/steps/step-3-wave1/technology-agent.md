### Technology agent

> Read config files, package.json/requirements.txt/Gemfile/build.gradle/pubspec.yaml/Package.swift, CI/CD configs, Dockerfiles, cloud platform files. Create a complete technology inventory.
>
> ### 1. Full Stack Inventory (by category)
> For each technology include: category, name, version, purpose, platform (backend|frontend|shared).
>
> Categories to check:
> 1. **Runtime**: Language, version, runtime environment (for each platform)
> 2. **Backend Framework**: Web framework, version, key features used
> 3. **Frontend Framework**: UI framework/library, version, rendering strategy
> 4. **Database**: Type, ORM/query builder, version
> 5. **Cache**: Redis, Memcached, in-memory, browser cache, etc.
> 6. **Queue**: Celery, RabbitMQ, ARQ, Redis Queue, etc.
> 7. **AI/ML**: Providers (OpenAI, Anthropic, etc.), SDKs, models
> 8. **Auth**: Library, provider, JWT/session handling
> 9. **State Management**: Frontend state (Redux, Zustand, React Query, etc.)
> 10. **Styling**: CSS framework, component library
> 11. **Validation**: Library, approach
> 12. **Testing**: Framework, tools, coverage approach (for each platform)
> 13. **Linting/Formatting**: Tools, configuration
> 14. **Monitoring**: Logging, metrics, error tracking
>
> ### 2. Run Commands
> From package.json scripts, Makefile, Rakefile, etc. Map command name to command string.
>
> ### 3. Project Structure
> ASCII directory tree from scan.json showing top-level organization.
>
> ### 4. Templates
> Common file patterns — how to create a new component/route/service/test in this codebase. Include file_path_template, component_type, description, and a brief code skeleton (max 3 lines).
>
> ### 5. Deployment Detection (CRITICAL — check for ALL of these)
> - **Cloud provider**: GCP (app.yaml, cloudbuild.yaml, google-cloud-* deps, firebase.json), AWS (boto3, aws-cdk, serverless.yml, buildspec.yml, template.yaml), Azure (azure-* SDKs, azure-pipelines.yml, host.json), Vercel (vercel.json), Netlify (netlify.toml), Fly.io (fly.toml), Railway (railway.json), Render (render.yaml)
> - **Compute**: Cloud Run, App Engine, Lambda, EC2, Fargate, Azure Functions, Vercel Edge, Heroku dynos
> - **Container**: Docker (Dockerfile, .dockerignore), Podman; orchestration (Kubernetes, Docker Compose, ECS, Helm, skaffold)
> - **Serverless**: Cloud Functions, Lambda, Edge Functions, Vercel Serverless
> - **CI/CD**: GitHub Actions (.github/workflows/), Cloud Build (cloudbuild.yaml), GitLab CI (.gitlab-ci.yml), CircleCI, Jenkins, Fastlane (Fastfile), Bitrise
> - **Distribution**: App Store, Google Play, npm registry, PyPI, Docker Hub, Maven Central, CocoaPods, pub.dev, Homebrew, APK sideload
> - **IaC**: Terraform (*.tf), CloudFormation/SAM (template.yaml), Pulumi, Helm charts
> - **Supporting services**: Firebase, Supabase, Redis Cloud, managed databases, CDNs, object storage (GCS, S3)
> - **Environment config**: .env files, Secret Manager, SSM Parameter Store, Vault, config maps
> - **Mobile-specific**: Backend services (BaaS), push notification providers, analytics, OTA updates, app signing
> - **Library-specific**: Package registry, build/publish pipeline, versioning strategy
> - List all deployment-related KEY FILES found in the repository
>
> ### 6. Development Rules + Infrastructure Rules
>
> Two SEPARATE output buckets. Each rule belongs in exactly one. The split is load-bearing — `development_rules` is what a coding agent reads at edit-time; `infrastructure_rules` is onboarding/ops info that the same agent never consults when writing code. Mixing them buries the coding rules in noise.
>
> #### `development_rules` — coding-time rules
>
> Rules a coding agent must follow when **writing, editing, or refactoring** code in this project. Each must answer at least one of:
> - "When I add a new <thing> (screen, repository, route, module), what must I do?"
> - "When I touch <area>, what must I avoid?"
> - "What boundary must I respect when changing <component>?"
>
> **Inclusion test:** would a coding agent change the file it's editing because of this rule? If no, it does NOT belong here — push it to `infrastructure_rules`.
>
> **Source priority:** actual code files first (real `.kt`, `.swift`, `.py`, `.ts` etc., the patterns visible there). Config files only when they map directly to a coding-time decision (e.g. a Gradle plugin that requires registering Fragments in NavGraph IS a coding rule; a CI YAML telling you which gradle tasks to run is NOT).
>
> Categories:
> - `pattern_to_follow` — a positive convention visible in existing code
> - `anti_pattern` — a thing the codebase carefully AVOIDS
> - `boundary` — a layer/module separation that must hold
> - `wiring` — what to register/connect when adding a feature (DI, navigation, routing, plugins)
> - `data_flow` — how data moves and where to plug in
>
> GOOD examples:
> - "Always extend TraceViewModel<T> for screen ViewModels — gives Firebase Performance traces per loading cycle" (source: util/perf/TraceViewModel.kt; usage: DashboardViewModel, SettingsViewModel)
> - "Always register new Fragments in res/navigation/navigation_main.xml and use Safe Args (navArgs() delegates) for type-safe nav arguments" (source: app/build.gradle.kts safeArgsPlugin + DashboardFragmentArgs)
> - "Never import from infrastructure/ in domain/ — dependency rule enforced by layer structure" (source: directory layout, no existing violations)
>
> BAD examples (push to infrastructure_rules instead):
> - "Always run `assemble{BuildType} cleanTestDebugUnitTest` in CI" — CI orchestration
> - "Always store keystores as Azure DevOps secure files named …" — signing infra
> - "Never commit local.properties" — onboarding gotcha, not coding rule
>
> #### `infrastructure_rules` — ops / build / onboarding
>
> Rules and gotchas about CI, distribution, signing, secrets, env setup, branch protection, dependency-registry auth — the kind of thing a developer needs to know once during onboarding or when touching pipelines / build configs, but not when writing a feature.
>
> Categories:
> - `ci_cd` — CI orchestration, pipeline triggers, mandatory tasks
> - `distribution` — App Store, Play Store, AppCenter, npm publish, etc.
> - `signing` — keystores, certificates, code-signing identities
> - `secrets` — env vars, secret-file conventions, `.gitignore` of sensitive paths
> - `env_setup` — `.env` files, local-only files, dev-machine prereqs
> - `dependency_registry` — auth to private registries (e.g. GitHub Packages, internal Artifactory), JitPack/private Maven repos
> - `git` — branch protection, PR-trigger conventions, commit hooks
>
> Every rule MUST cite its source file. State each as "Always X" or "Never Y".
>
> **CRITICAL**: Both buckets must be specific to THIS project. Generic rules are WORTHLESS in either bucket.
>
> Return JSON:
> ```json
> {
>   "technology": {
>     "stack": [{"category": "", "name": "", "version": "", "purpose": ""}],
>     "run_commands": {"command_name": "command_string"},
>     "project_structure": "ASCII tree",
>     "templates": [{"component_type": "", "description": "", "file_path_template": "", "code": ""}]
>   },
>   "deployment": {
>     "runtime_environment": "GCP|AWS|Azure|Vercel|on-device|browser|self-hosted",
>     "compute_services": [],
>     "container_runtime": "Docker|Podman|none",
>     "orchestration": "Kubernetes|Docker Compose|ECS|none",
>     "serverless_functions": "Cloud Functions|Lambda|Edge Functions|none",
>     "ci_cd": [],
>     "distribution": [],
>     "infrastructure_as_code": "Terraform|CloudFormation|Pulumi|none",
>     "supporting_services": [],
>     "environment_config": "",
>     "key_files": []
>   },
>   "development_rules": [
>     {"category": "pattern_to_follow|anti_pattern|boundary|wiring|data_flow", "rule": "Always/Never ...", "source": "file_that_proves_it"}
>   ],
>   "infrastructure_rules": [
>     {"category": "ci_cd|distribution|signing|secrets|env_setup|dependency_registry|git", "rule": "Always/Never ...", "source": "file_that_proves_it"}
>   ]
> }
> ```

