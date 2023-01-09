FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY flexlb-kube-controller .
USER 65532:65532

ENTRYPOINT ["/flexlb-kube-controller"]
