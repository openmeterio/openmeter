# _utils

<!-- archie:ai-start -->

> Low-level serialization/deserialization utilities for the generated Python SDK. Provides SdkJSONEncoder, the _Model MutableMapping-based base class (model_base.py), format-aware type converters for datetime/bytes/timedelta/decimal, and the legacy Autorest-style Model/Serializer/Deserializer (serialization.py). These files are fully generated and must not be hand-edited.

## Patterns

**Format-dispatch deserialization via get_deserializer()** — Always route datetime/bytes/timedelta/decimal deserialization through get_deserializer(annotation, rf). It selects from _DESERIALIZE_MAPPING (keyed by Python type) or _DESERIALIZE_MAPPING_WITHFORMAT (keyed by wire format string like 'rfc7231', 'base64url'). Never hand-roll datetime or bytes parsing. (`deserializer = get_deserializer(datetime, rf)  # rf._format may be 'rfc7231' or None; value = deserializer(raw_value)`)
**SdkJSONEncoder for all request body serialization** — Pass exclude_readonly=True when building request bodies so read-only fields (visibility=['read']) are stripped. SdkJSONEncoder handles datetime, bytes, timedelta, decimal, and _Model instances via its default() method. (`json.dumps(payload, cls=SdkJSONEncoder, exclude_readonly=True)`)
**Two distinct Model base classes — do not mix** — model_base.py defines _Model (MutableMapping-backed, uses rest_field annotations). serialization.py defines Model (attribute_map class var, Autorest-style Serializer/Deserializer). _operations.py uses the legacy serialization.Model path via _SERIALIZER; generated dataclass-style models extend _Model. (`# New models: class MyModel(_Model): field: str = rest_field(...)
# Never mix: do not subclass _Model and also declare _attribute_map`)
**_MyMutableMapping cache invalidation on __setitem__** — The _MyMutableMapping backing _Model clears _deserialized_<key> attribute cache on every __setitem__. Mutating _data directly bypasses this invalidation. Always go through attribute access or the dict interface (obj['key'] = value or obj.key = value). (`# Correct: obj.name = 'new'  or  obj['name'] = 'new'
# Wrong: obj._data['name'] = 'new'  # bypasses cache invalidation`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `model_base.py` | Defines _Model (MutableMapping-based base for all generated models), rest_field, rest_discriminator, SdkJSONEncoder, and all _deserialize_* helper functions. Core of the new-codegen path. | Mutating _data directly on a _Model bypasses the _deserialized_<key> cache-invalidation in __setitem__; always go through attribute or dict interface. |
| `serialization.py` | Legacy Autorest-style Model base class, Serializer, Deserializer, RawDeserializer, and key transformers (attribute_transformer, full_restapi_key_transformer). Used by _operations.py via module-level _SERIALIZER singleton. | _SERIALIZER has client_side_validation=False; do not create per-call Serializer instances or re-enable validation. |
| `__init__.py` | Re-exports SdkJSONEncoder, Model, rest_field, rest_discriminator as the public surface of this utils package. | Any new public symbol added here must also appear in __all__. |

## Anti-Patterns

- Hand-rolling datetime or bytes serialization instead of using _serialize_datetime / _serialize_bytes / get_deserializer()
- Editing model_base.py or serialization.py directly — they are generated and overwritten by make gen-api
- Creating per-request Serializer() instances — _SERIALIZER is a module-level singleton with client_side_validation=False
- Importing from corehttp internals not already imported in these files (adds fragile transitive dependency)
- Subclassing _Model and also declaring _attribute_map — these are mutually exclusive codegen paths

## Decisions

- **Two separate Model base classes coexist: _Model (MutableMapping, rest_field) and serialization.Model (attribute_map, Autorest)** — SDK was generated with a newer codegen (_Model) but retains the legacy Autorest Model for backwards compatibility with older client patterns and XML support via serialization.py.
- **_DESERIALIZE_MAPPING_WITHFORMAT separates format-aware converters from type-based converters** — The same Python type (datetime) maps to multiple wire formats (rfc3339, rfc7231, unix-timestamp). Splitting into two maps lets get_deserializer() check format first, then fall back to type-only dispatch.

## Example: Deserialize a datetime field with format awareness from a rest_field annotation

```
from openmeter._generated._utils.model_base import get_deserializer
# rf is the _RestField instance whose _format may be 'rfc7231', 'unix-timestamp', or None
deserializer = get_deserializer(datetime, rf)
value = deserializer(raw_value)  # returns a timezone-aware datetime
```

<!-- archie:ai-end -->
