This document is a collection of notes on how to develop prenv.

## Building prenv container image

```bash
docker build -t myrepo/prenv:dev .

docker run -ti --rm myrepo/prenv:dev prenv --help
```
