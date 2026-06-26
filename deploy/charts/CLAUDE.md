# charts

<!-- archie:ai-start -->

> Organisational root for OpenMeter's Helm distribution. Holds two independent charts — the full `openmeter` platform install and the lightweight `benthos-collector` sidecar — plus the shared helm-docs tooling (Makefile + template.md) that regenerates every chart README from its values.yaml comments.

## Patterns

**Generated READMEs via helm-docs** — Chart READMEs are never hand-written; `make docs` runs helm-docs with the shared `template.md` plus each chart's `README.tmpl.md`. Edit the template files and values.yaml `# --` comments, then regenerate. (`make docs  # helm-docs -s file -c . -t $PWD/template.md -t README.tmpl.md`)
**Shared badge/TL;DR template partials** — template.md defines reusable helm-docs partials (`chart.versionBadge`, `chart.artifactHubBadge`, `tldr`, `chart.base`) that every chart README inherits, so per-chart README.tmpl.md only supplies chart-specific prose. (`{{- define "tldr" -}} helm install --generate-name --wait oci://ghcr.io/openmeterio/helm-charts/{{ .Name }} {{- end -}}`)
**Charts published as OCI artifacts** — Both charts are distributed via `oci://ghcr.io/openmeterio/helm-charts/<name>` and indexed on artifacthub.io/packages/helm/openmeter/<name>; install snippets and badge URLs derive from `.Name`. (`[![artifact hub](.../artifact%20hub-{{ .Name | replace "-" "--" }}-...)](https://artifacthub.io/packages/helm/openmeter/{{ .Name }})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `Makefile` | Single `docs` target that drives helm-docs across all charts using `-s file` (sort values by source order), `-c .` (this dir), the shared `template.md`, and each chart's `README.tmpl.md`. | Only sanctioned way to regenerate READMEs; running helm-docs without these flags reorders the values table and drops the shared badges. |
| `template.md` | helm-docs partial library: badge definitions (type/kube/appVersion/version/artifactHub), the `tldr` install block, and the `chart.base`/`chart.baseHead` composition that assembles each README. | Renaming a `define` here silently breaks every chart README referencing it; the OCI registry path `ghcr.io/openmeterio/helm-charts` is hardcoded in the `tldr` partial. |

## Anti-Patterns

- Editing any chart's README.md by hand — it is overwritten on the next `make docs` from template.md + README.tmpl.md + values.yaml comments.
- Adding a third chart without giving it a README.tmpl.md and re-running `make docs`, leaving it undocumented.
- Forking the badge/TL;DR markup into a per-chart template instead of reusing the shared `template.md` partials.
- Hardcoding a different OCI registry in a chart README instead of relying on the `tldr` partial's `ghcr.io/openmeterio/helm-charts/{{ .Name }}` path.

## Decisions

- **Two separate charts (openmeter, benthos-collector) rather than one umbrella chart with a toggle.** — The collector is an independently-deployed StatefulSet sidecar with its own lifecycle and a much smaller values contract; coupling it to the full platform chart would force unrelated upgrades.
- **Documentation is generated, with shared markup centralised in template.md.** — Keeps version/type/artifacthub badges and the install TL;DR identical across charts and keeps the values table in sync with the actual values.yaml `# --` comments.

## Example: Regenerate all chart READMEs after editing values.yaml documentation comments

```
# from deploy/charts
make docs
# == helm-docs --log-level trace -s file -c . -t $PWD/template.md -t README.tmpl.md
# reads each chart's values.yaml `# --` comments + README.tmpl.md, applies template.md partials
```

<!-- archie:ai-end -->
