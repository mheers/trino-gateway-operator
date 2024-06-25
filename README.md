# Trino-Gateway-Operator

POC for a Trino Gateway Operator. Manages the cluster backends in a [Trino Gateway](https://github.com/trinodb/trino-gateway).

## Dev

### Test / Use

```bash
k create -f crd.yaml
go run main.go &
k create -f cr-example.yaml
```

## TODO:

- [ ] delete
