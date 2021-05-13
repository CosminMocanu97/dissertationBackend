A barebone Golang template for new projects.
This is based on https://github.com/cpl/goose.

- Build docker image: `docker build .`
    - It may not run on the empty template as there is no go.sum

- The `docker-compose.yml` builds the app and an instance of Postgres and it connects them.
