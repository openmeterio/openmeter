### UI Layer agent (only if `frontend_ratio` >= 0.20)

> Read all UI/frontend source files. This codebase contains a significant UI layer. Analyze it as ONE ASPECT of the project — do NOT label the entire project as a "frontend app" or "iOS app" if the UI is only part of a larger system. Adapt each section to the platform detected.
>
> ### 1. Framework & Rendering
> - **Web**: What UI framework? (React, Vue, Angular, Svelte, etc.) Rendering strategy? (SSR, SSG, CSR, ISR, hybrid) Meta-framework? (Next.js, Nuxt, Remix, etc.)
> - **iOS**: Declarative (SwiftUI) vs imperative (UIKit)? App architecture? (MVVM, VIPER, Clean Swift, etc.)
> - **Android**: Jetpack Compose vs XML layouts? Architecture? (MVVM, MVI, Clean Architecture?)
> - **Cross-platform**: Framework? (Flutter, React Native, KMP, etc.) Platform-specific bridging?
>
> ### 2. UI Components
> For each major component/screen/view:
> - **name** and **location** (must exist in file_tree)
> - **component_type**: screen | page | layout | feature | shared | primitive | widget
> - **description**: What it renders and its purpose
> - **props**/inputs/parameters
> - **children**: Child components it renders or embeds
>
> ### 3. State Management
> - **Web**: Global state (Context, Redux, Zustand, Recoil), server state (React Query, SWR, Apollo), local state, form state
> - **iOS**: @State, @Observable, @EnvironmentObject, Combine, MVVM with ObservableObject, @Published
> - **Android**: ViewModel + StateFlow/LiveData, Compose state (remember/mutableStateOf), MVI pattern, SavedStateHandle
> - **Cross-platform**: Platform-specific (Bloc, Riverpod, Provider for Flutter; Redux/MobX for RN)
>
> Document: approach, global_state stores with purposes, server_state mechanism, local_state approach, and rationale for the choices.
>
> ### 4. Routing / Navigation
> - **Web**: File-based or config-based? List routes with paths, components, descriptions, auth requirements, dynamic segments.
> - **iOS**: NavigationStack, UINavigationController, Coordinator pattern, TabBar, deep links
> - **Android**: Navigation Component (NavHost/NavGraph), bottom navigation, deep links, Intent-based
> - **Cross-platform**: Navigator/Router, tab/stack/drawer navigation
>
> ### 5. Data Fetching / Networking
> - **Web**: Data fetching hooks/patterns, loading/error states, caching strategy
> - **iOS**: URLSession, Alamofire, Moya; Combine/async-await patterns; offline support
> - **Android**: Retrofit, Ktor, OkHttp; Coroutine/Flow patterns; offline support (Room + network)
> - **Cross-platform**: Dio/http for Flutter; fetch/axios for RN; Ktor for KMP
>
> ### 6. Styling / Theming
> - **Web**: Tailwind, CSS Modules, Styled Components, component library, design tokens
> - **iOS**: SwiftUI modifiers, UIKit programmatic styling, Interface Builder, custom themes
> - **Android**: Compose theming/Material3, XML themes, Material Components, custom design system
> - **Cross-platform**: Flutter ThemeData, RN StyleSheet, shared design tokens
>
> ### 7. Key Conventions
> - File naming conventions for components/screens/views
> - Component organization (co-located, feature-based, atomic, module-based)
> - Custom hooks/extensions/utilities naming and organization
> - Test file placement
>
> Return JSON:
> ```json
> {
>   "frontend": {
>     "framework": "",
>     "rendering_strategy": "SSR | SSG | CSR | ISR | hybrid | declarative | imperative",
>     "ui_components": [
>       {"name": "", "location": "", "component_type": "", "description": "", "props": [], "children": []}
>     ],
>     "state_management": {
>       "approach": "",
>       "global_state": [{"store": "", "purpose": ""}],
>       "server_state": "",
>       "local_state": "",
>       "rationale": ""
>     },
>     "routing": [
>       {"path": "", "component": "", "description": "", "auth_required": false}
>     ],
>     "data_fetching": [
>       {"name": "", "mechanism": "", "when_to_use": "", "examples": []}
>     ],
>     "styling": "",
>     "key_conventions": []
>   }
> }
> ```

