# _utils

<!-- archie:ai-start -->

> Low-level serialization/deserialization utilities for the generated Python SDK. Provides SdkJSONEncoder, the Model base class (from serialization.py), and format-aware type converters for datetime, bytes, timedelta, decimal, and array-encoded strings.

## Patterns

**Format-dispatch deserialization** — get_deserializer() selects a converter from _DESERIALIZE_MAPPING (keyed by Python type) or _DESERIALIZE_MAPPING_WITHFORMAT (keyed by wire format string like 'rfc7231', 'base64url'). Never hand-roll datetime/bytes parsing; route through this function. (`get_deserializer(datetime, rf) returns _deserialize_datetime or _deserialize_datetime_rfc7231 based on rf._format`)
**SdkJSONEncoder for serialization** — Custom JSONEncoder in model_base.py handles datetime, bytes, timedelta, decimal, and Model instances. Pass exclude_readonly=True when building request bodies to strip read-only fields. (`json.dumps(payload, cls=SdkJSONEncoder, exclude_readonly=True)`)
**Two-layer Model hierarchy** — model_base.py defines _Model (MutableMapping-backed, rest_field-annotated). serialization.py defines Model (attribute_map + _subtype_map, Autorest-style). These are distinct base classes for different codegen patterns; do not mix. (`_Model uses rest_field(); serialization.Model uses _attribute_map class var`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `model_base.py` | Defines _Model (the MutableMapping-based base for generated models), rest_field, rest_discriminator, SdkJSONEncoder, and all _deserialize_* helpers used by operations. | Mutating _data directly on a _Model instance bypasses the cache-invalidation in __setitem__; always go through attribute access or dict interface. |
| `serialization.py` | Defines the legacy Autorest-style Model base class, Serializer, Deserializer, RawDeserializer, and key transformers (attribute_transformer, full_restapi_key_transformer). Used by _operations.py via _SERIALIZER = Serializer(). | _SERIALIZER is a module-level singleton with client_side_validation disabled; do not re-enable per-call. |
| `__init__.py` | Re-exports SdkJSONEncoder, Model, rest_field, rest_discriminator as the public surface of this utils package. | Adding new public symbols here must also be listed in __all__. |

## Anti-Patterns

- Hand-rolling datetime or bytes serialization instead of using _serialize_datetime / _serialize_bytes
- Editing these files directly — they are generated and will be overwritten by make gen-api
- Importing from corehttp internals not already imported here (adds fragile transitive dependency)

## Decisions

- **Two separate Model base classes coexist: _Model (MutableMapping, rest_field) and serialization.Model (attribute_map, Autorest).** — The SDK was generated with a newer codegen (_Model) but retains the legacy Autorest Model for backwards compatibility with older client patterns and XML support.

## Example: Deserialize a datetime field with format awareness

```
from openmeter._generated._utils.model_base import get_deserializer
deserializer = get_deserializer(datetime, rf)  # rf._format may be 'rfc7231' or None
value = deserializer(raw_value)
```

<!-- archie:ai-end -->
