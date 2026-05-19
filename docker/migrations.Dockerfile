FROM golang:1.26.3-bookworm

RUN go install github.com/pressly/goose/v3/cmd/goose@latest

WORKDIR /migrations
COPY migrations/ /migrations/

ENV PATH="/go/bin:$PATH"

CMD ["goose", "up"]
