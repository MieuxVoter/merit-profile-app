FROM dhi.io/golang:1.26-debian13-dev AS builder
WORKDIR /build
COPY . /build
RUN touch .env.local
RUN make clean
RUN --mount=type=cache,sharing=locked,target=/go/pkg/mod \
    --mount=type=cache,sharing=locked,target=/root/.cache/go-build \
    make build


FROM dhi.io/debian-base:trixie-dev AS provider
RUN --mount=type=cache,sharing=locked,target=/var/lib/apt/lists \
    --mount=type=cache,sharing=locked,target=/var/cache/apt \
    ls -FlAh /etc/apt/apt.conf.d/ \
    && rm -f /etc/apt/apt.conf.d/docker-clean \
    && apt-get update \
    && apt-get --yes --no-install-recommends install \
        make
    # Note: We don't actually *need* make ; but eventually we will need some packages.


FROM dhi.io/debian-base:trixie AS runner
WORKDIR /app
# Example of how we could copy stuff from the provider image
#COPY --from=provider /etc/fonts/conf.d/70-fonts-noto-cjk.conf /etc/fonts/conf.d
# And now we copy the app itself and its config
COPY --from=builder /build/build/mpa ./
COPY --from=builder /build/.env ./
COPY --from=builder /build/.env.local ./
COPY --from=builder /build/public ./public/
CMD ["/app/mpa"]
