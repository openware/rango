# Troubleshooting docs

## "message":"Websocket upgrade failed: websocket: request origin not allowed by Upgrader.CheckOrigin"

If you deploy your Rango somewhere, lets say to _www.rango.host_, and try to connect with some local
frontend app you develop, you'll probably face this problem.

It is caused by [CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS), to resolve this issue
specify a list of domains, allowed to connect to server in Rango `API_CORS_ORIGINS` env var.

By default, if this env is not set, only requests from _www.rango.host_ are enabled.

So, if I have some react-app running on _localhost:3000_:

```
API_CORS_ORIGINS=www.rango.host,localhost:3000
```
