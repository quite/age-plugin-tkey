FROM docker.io/silkeh/clang:17-bookworm as base

# golang-1.20 from backports is needed for crypto/ecdh

# TODO should we do apt-get upgrade to get latest patch release of
# clang? (still from apt.llvm.org ofcourse!)
RUN sed -E -i "s/^Suites: ([a-z]+) \1-updates$/Suites: \1 \1-updates \1-backports/" \
        /etc/apt/sources.list.d/debian.sources \
    && apt-get update -y \
    && DEBIAN_FRONTEND=noninteractive \
       apt-get install -y --no-install-recommends \
               git \
               golang-1.20 \
    && rm -rf /var/lib/apt/lists/*

ENV PATH="$PATH:/usr/lib/go-1.20/bin"
