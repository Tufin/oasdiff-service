# oasdiff-service

### Validate HTTP server on localhost
```
curl -X POST \
    -F base=@data/openapi-test1.yaml \
    -F revision=@data/openapi-test3.yaml \
    http://localhost:8080/diff
```
