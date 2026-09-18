# Configuration Documentation

> Auto-generated documentation. Do not edit manually.

## Configuration Overview

| Key | Environment Variable | Type | Default | Required | Description |
| --- | --- | --- | --- | --- | --- |
| `StringValue` | `MY_APP_PREFIX_STRING_VALUE` | `string` | DefaultStringValue | `No` | An example for a string value |
| `IntValue` | `MY_APP_PREFIX_INT_VALUE` | `int` | 100 | `No` | An example for an int value |
| `FloatValue` | `MY_APP_PREFIX_FLOAT_VALUE` | `float64` | 3.141 | `No` | An example for a float value |
| `BoolValue` | `MY_APP_PREFIX_BOOL_VALUE` | `bool` | true | `No` | An example for a boolean value |
| `MaskedValue` | `MY_APP_PREFIX_MASKED_VALUE` | `float64` | [Masked (len: 5)] | `No` | An example for a masked value |
| `RequiredValue` | `MY_APP_PREFIX_REQUIRED_VALUE` | `string` | MyRequiredValue | `Yes` | An example for a required value |

## Docker Compose Example

```yaml
services:
  app:
    environment:
      MY_APP_PREFIX_STRING_VALUE: "DefaultStringValue"
      MY_APP_PREFIX_INT_VALUE: "100"
      MY_APP_PREFIX_FLOAT_VALUE: "3.141"
      MY_APP_PREFIX_BOOL_VALUE: "true"
      MY_APP_PREFIX_MASKED_VALUE: "[Masked (len: 5)]"
      MY_APP_PREFIX_REQUIRED_VALUE: "MyRequiredValue"
```

## Docker Run Example

```bash
docker run -it \
  -e MY_APP_PREFIX_STRING_VALUE='DefaultStringValue' \
  -e MY_APP_PREFIX_INT_VALUE='100' \
  -e MY_APP_PREFIX_FLOAT_VALUE='3.141' \
  -e MY_APP_PREFIX_BOOL_VALUE='true' \
  -e MY_APP_PREFIX_MASKED_VALUE='[Masked (len: 5)]' \
  -e MY_APP_PREFIX_REQUIRED_VALUE='MyRequiredValue' \
  your-image:latest
```
