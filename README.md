# wireguard-kubernetes

## Deploying in a kind environment

Deploy with:
~~~
scripts/kind.sh
~~~

Tear down with:
~~~
scripts/kind.sh --delete
~~~

## Running e2e tests against a kind environment

Run e2e tests with:
~~~
scripts/e2e.sh
~~~

## Running unit tests locally

Run unit tests with:
~~~
make -C controller test
~~~
