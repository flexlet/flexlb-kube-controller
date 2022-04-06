FROM golang:1.17 as builder

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY handlers/ handlers/
COPY utils/ utils/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o flexlb-kube-controller main.go

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/flexlb-kube-controller .
USER 65532:65532

ENTRYPOINT ["/flexlb-kube-controller"]
