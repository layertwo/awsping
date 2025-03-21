FROM golang:1.24-bullseye as build
COPY . /build
WORKDIR /build
RUN make

FROM gcr.io/distroless/base
COPY --from=build /build/awsping /

ENTRYPOINT ["/awsping"]
