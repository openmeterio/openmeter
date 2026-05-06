#!/usr/bin/env bash
set -euo pipefail

cd api/spec

pnpm --frozen-lockfile install
pnpm format
pnpm generate

# Replace inline filter definitions with $ref to common/definitions/aip_filters.yaml.
AIP_REF="../../../../common/definitions/aip_filters.yaml#/components/schemas"
FILE="packages/aip/output/definitions/metering-and-billing/v3/openapi.MeteringAndBilling.yaml"

for schema in SortQuery BooleanFieldFilter NumericFieldFilter StringFieldFilter StringFieldFilterExact DateTimeFieldFilter LabelsFieldFilter; do
  if yq -e ".components.schemas | has(\"$schema\")" "$FILE" > /dev/null 2>&1; then
    REF_VAL="$AIP_REF/$schema" SCHEMA="$schema" \
      yq -i '.components.schemas[strenv(SCHEMA)] = {"$ref": strenv(REF_VAL)}' "$FILE"
  fi
done

pnpm --filter @openmeter/api-spec-aip exec openapi bundle output/definitions/metering-and-billing/v3/openapi.OpenMeter.yaml -o ../../../v3/openapi.yaml
cp packages/legacy/output/openapi.OpenMeter.yaml ../openapi.yaml
cp packages/legacy/output/openapi.OpenMeterCloud.yaml ../openapi.cloud.yaml
