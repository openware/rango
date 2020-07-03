## Connect to private channel

```bash
wscat --connect localhost:8080/private --header "Authorization: Bearer $(go run ./tools/jwt)"
```
