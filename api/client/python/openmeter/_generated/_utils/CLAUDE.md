# _utils

<!-- archie:ai-start -->

> Low-level serialization/deserialization utilities for the generated Python SDK: SdkJSONEncoder, the MutableMapping-based _Model base (model_base.py), format-aware datetime/bytes/timedelta/decimal converters, and the legacy Autorest-style Model/Serializer/Deserializer (serialization.py). Fully generated — never hand-edited.

## Patterns

**Format-dispatch deserialization via get_deserializer()** — Route datetime/bytes/timedelta/decimal deserialization through get_deserializer(annotation, rf); it selects from _DESERIALIZE_MAPPING (keyed by Python type) or _DESERIALIZE_MAPPING_WITHFORMAT (keyed by wire format e.g. 'rfc7231', 'base64url'). Never hand-roll datetime/bytes parsing. (`deserializer = get_deserializer(datetime, rf); value = deserializer(raw_value)`)
**SdkJSONEncoder with exclude_readonly for request bodies** — Build request bodies via json.dumps(payload, cls=SdkJSONEncoder, exclude_readonly=True). default() strips fields whose rest_field _visibility == ['read'] and handles datetime, bytes, timedelta, decimal, _Null, and _Model instances. (`json.dumps(payload, cls=SdkJSONEncoder, exclude_readonly=True)`)
**Two distinct Model base classes — do not mix** — model_base.py defines _Model (MutableMapping-backed, rest_field annotations); serialization.py defines Model (_attribute_map class var, Autorest Serializer/Deserializer). New dataclass-style models extend _Model; _operations.py uses serialization.Model via _SERIALIZER. Never subclass _Model and also declare _attribute_map. (`class MyModel(_Model): field: str = rest_field(...)`)
**_MyMutableMapping cache invalidation on __setitem__** — _MyMutableMapping clears the _deserialized_<key> attribute cache on every __setitem__. Mutating _data directly bypasses invalidation; always go through attribute or dict access. (`obj.name = 'new'   # or obj['name'] = 'new'  — never obj._data['name'] = 'new'`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `model_base.py` | Defines _Model, _MyMutableMapping, rest_field, rest_discriminator, SdkJSONEncoder, and all _deserialize_*/_serialize_* helpers plus get_deserializer(). Core of the new-codegen path. | Mutating _data directly bypasses the _deserialized_<key> cache invalidation in __setitem__; always go through attribute or dict interface. |
| `serialization.py` | Legacy Autorest Model base, Serializer, Deserializer, RawDeserializer, key transformers (attribute_transformer, full_restapi_key_transformer). Consumed by _operations.py via the module-level _SERIALIZER singleton. | _SERIALIZER has client_side_validation=False; do not create per-call Serializer instances or re-enable validation. |
| `__init__.py` | Re-exports SdkJSONEncoder, Model, rest_field, rest_discriminator as the package public surface. | Any new public symbol must also appear in __all__. |

## Anti-Patterns

- Hand-rolling datetime/bytes serialization instead of using _serialize_datetime / _serialize_bytes / get_deserializer()
- Editing model_base.py or serialization.py — they are generated and overwritten by make gen-api
- Creating per-request Serializer() instances — _SERIALIZER is a module-level singleton with client_side_validation=False
- Subclassing _Model and also declaring _attribute_map — mutually exclusive codegen paths
- Importing corehttp internals not already imported here (fragile transitive dependency)

## Decisions

- **Two Model base classes coexist: _Model (MutableMapping, rest_field) and serialization.Model (attribute_map, Autorest)** — SDK was generated with newer codegen (_Model) but retains the legacy Autorest Model for backwards compatibility and XML support.
- **_DESERIALIZE_MAPPING_WITHFORMAT separates format-aware converters from type-based converters** — The same Python type (datetime) maps to multiple wire formats (rfc3339, rfc7231, unix-timestamp); splitting lets get_deserializer() check format first, then fall back to type-only dispatch.

## Example: Deserialize a datetime field with format awareness from a rest_field annotation

```
from openmeter._generated._utils.model_base import get_deserializer
deserializer = get_deserializer(datetime, rf)  # rf._format may be 'rfc7231', 'unix-timestamp', or None
value = deserializer(raw_value)  # returns a timezone-aware datetime
```

<!-- archie:ai-end -->
