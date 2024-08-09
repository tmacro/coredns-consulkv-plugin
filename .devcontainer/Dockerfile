FROM golang:1.22

RUN apt-get update && apt-get install -y git

# Install CoreDNS
RUN go install github.com/coredns/coredns@latest

# Create a non-root user
ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=$USER_UID

RUN groupadd --gid $USER_GID $USERNAME \
    && useradd -s /bin/bash --uid $USER_UID --gid $USER_GID -m $USERNAME

# Set up sudo
RUN apt-get install -y sudo \
    && echo $USERNAME ALL=\(root\) NOPASSWD:ALL > /etc/sudoers.d/$USERNAME \
    && chmod 0440 /etc/sudoers.d/$USERNAME

USER $USERNAME

# Set up Go environment
ENV GOPATH /go
ENV PATH $PATH:/go/bin

WORKDIR /workspace