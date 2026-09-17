# Configuration Documentation

> Auto-generated documentation. Do not edit manually.

## Configuration Overview

| Key | Environment Variable | Type | Default | Required | Description |
| --- | --- | --- | --- | --- | --- |
| `MyOwnSetting` | `MY_OWN_SETTING` | `int` | 42 | `No` | This is just a simple example description so this map is not empty |
| `MyStringSetting` | `MY_STRING_SETTING` | `string` | defaultStringSetting | `Yes` | This is just a string setting that is empty but required. |

## Docker Compose Example

```yaml
services:
  app:
    environment:
      MY_OWN_SETTING: "42"
      MY_STRING_SETTING: "defaultStringSetting"
```

## Docker Run Example

```bash
docker run -it \
  -e MY_OWN_SETTING='42' \
  -e MY_STRING_SETTING='defaultStringSetting'
  your-image:latest
```
