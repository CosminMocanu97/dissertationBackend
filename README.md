A barebone Golang backend implementation for the dissertation thesis
"Web security for a document management application"

- Build docker image: `docker build .`
    - It may not run on the empty template as there is no go.sum

- The `docker-compose.yml` builds an instance of Postgres and it connects to it.